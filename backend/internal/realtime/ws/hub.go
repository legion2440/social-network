package ws

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"social-network/backend/internal/domain"
)

const (
	MaxConnectionsPerUser = 8
	ClientQueueSize       = 64
)

var (
	ErrHubDraining        = errors.New("websocket hub is draining")
	ErrHubStopped         = errors.New("websocket hub is stopped")
	ErrConnectionLimit    = errors.New("websocket connection limit reached")
	ErrInvalidOrigin      = errors.New("invalid websocket origin")
	ErrSessionUnavailable = errors.New("websocket session is unavailable")
)

type SessionKey [sha256.Size]byte

func HashSessionToken(rawToken string) SessionKey {
	return sha256.Sum256([]byte(rawToken))
}

type hubState uint8

const (
	hubRunning hubState = iota + 1
	hubDraining
	hubStopped
)

type operationLease struct {
	id             uint64
	sessionKey     SessionKey
	originClientID string
	senderUserID   int64
	ctx            context.Context
	cancel         context.CancelFunc
}

type sessionState struct {
	key       SessionKey
	userID    int64
	expiresAt time.Time
	revoked   bool
	clients   map[string]*Client
	leases    map[uint64]*operationLease
}

type Delivery struct {
	Created                   bool
	Chat                      domain.ChatRef
	RecipientUserIDs          []int64
	AckPayload                []byte
	SenderBroadcastPayload    []byte
	RecipientBroadcastPayload []byte
}

type registerCommand struct {
	client  *Client
	peerIDs []int64
	result  chan error
}

type unregisterCommand struct{ clientID string }

type acquireCommand struct {
	sessionKey SessionKey
	clientID   string
	result     chan acquireResult
}

type acquireResult struct {
	leaseID uint64
	ctx     context.Context
	err     error
}

type completeCommand struct {
	leaseID  uint64
	delivery *Delivery
	result   chan error
}

type revokeCommand struct {
	sessionKey SessionKey
	result     chan error
}

type relationshipCommand struct {
	firstUserID  int64
	secondUserID int64
	eligible     bool
	result       chan struct{}
}

type groupAccessCommand struct {
	groupID int64
	userID  int64
	active  bool
	result  chan struct{}
}

type clientPayloadCommand struct {
	clientID string
	payload  []byte
	result   chan error
}

type beginDrainCommand struct {
	deadline <-chan time.Time
	result   chan error
}

type forceStopCommand struct{}

type Hub struct {
	commands chan any
	done     chan struct{}
	logger   *log.Logger
	now      func() time.Time

	stateMu sync.RWMutex
	state   hubState

	sessions       map[SessionKey]*sessionState
	clients        map[string]*Client
	clientsByUser  map[int64]map[string]*Client
	leases         map[uint64]*operationLease
	nextLeaseID    uint64
	presencePeers  map[int64]map[int64]struct{}
	blockedGroups  map[int64]map[int64]struct{}
	typingByTarget map[typingTarget]map[string]*typingEntry
	typingByClient map[string]map[typingTarget]struct{}
	doneOnce       sync.Once
}

func NewHub(logger *log.Logger) *Hub {
	return NewHubWithNow(logger, func() time.Time { return time.Now().UTC() })
}

func NewHubWithNow(logger *log.Logger, now func() time.Time) *Hub {
	if logger == nil {
		logger = log.Default()
	}
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Hub{
		commands:       make(chan any),
		done:           make(chan struct{}),
		logger:         logger,
		now:            now,
		state:          hubRunning,
		sessions:       make(map[SessionKey]*sessionState),
		clients:        make(map[string]*Client),
		clientsByUser:  make(map[int64]map[string]*Client),
		leases:         make(map[uint64]*operationLease),
		presencePeers:  make(map[int64]map[int64]struct{}),
		blockedGroups:  make(map[int64]map[int64]struct{}),
		typingByTarget: make(map[typingTarget]map[string]*typingEntry),
		typingByClient: make(map[string]map[typingTarget]struct{}),
	}
}

func (h *Hub) Run() {
	if h == nil {
		return
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case command := <-h.commands:
			h.handleCommand(command)
		case <-ticker.C:
			h.expire(h.now())
		}
		if h.currentState() == hubStopped {
			return
		}
	}
}

