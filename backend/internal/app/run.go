package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"social-network/backend/internal/config"
	httpserver "social-network/backend/internal/http"
	"social-network/backend/internal/platform/clock"
	"social-network/backend/internal/platform/id"
	realtimews "social-network/backend/internal/realtime/ws"
	"social-network/backend/internal/repo/sqlite"
	"social-network/backend/internal/service"
)

type runtime struct {
	db      *sql.DB
	server  *http.Server
	handler *httpserver.Handler
	hub     *realtimews.Hub
}

func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()
	return runWithContext(ctx, cfg)
}

func runWithContext(ctx context.Context, cfg config.Config) error {
	runtime, err := bootstrap(ctx, cfg)
	if err != nil {
		return err
	}
	defer runtime.close()

	listener, err := net.Listen("tcp", cfg.HTTPAddr)
	if err != nil {
		return fmt.Errorf("http listen: %w", err)
	}
	runtime.server.Addr = listener.Addr().String()

	serverErr := make(chan error, 1)
	go func() {
		log.Printf("http server listening on %s", runtime.server.Addr)
		err := runtime.server.Serve(listener)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("http server: %w", err)
			return
		}
		serverErr <- nil
	}()

	select {
	case err := <-serverErr:
		if err == nil {
			return errors.New("http server stopped unexpectedly")
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		runtime.handler.CloseAdmission()
		if err := runtime.hub.BeginDrain(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("realtime shutdown: %w", err)
		}
		if err := runtime.server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("http shutdown: %w", err)
		}
		if err := <-serverErr; err != nil {
			return err
		}
		select {
		case <-runtime.hub.Done():
		case <-shutdownCtx.Done():
			return fmt.Errorf("realtime shutdown: %w", shutdownCtx.Err())
		}
		return nil
	}
}

func bootstrap(ctx context.Context, cfg config.Config) (*runtime, error) {
	db, err := sqlite.Open(ctx, cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("db open: %w", err)
	}

	appClock := clock.RealClock{}
	ids := id.UUIDGenerator{}
	users := sqlite.NewUserRepo(db)
	sessions := service.NewSessionService(sqlite.NewSessionRepo(db), appClock, ids, cfg.SessionTTL)
	media, err := service.NewMediaService(sqlite.NewMediaRepo(db), appClock, ids, cfg.UploadDir)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("media init: %w", err)
	}
	avatarStager, err := service.NewMediaStager(ids, cfg.UploadDir, service.MaxAvatarBytes)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("avatar storage init: %w", err)
	}
	transactions := sqlite.NewTransactionManager(db)
	postStager, err := service.NewMediaStager(ids, cfg.UploadDir, service.MaxMediaBytes)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("post media storage init: %w", err)
	}
	auth := service.NewAuthService(
		users,
		transactions,
		sessions,
		service.BcryptHasher{},
		appClock,
		avatarStager,
	)
	profile := service.NewProfileService(transactions, appClock, avatarStager, log.Default())
	follows := service.NewFollowService(users, sqlite.NewFollowRepo(db), transactions, appClock)
	userProfiles := service.NewUserService(transactions)
	avatarDelivery := service.NewAvatarDeliveryService(transactions, cfg.UploadDir)
	posts := service.NewPostService(transactions, appClock, postStager)
	postMedia := service.NewPostMediaDeliveryService(transactions, cfg.UploadDir)
	comments := service.NewCommentService(transactions, appClock)
	groups := service.NewGroupService(transactions, appClock)
	groupEvents := service.NewGroupEventService(transactions, appClock)
	notifications := service.NewNotificationService(transactions, appClock)
	chats := service.NewChatService(transactions, appClock)
	hub := realtimews.NewHub(log.Default())
	go hub.Run()
	handler := httpserver.NewHandler(
		db,
		sessions,
		media,
		auth,
		profile,
		follows,
		userProfiles,
		avatarDelivery,
		posts,
		postMedia,
		comments,
		groups,
		groupEvents,
		notifications,
		chats,
		httpserver.NewCookieSessionTokenExtractor(config.SessionCookieName),
		cfg.CookieSecure,
		cfg.FrontendDir,
		log.Default(),
	)
	handler.SetRealtimeHub(hub)

	return &runtime{
		db: db, handler: handler, hub: hub,
		server: &http.Server{
			Handler:           handler.Routes(),
			ReadTimeout:       15 * time.Second,
			ReadHeaderTimeout: 5 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       60 * time.Second,
			MaxHeaderBytes:    1 << 20,
		},
	}, nil
}

func (r *runtime) close() {
	if r == nil {
		return
	}
	if r.handler != nil {
		r.handler.CloseAdmission()
	}
	if r.hub != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		_ = r.hub.BeginDrain(ctx)
		cancel()
		select {
		case <-r.hub.Done():
		case <-time.After(time.Second):
		}
	}
	if r.db != nil {
		_ = r.db.Close()
	}
}
