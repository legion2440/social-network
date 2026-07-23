package ws

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/service"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 16 << 10
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin:     IsSameOrigin,
}

type Client struct {
	id          string
	userID      int64
	displayName string
	sessionKey  SessionKey
	expiresAt   time.Time
	conn        *websocket.Conn
	send        chan []byte
	done        chan struct{}
	hub         *Hub

	active  bool
	revoked bool
	ready   bool

	closeOnce      sync.Once
	unregisterOnce sync.Once
}

type connectionRuntime struct {
	client   *Client
	chats    *service.ChatService
	rawToken string
}

type chatSendRequest struct {
	Type            string         `json:"type"`
	ClientMessageID string         `json:"client_message_id"`
	Chat            domain.ChatRef `json:"chat"`
	Text            string         `json:"text"`
}

type typingRequest struct {
	Type string         `json:"type"`
	Chat domain.ChatRef `json:"chat"`
}

type wsUserSummary struct {
	ID          int64  `json:"id"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
}

type wsMessage struct {
	ID              int64          `json:"id"`
	ClientMessageID string         `json:"client_message_id"`
	Chat            domain.ChatRef `json:"chat"`
	Sender          wsUserSummary  `json:"sender"`
	Body            string         `json:"body"`
	CreatedAt       time.Time      `json:"created_at"`
}

type chatMessageEnvelope struct {
	Type            string     `json:"type"`
	ClientMessageID string     `json:"client_message_id"`
	Message         *wsMessage `json:"message"`
}

type chatErrorEnvelope struct {
	Type            string `json:"type"`
	ClientMessageID string `json:"client_message_id,omitempty"`
	Code            string `json:"code"`
	Message         string `json:"message"`
}

type chatUnreadEnvelope struct {
	Type                 string         `json:"type"`
	Chat                 domain.ChatRef `json:"chat"`
	ChatUnreadCount      int64          `json:"chat_unread_count"`
	UnreadCount          int64          `json:"unread_count"`
	Revision             int64          `json:"revision"`
	ReadThroughMessageID *int64         `json:"read_through_message_id"`
}

func IsSameOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return false
	}
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	if !strings.EqualFold(parsed.Scheme, "http") && !strings.EqualFold(parsed.Scheme, "https") {
		return false
	}
	if parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	return strings.EqualFold(parsed.Host, r.Host)
}

func Serve(
	w http.ResponseWriter,
	r *http.Request,
	hub *Hub,
	chats *service.ChatService,
	userID int64,
	rawSessionToken string,
	expiresAt time.Time,
	displayName string,
	presenceGeneration PresenceSyncGeneration,
	peerIDs []int64,
) error {
	if hub == nil || chats == nil || userID <= 0 || strings.TrimSpace(rawSessionToken) == "" || expiresAt.IsZero() {
		return ErrSessionUnavailable
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}
	client := &Client{
		id: uuid.NewString(), userID: userID, displayName: strings.TrimSpace(displayName),
		sessionKey: HashSessionToken(rawSessionToken), expiresAt: expiresAt,
		conn: conn, send: make(chan []byte, ClientQueueSize), done: make(chan struct{}), hub: hub,
	}
	if err := hub.Register(client, presenceGeneration, peerIDs); err != nil {
		code := websocket.CloseTryAgainLater
		reason := "realtime unavailable"
		if errors.Is(err, ErrConnectionLimit) {
			code = websocket.ClosePolicyViolation
			reason = "connection limit reached"
		}
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason), time.Now().Add(writeWait))
		_ = conn.Close()
		return nil
	}

	runtime := &connectionRuntime{client: client, chats: chats, rawToken: rawSessionToken}
	go client.writePump()
	runtime.readPump()
	return nil
}

func (runtime *connectionRuntime) readPump() {
	client := runtime.client
	defer client.unregister()
	client.conn.SetReadLimit(maxMessageSize)
	_ = client.conn.SetReadDeadline(time.Now().Add(pongWait))
	client.conn.SetPongHandler(func(string) error {
		return client.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		messageType, raw, err := client.conn.ReadMessage()
		if err != nil {
			return
		}
		if messageType != websocket.TextMessage {
			runtime.queueError("", "invalid_event", "text JSON event required")
			continue
		}
		runtime.handleIncoming(raw)
	}
}

func (runtime *connectionRuntime) handleIncoming(raw []byte) {
	var envelope struct {
		Type string `json:"type"`
	}
	if err := decodeStrictJSON(raw, &envelope, false); err != nil || strings.TrimSpace(envelope.Type) == "" {
		runtime.queueError("", "invalid_event", "invalid event")
		return
	}
	switch envelope.Type {
	case "chat:send":
		var request chatSendRequest
		if err := decodeStrictJSON(raw, &request, true); err != nil {
			runtime.queueError(extractClientMessageID(raw), "invalid_event", "invalid event")
			return
		}
		runtime.handleChatSend(request)
	case "typing:start", "typing:heartbeat", "typing:stop":
		var request typingRequest
		if err := decodeStrictJSON(raw, &request, true); err != nil {
			runtime.queueError("", "invalid_event", "invalid event")
			return
		}
		runtime.handleTyping(request)
	default:
		runtime.queueError("", "invalid_event", "unknown event type")
	}
}

func decodeStrictJSON(raw []byte, target any, disallowUnknown bool) error {
	if err := rejectDuplicateJSONFields(raw); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if disallowUnknown {
		decoder.DisallowUnknownFields()
	}
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return errors.New("multiple JSON values")
	} else if !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func rejectDuplicateJSONFields(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := consumeUniqueJSONValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func consumeUniqueJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("invalid JSON object key")
			}
			if _, duplicate := seen[key]; duplicate {
				return errors.New("duplicate JSON object key")
			}
			seen[key] = struct{}{}
			if err := consumeUniqueJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil || closing != json.Delim('}') {
			return errors.New("invalid JSON object")
		}
	case '[':
		for decoder.More() {
			if err := consumeUniqueJSONValue(decoder); err != nil {
				return err
			}
		}
		closing, err := decoder.Token()
		if err != nil || closing != json.Delim(']') {
			return errors.New("invalid JSON array")
		}
	default:
		return errors.New("unexpected JSON delimiter")
	}
	return nil
}

func extractClientMessageID(raw []byte) string {
	var value struct {
		ClientMessageID string `json:"client_message_id"`
	}
	if json.Unmarshal(raw, &value) != nil {
		return ""
	}
	if _, err := uuid.Parse(strings.TrimSpace(value.ClientMessageID)); err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(value.ClientMessageID))
}

func (runtime *connectionRuntime) handleChatSend(request chatSendRequest) {
	leaseID, operationContext, err := runtime.client.hub.AcquireSessionOperation(runtime.client.sessionKey, runtime.client.id)
	if err != nil {
		return
	}
	go func() {
		var delivery *Delivery
		defer func() {
			if recovered := recover(); recovered != nil {
				delivery = &Delivery{AckPayload: mustMarshalChatError(request.ClientMessageID, "internal", "internal server error")}
			}
			_ = runtime.client.hub.CompleteSessionOperation(leaseID, delivery)
		}()

		result, sendErr := runtime.chats.Send(operationContext, runtime.client.userID, runtime.rawToken, service.ChatSendInput{
			ClientMessageID: request.ClientMessageID, Chat: request.Chat, Body: request.Text,
		})
		if sendErr != nil {
			delivery = &Delivery{AckPayload: mustMarshalChatError(request.ClientMessageID, chatErrorCode(sendErr), chatErrorMessage(sendErr))}
			return
		}
		ackPayload, marshalErr := marshalChatMessage(result.Message, result.Message.Chat)
		if marshalErr != nil {
			delivery = &Delivery{AckPayload: mustMarshalChatError(request.ClientMessageID, "internal", "internal server error")}
			return
		}
		senderBroadcast := ackPayload
		recipientChat := result.Message.Chat
		if recipientChat.Kind == domain.ChatDirect {
			recipientChat.TargetID = runtime.client.userID
		}
		recipientBroadcast, marshalErr := marshalChatMessage(result.Message, recipientChat)
		if marshalErr != nil {
			delivery = &Delivery{AckPayload: mustMarshalChatError(request.ClientMessageID, "internal", "internal server error")}
			return
		}
		recipientPayloads := make(map[int64][][]byte)
		for userID, state := range result.UnreadEffects.StatesByUser {
			payload, err := marshalChatUnread(state)
			if err != nil {
				delivery = &Delivery{AckPayload: mustMarshalChatError(request.ClientMessageID, "internal", "internal server error")}
				return
			}
			recipientPayloads[userID] = [][]byte{payload}
		}
		delivery = &Delivery{
			Created: result.Created, Chat: result.Message.Chat, RecipientUserIDs: result.RecipientUserIDs,
			RecipientUserPayloads: recipientPayloads,
			AckPayload:            ackPayload, SenderBroadcastPayload: senderBroadcast, RecipientBroadcastPayload: recipientBroadcast,
		}
		_ = runtime.client.hub.QueueTyping(runtime.client.id, request.Chat, nil, TypingStop)
	}()
}

func marshalChatUnread(state *domain.ChatUnreadState) ([]byte, error) {
	if state == nil || !state.Chat.Kind.Valid() || state.Chat.TargetID <= 0 {
		return nil, errors.New("chat unread state is missing")
	}
	return json.Marshal(chatUnreadEnvelope{
		Type: "chat:unread", Chat: state.Chat,
		ChatUnreadCount: state.ChatUnreadCount, UnreadCount: state.UnreadCount,
		Revision: state.Revision, ReadThroughMessageID: state.ReadThroughMessageID,
	})
}

func (runtime *connectionRuntime) handleTyping(request typingRequest) {
	kind := TypingStop
	switch request.Type {
	case "typing:start":
		kind = TypingStart
	case "typing:heartbeat":
		kind = TypingHeartbeat
	}
	if kind == TypingStop {
		_ = runtime.client.hub.QueueTyping(runtime.client.id, request.Chat, nil, kind)
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		recipients, err := runtime.chats.AuthorizeTyping(ctx, runtime.client.userID, runtime.rawToken, request.Chat)
		if err != nil {
			runtime.queueError("", chatErrorCode(err), chatErrorMessage(err))
			return
		}
		_ = runtime.client.hub.QueueTyping(runtime.client.id, request.Chat, recipients, kind)
	}()
}

func marshalChatMessage(message *domain.ChatMessage, chat domain.ChatRef) ([]byte, error) {
	if message == nil || message.Sender == nil {
		return nil, errors.New("message sender is missing")
	}
	displayName := strings.TrimSpace(message.Sender.FirstName + " " + message.Sender.LastName)
	if message.Sender.Nickname != nil && strings.TrimSpace(*message.Sender.Nickname) != "" {
		displayName = strings.TrimSpace(*message.Sender.Nickname)
	}
	return json.Marshal(chatMessageEnvelope{
		Type: "chat:message", ClientMessageID: message.ClientMessageID,
		Message: &wsMessage{
			ID: message.ID, ClientMessageID: message.ClientMessageID, Chat: chat,
			Sender: wsUserSummary{ID: message.SenderUserID, DisplayName: displayName, AvatarURL: domain.NeutralAvatarPlaceholderURL},
			Body:   message.Body, CreatedAt: message.CreatedAt,
		},
	})
}

func mustMarshalChatError(clientMessageID, code, message string) []byte {
	payload, _ := json.Marshal(chatErrorEnvelope{
		Type: "chat:error", ClientMessageID: strings.TrimSpace(clientMessageID), Code: code, Message: message,
	})
	return payload
}

func (runtime *connectionRuntime) queueError(clientMessageID, code, message string) {
	payload := mustMarshalChatError(clientMessageID, code, message)
	_ = runtime.client.hub.SendClient(runtime.client.id, payload)
}

func chatErrorCode(err error) string {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		return "invalid_input"
	case errors.Is(err, service.ErrUnauthorized):
		return "forbidden"
	case errors.Is(err, service.ErrForbidden):
		return "forbidden"
	case errors.Is(err, service.ErrNotFound):
		return "not_found"
	case errors.Is(err, service.ErrConflict):
		return "conflict"
	default:
		return "internal"
	}
}

func chatErrorMessage(err error) string {
	switch chatErrorCode(err) {
	case "invalid_input":
		return "invalid input"
	case "forbidden":
		return "forbidden"
	case "not_found":
		return "not found"
	case "conflict":
		return "conflict"
	default:
		return "internal server error"
	}
}

func (client *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case payload := <-client.send:
			if err := client.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				client.unregister()
				return
			}
			if err := client.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				client.unregister()
				return
			}
		case <-ticker.C:
			_ = client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				client.unregister()
				return
			}
		case <-client.done:
			return
		}
	}
}

func (client *Client) unregister() {
	if client == nil {
		return
	}
	client.unregisterOnce.Do(func() {
		if client.hub == nil || client.hub.currentState() == hubStopped {
			client.close()
			return
		}
		client.hub.Unregister(client.id)
	})
}

func (client *Client) close() {
	if client == nil {
		return
	}
	client.closeOnce.Do(func() {
		close(client.done)
		if client.conn != nil {
			_ = client.conn.Close()
		}
	})
}