func (h *Hub) Done() <-chan struct{} {
	if h == nil {
		closed := make(chan struct{})
		close(closed)
		return closed
	}
	return h.done
}

func (h *Hub) currentState() hubState {
	h.stateMu.RLock()
	defer h.stateMu.RUnlock()
	return h.state
}

func (h *Hub) setState(state hubState) {
	h.stateMu.Lock()
	h.state = state
	h.stateMu.Unlock()
}

func (h *Hub) submit(command any) error {
	if h == nil {
		return ErrHubStopped
	}
	if h.currentState() == hubStopped {
		return ErrHubStopped
	}
	select {
	case h.commands <- command:
		return nil
	case <-h.done:
		return ErrHubStopped
	}
}

func (h *Hub) Register(client *Client, peerIDs []int64) error {
	if client == nil {
		return ErrSessionUnavailable
	}
	result := make(chan error, 1)
	if err := h.submit(registerCommand{client: client, peerIDs: peerIDs, result: result}); err != nil {
		return err
	}
	select {
	case err := <-result:
		return err
	case <-h.done:
		return ErrHubStopped
	}
}

func (h *Hub) Unregister(clientID string) {
	if h == nil || clientID == "" || h.currentState() == hubStopped {
		return
	}
	_ = h.submit(unregisterCommand{clientID: clientID})
}

func (h *Hub) AcquireSessionOperation(sessionKey SessionKey, originClientID string) (uint64, context.Context, error) {
	if h == nil || originClientID == "" {
		return 0, nil, ErrSessionUnavailable
	}
	state := h.currentState()
	if state == hubDraining {
		return 0, nil, ErrHubDraining
	}
	if state == hubStopped {
		return 0, nil, ErrHubStopped
	}
	result := make(chan acquireResult, 1)
	if err := h.submit(acquireCommand{sessionKey: sessionKey, clientID: originClientID, result: result}); err != nil {
		return 0, nil, err
	}
	select {
	case value := <-result:
		return value.leaseID, value.ctx, value.err
	case <-h.done:
		return 0, nil, ErrHubStopped
	}
}

func (h *Hub) CompleteSessionOperation(leaseID uint64, delivery *Delivery) error {
	if h == nil || h.currentState() == hubStopped {
		return ErrHubStopped
	}
	result := make(chan error, 1)
	if err := h.submit(completeCommand{leaseID: leaseID, delivery: delivery, result: result}); err != nil {
		return err
	}
	select {
	case err := <-result:
		return err
	case <-h.done:
		return ErrHubStopped
	}
}

func (h *Hub) RevokeSession(sessionKey SessionKey) error {
	if h == nil || h.currentState() == hubStopped {
		return ErrHubStopped
	}
	result := make(chan error, 1)
	if err := h.submit(revokeCommand{sessionKey: sessionKey, result: result}); err != nil {
		return err
	}
	select {
	case err := <-result:
		return err
	case <-h.done:
		return ErrHubStopped
	}
}

func (h *Hub) RelationshipChanged(firstUserID, secondUserID int64, eligible bool) {
	if firstUserID <= 0 || secondUserID <= 0 || firstUserID == secondUserID || h == nil || h.currentState() != hubRunning {
		return
	}
	result := make(chan struct{}, 1)
	if h.submit(relationshipCommand{
		firstUserID: firstUserID, secondUserID: secondUserID, eligible: eligible, result: result,
	}) != nil {
		return
	}
	select {
	case <-result:
	case <-h.done:
	}
}

func (h *Hub) GroupAccessChanged(groupID, userID int64, active bool) {
	if groupID <= 0 || userID <= 0 || h == nil || h.currentState() != hubRunning {
		return
	}
	result := make(chan struct{}, 1)
	if h.submit(groupAccessCommand{groupID: groupID, userID: userID, active: active, result: result}) != nil {
		return
	}
	select {
	case <-result:
	case <-h.done:
	}
}

func (h *Hub) SendClient(clientID string, payload []byte) error {
	if h == nil || clientID == "" || len(payload) == 0 {
		return ErrSessionUnavailable
	}
	result := make(chan error, 1)
	if err := h.submit(clientPayloadCommand{clientID: clientID, payload: payload, result: result}); err != nil {
		return err
	}
	select {
	case err := <-result:
		return err
	case <-h.done:
		return ErrHubStopped
	}
}

