package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"social-network/backend/internal/repo"
)

type TransactionManager struct {
	db *sql.DB
}

func NewTransactionManager(db *sql.DB) *TransactionManager {
	return &TransactionManager{db: db}
}

func (m *TransactionManager) WithinTransaction(ctx context.Context, fn func(repo.TransactionRepositories) error) error {
	if m == nil || m.db == nil || fn == nil {
		return errors.New("transaction manager is not configured")
	}
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	finished := false
	defer func() {
		if !finished {
			_ = tx.Rollback()
		}
	}()

	repositories := &transactionRepositories{
		users:    &UserRepo{db: tx},
		sessions: &SessionRepo{db: tx},
		media:    &MediaRepo{db: tx},
	}
	if err := fn(repositories); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			return fmt.Errorf("%w (rollback: %v)", err, rollbackErr)
		}
		finished = true
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	finished = true
	return nil
}

type transactionRepositories struct {
	users    *UserRepo
	sessions *SessionRepo
	media    *MediaRepo
}

func (r *transactionRepositories) Users() repo.UserRepo {
	return r.users
}

func (r *transactionRepositories) Sessions() repo.SessionRepo {
	return r.sessions
}

func (r *transactionRepositories) Media() repo.MediaRepo {
	return r.media
}