func (h *Hub) BeginDrain(ctx context.Context) error {
	if h == nil {
		return ErrHubStopped
	}
	state := h.currentState()
	if state == hubStopped {
		return nil
	}
	if state == hubDraining {
		return nil
	}
	result := make(chan error, 1)
	var deadline <-chan time.Time
	if value, ok := ctx.Deadline(); ok {
		timer := time.NewTimer(time.Until(value))
		deadline = timer.C
	}
	if err := h.submit(beginDrainCommand{deadline: deadline, result: result}); err != nil {
		return err
	}
	select {
	case err := <-result:
		return err
	case <-h.done:
		return nil
	case <-ctx.Done():
		_ = h.submit(forceStopCommand{})
		return ctx.Err()
	}
}

func (h *Hub) handleCommand(command any) {
	switch value := command.(type) {
	case registerCommand:
		value.result <- h.register(value.client, value.peerIDs)
	case unregisterCommand:
		h.deactivateClient(value.clientID, deactivateDisconnect)
	case acquireCommand:
		value.result <- h.acquire(value)
	case completeCommand:
		value.result <- h.complete(value.leaseID, value.delivery)
	case revokeCommand:
		h.revokeSession(value.sessionKey, deactivateRevoked)
		value.result <- nil
	case relationshipCommand:
		h.relationshipChanged(value)
		value.result <- struct{}{}
	case groupAccessCommand:
		h.groupAccessChanged(value)
		value.result <- struct{}{}
	case clientPayloadCommand:
		client := h.activeClient(value.clientID)
		if client == nil {
			value.result <- ErrSessionUnavailable
		} else if !h.enqueue(client, value.payload) {
			h.deactivateClient(client.id, deactivateSlow)
			value.result <- ErrSessionUnavailable
		} else {
			value.result <- nil
		}
	case typingCommand:
		h.handleTyping(value, h.now())
	case beginDrainCommand:
		h.beginDrain(value)
	case forceStopCommand:
		h.forceStop()
	}
	if h.currentState() == hubDraining && len(h.leases) == 0 {
		h.finishStop(false)
	}
}

func (h *Hub) register(client *Client, peerIDs []int64) error {
	if h.currentState() != hubRunning {
		return ErrHubDraining
	}
	if client == nil || client.id == "" || client.userID <= 0 || !client.expiresAt.After(h.now()) {
		return ErrSessionUnavailable
	}
	if h.activeClientCount(client.userID) >= MaxConnectionsPerUser {
		return ErrConnectionLimit
	}
	session := h.sessions[client.sessionKey]
	if session == nil {
		session = &sessionState{
			key: client.sessionKey, userID: client.userID, expiresAt: client.expiresAt,
			clients: make(map[string]*Client), leases: make(map[uint64]*operationLease),
		}
		h.sessions[client.sessionKey] = session
	}
	if session.revoked || session.userID != client.userID || !session.expiresAt.Equal(client.expiresAt) {
		return ErrSessionUnavailable
	}
	wasOnline := h.userOnline(client.userID)
	client.active = true
	session.clients[client.id] = client
	h.clients[client.id] = client
	userClients := h.clientsByUser[client.userID]
	if userClients == nil {
		userClients = make(map[string]*Client)
		h.clientsByUser[client.userID] = userClients
	}
	userClients[client.id] = client
	h.setPresencePeers(client.userID, peerIDs)
	if !wasOnline {
		h.notifyPresenceTransition(client.userID, true)
	}
	h.initializeClient(client)
	return nil
}

func (h *Hub) initializeClient(client *Client) {
	peerIDs := make([]int64, 0)
	for peerID := range h.presencePeers[client.userID] {
		if h.userOnline(peerID) {
			peerIDs = append(peerIDs, peerID)
		}
	}
	sort.Slice(peerIDs, func(i, j int) bool { return peerIDs[i] < peerIDs[j] })
	payload, err := marshalPresenceInit(peerIDs)
	if err != nil || !h.enqueue(client, payload) {
		h.deactivateClient(client.id, deactivateSlow)
		return
	}
	client.ready = true
}

func (h *Hub) acquire(command acquireCommand) acquireResult {
	if h.currentState() != hubRunning {
		return acquireResult{err: ErrHubDraining}
	}
	client := h.clients[command.clientID]
	session := h.sessions[command.sessionKey]
	now := h.now()
	if client == nil || session == nil || client.sessionKey != command.sessionKey || client.revoked || !client.active || session.revoked {
		return acquireResult{err: ErrSessionUnavailable}
	}
	if !session.expiresAt.After(now) {
		h.revokeSession(command.sessionKey, deactivateExpired)
		return acquireResult{err: ErrSessionUnavailable}
	}
	h.nextLeaseID++
	ctx, cancel := context.WithCancel(context.Background())
	lease := &operationLease{
		id: h.nextLeaseID, sessionKey: command.sessionKey, originClientID: command.clientID,
		senderUserID: client.userID, ctx: ctx, cancel: cancel,
	}
	h.leases[lease.id] = lease
	session.leases[lease.id] = lease
	return acquireResult{leaseID: lease.id, ctx: ctx}
}

func (h *Hub) complete(leaseID uint64, delivery *Delivery) error {
	lease := h.leases[leaseID]
	if lease == nil {
		if h.currentState() != hubStopped {
			h.logger.Printf("ws_operation_completion_unknown_lease")
		}
		return errors.New("unknown websocket operation lease")
	}
	delete(h.leases, leaseID)
	session := h.sessions[lease.sessionKey]
	if session != nil {
		delete(session.leases, leaseID)
	}
	lease.cancel()
	if session == nil {
		h.logger.Printf("ws_session_completion_missing_state")
		return errors.New("missing websocket session state")
	}
	if delivery != nil && !session.revoked && session.expiresAt.After(h.now()) && h.currentState() != hubStopped {
		h.routeDelivery(lease, delivery)
	}
	h.cleanupSession(session)
	return nil
}

func (h *Hub) routeDelivery(lease *operationLease, delivery *Delivery) {
	origin := h.activeClient(lease.originClientID)
	if delivery.Chat.Kind.Valid() && !h.chatDeliveryAllowed(lease.senderUserID, delivery.Chat) {
		return
	}
	if !delivery.Created {
		if origin != nil && len(delivery.AckPayload) > 0 && !h.enqueue(origin, delivery.AckPayload) {
			h.deactivateClient(origin.id, deactivateSlow)
		}
		return
	}
	if origin != nil && len(delivery.AckPayload) > 0 && !h.enqueue(origin, delivery.AckPayload) {
		h.deactivateClient(origin.id, deactivateSlow)
	}
	targets := make(map[string]*Client)
	for id, client := range h.clientsByUser[lease.senderUserID] {
		if id != lease.originClientID && h.clientCanReceive(client) {
			targets[id] = client
		}
	}
	recipients := make(map[int64]struct{}, len(delivery.RecipientUserIDs))
	for _, userID := range delivery.RecipientUserIDs {
		if userID > 0 && userID != lease.senderUserID && h.chatRecipientAllowed(userID, delivery.Chat) {
			recipients[userID] = struct{}{}
		}
	}
	for userID := range recipients {
		for id, client := range h.clientsByUser[userID] {
			if h.clientCanReceive(client) {
				targets[id] = client
			}
		}
	}
	for id, client := range targets {
		payload := delivery.RecipientBroadcastPayload
		if client.userID == lease.senderUserID {
			payload = delivery.SenderBroadcastPayload
		}
		if len(payload) > 0 && !h.enqueue(client, payload) {
			h.deactivateClient(id, deactivateSlow)
		}
	}
}

func (h *Hub) chatDeliveryAllowed(senderUserID int64, chat domain.ChatRef) bool {
	switch chat.Kind {
	case domain.ChatDirect:
		_, eligible := h.presencePeers[senderUserID][chat.TargetID]
		return eligible
	case domain.ChatGroup:
		return !h.groupAccessBlocked(chat.TargetID, senderUserID)
	default:
		return false
	}
}

func (h *Hub) chatRecipientAllowed(userID int64, chat domain.ChatRef) bool {
	if chat.Kind != domain.ChatGroup {
		return true
	}
	return !h.groupAccessBlocked(chat.TargetID, userID)
}

func (h *Hub) groupAccessBlocked(groupID, userID int64) bool {
	_, blocked := h.blockedGroups[groupID][userID]
	return blocked
}

func (h *Hub) activeClient(clientID string) *Client {
	client := h.clients[clientID]
	if !h.clientCanReceive(client) {
		return nil
	}
	return client
}

func (h *Hub) clientCanReceive(client *Client) bool {
	if client == nil || !client.active || client.revoked || !client.ready {
		return false
	}
	session := h.sessions[client.sessionKey]
	if session == nil || session.revoked {
		return false
	}
	if !session.expiresAt.After(h.now()) {
		h.revokeSession(client.sessionKey, deactivateExpired)
		return false
	}
	return true
}

func (h *Hub) enqueue(client *Client, payload []byte) bool {
	if client == nil || len(payload) == 0 {
		return false
	}
	select {
	case client.send <- payload:
		return true
	default:
		return false
	}
}

func (h *Hub) beginDrain(command beginDrainCommand) {
	if h.currentState() == hubStopped {
		command.result <- nil
		return
	}
	if h.currentState() == hubRunning {
		h.setState(hubDraining)
		for _, lease := range h.leases {
			lease.cancel()
		}
		if command.deadline != nil {
			go func(deadline <-chan time.Time) {
				select {
				case <-deadline:
					_ = h.submit(forceStopCommand{})
				case <-h.done:
				}
			}(command.deadline)
		}
	}
	command.result <- nil
}

func (h *Hub) forceStop() {
	if h.currentState() == hubStopped {
		return
	}
	for id, lease := range h.leases {
		lease.cancel()
		delete(h.leases, id)
	}
	for _, session := range h.sessions {
		clear(session.leases)
	}
	h.finishStop(true)
}

func (h *Hub) finishStop(_ bool) {
	if h.currentState() == hubStopped {
		return
	}
	for id := range h.clients {
		h.deactivateClient(id, deactivateShutdown)
	}
	clear(h.clients)
	clear(h.clientsByUser)
	clear(h.sessions)
	clear(h.leases)
	clear(h.presencePeers)
	clear(h.blockedGroups)
	clear(h.typingByTarget)
	clear(h.typingByClient)
	h.setState(hubStopped)
	h.doneOnce.Do(func() { close(h.done) })
}

func (h *Hub) expire(now time.Time) {
	for key, session := range h.sessions {
		if !session.revoked && !session.expiresAt.After(now) {
			h.revokeSession(key, deactivateExpired)
		}
	}
	h.expireTyping(now)
}

func (h *Hub) cleanupSession(session *sessionState) {
	if session == nil || len(session.clients) != 0 || len(session.leases) != 0 {
		return
	}
	delete(h.sessions, session.key)
}

func (h *Hub) activeClientCount(userID int64) int {
	count := 0
	for _, client := range h.clientsByUser[userID] {
		if client.active && !client.revoked {
			count++
		}
	}
	return count
}

func (h *Hub) userOnline(userID int64) bool {
	return h.activeClientCount(userID) > 0
}

type deactivateReason uint8

const (
	deactivateDisconnect deactivateReason = iota + 1
	deactivateSlow
	deactivateRevoked
	deactivateExpired
	deactivateShutdown
)

func (h *Hub) deactivateClient(clientID string, reason deactivateReason) {
	client := h.clients[clientID]
	if client == nil {
		return
	}
	wasOnline := h.userOnline(client.userID)
	wasActive := client.active
	client.active = false
	if reason == deactivateRevoked || reason == deactivateExpired {
		client.revoked = true
	}
	h.clearClientTyping(client, true)
	delete(h.clients, clientID)
	if userClients := h.clientsByUser[client.userID]; userClients != nil {
		delete(userClients, clientID)
		if len(userClients) == 0 {
			delete(h.clientsByUser, client.userID)
		}
	}
	if session := h.sessions[client.sessionKey]; session != nil {
		delete(session.clients, clientID)
		h.cleanupSession(session)
	}
	client.close()
	if wasActive && wasOnline && !h.userOnline(client.userID) && reason != deactivateShutdown {
		h.notifyPresenceTransition(client.userID, false)
	}
}

func (h *Hub) revokeSession(key SessionKey, reason deactivateReason) {
	session := h.sessions[key]
	if session == nil {
		return
	}
	session.revoked = true
	ids := make([]string, 0, len(session.clients))
	for id := range session.clients {
		ids = append(ids, id)
	}
	for _, id := range ids {
		h.deactivateClient(id, reason)
	}
	h.cleanupSession(session)
}

func (h *Hub) diagnosticState() string {
	return fmt.Sprintf("state=%d clients=%d leases=%d", h.currentState(), len(h.clients), len(h.leases))
}
