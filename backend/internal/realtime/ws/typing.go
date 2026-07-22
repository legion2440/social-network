package ws

import (
	"encoding/json"
	"strings"
	"time"

	"social-network/backend/internal/domain"
)

const typingTTL = 5 * time.Second

type TypingKind uint8

const (
	TypingStart TypingKind = iota + 1
	TypingHeartbeat
	TypingStop
)

type typingTarget struct {
	kind   domain.ChatKind
	first  int64
	second int64
}

type typingEntry struct {
	clientID     string
	userID       int64
	displayName  string
	expiresAt    time.Time
	recipientIDs map[int64]struct{}
}

type typingCommand struct {
	clientID         string
	chat             domain.ChatRef
	recipientUserIDs []int64
	kind             TypingKind
}

type typingEnvelope struct {
	Type   string         `json:"type"`
	Chat   domain.ChatRef `json:"chat"`
	User   typingUser     `json:"user"`
	Typing bool           `json:"typing"`
}

type typingUser struct {
	ID          int64  `json:"id"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
}

func (h *Hub) QueueTyping(clientID string, chat domain.ChatRef, recipientUserIDs []int64, kind TypingKind) error {
	if h == nil || h.currentState() != hubRunning {
		return ErrHubDraining
	}
	return h.submit(typingCommand{clientID: clientID, chat: chat, recipientUserIDs: recipientUserIDs, kind: kind})
}

func makeTypingTarget(actorUserID int64, chat domain.ChatRef) (typingTarget, bool) {
	if actorUserID <= 0 || !chat.Kind.Valid() || chat.TargetID <= 0 {
		return typingTarget{}, false
	}
	if chat.Kind == domain.ChatDirect {
		if chat.TargetID == actorUserID {
			return typingTarget{}, false
		}
		first, second := actorUserID, chat.TargetID
		if first > second {
			first, second = second, first
		}
		return typingTarget{kind: domain.ChatDirect, first: first, second: second}, true
	}
	return typingTarget{kind: domain.ChatGroup, first: chat.TargetID}, true
}

func (h *Hub) handleTyping(command typingCommand, now time.Time) {
	client := h.activeClient(command.clientID)
	if client == nil {
		return
	}
	target, ok := makeTypingTarget(client.userID, command.chat)
	if !ok {
		return
	}
	if command.kind == TypingStop {
		h.removeTypingClient(target, client.id, true)
		return
	}
	if !h.chatDeliveryAllowed(client.userID, command.chat) {
		h.removeTypingClient(target, client.id, true)
		return
	}
	bucket := h.typingByTarget[target]
	if bucket == nil {
		bucket = make(map[string]*typingEntry)
		h.typingByTarget[target] = bucket
	}
	wasUserTyping := typingUserActive(bucket, client.userID)
	entry := bucket[client.id]
	if entry == nil {
		entry = &typingEntry{clientID: client.id, userID: client.userID}
		bucket[client.id] = entry
		targets := h.typingByClient[client.id]
		if targets == nil {
			targets = make(map[typingTarget]struct{})
			h.typingByClient[client.id] = targets
		}
		targets[target] = struct{}{}
	}
	entry.displayName = strings.TrimSpace(client.displayName)
	entry.expiresAt = now.Add(typingTTL)
	entry.recipientIDs = normalizeRecipientSet(command.recipientUserIDs, client.userID)
	if command.chat.Kind == domain.ChatGroup {
		for userID := range entry.recipientIDs {
			if h.groupAccessBlocked(command.chat.TargetID, userID) {
				delete(entry.recipientIDs, userID)
			}
		}
	}
	if !wasUserTyping {
		h.notifyTyping(target, entry, true)
	}
}

func (h *Hub) removeTypingClient(target typingTarget, clientID string, notify bool) {
	bucket := h.typingByTarget[target]
	entry := bucket[clientID]
	if entry == nil {
		return
	}
	delete(bucket, clientID)
	if targets := h.typingByClient[clientID]; targets != nil {
		delete(targets, target)
		if len(targets) == 0 {
			delete(h.typingByClient, clientID)
		}
	}
	if len(bucket) == 0 {
		delete(h.typingByTarget, target)
	}
	if notify && !typingUserActive(bucket, entry.userID) {
		h.notifyTyping(target, entry, false)
	}
}

func (h *Hub) clearClientTyping(client *Client, notify bool) {
	if client == nil {
		return
	}
	targets := h.typingByClient[client.id]
	for target := range targets {
		h.removeTypingClient(target, client.id, notify)
	}
}

func (h *Hub) clearDirectTyping(firstUserID, secondUserID int64) {
	target, ok := makeTypingTarget(firstUserID, domain.ChatRef{Kind: domain.ChatDirect, TargetID: secondUserID})
	if !ok {
		return
	}
	bucket := h.typingByTarget[target]
	ids := make([]string, 0, len(bucket))
	for id := range bucket {
		ids = append(ids, id)
	}
	for _, id := range ids {
		h.removeTypingClient(target, id, true)
	}
}

func (h *Hub) groupAccessChanged(command groupAccessCommand) {
	if command.active {
		if blocked := h.blockedGroups[command.groupID]; blocked != nil {
			delete(blocked, command.userID)
			if len(blocked) == 0 {
				delete(h.blockedGroups, command.groupID)
			}
		}
		return
	}
	blocked := h.blockedGroups[command.groupID]
	if blocked == nil {
		blocked = make(map[int64]struct{})
		h.blockedGroups[command.groupID] = blocked
	}
	blocked[command.userID] = struct{}{}
	target := typingTarget{kind: domain.ChatGroup, first: command.groupID}
	bucket := h.typingByTarget[target]
	ids := make([]string, 0)
	for id, entry := range bucket {
		delete(entry.recipientIDs, command.userID)
		if entry.userID == command.userID {
			ids = append(ids, id)
		}
	}
	for _, id := range ids {
		h.removeTypingClient(target, id, true)
	}
}

func (h *Hub) expireTyping(now time.Time) {
	for target, bucket := range h.typingByTarget {
		ids := make([]string, 0)
		for id, entry := range bucket {
			if !entry.expiresAt.After(now) {
				ids = append(ids, id)
			}
		}
		for _, id := range ids {
			h.removeTypingClient(target, id, true)
		}
	}
}

func (h *Hub) notifyTyping(target typingTarget, entry *typingEntry, active bool) {
	if entry == nil {
		return
	}
	chat := domain.ChatRef{Kind: target.kind, TargetID: target.first}
	if target.kind == domain.ChatDirect {
		chat.TargetID = entry.userID
	}
	payload, err := json.Marshal(typingEnvelope{
		Type: "typing:update", Chat: chat, Typing: active,
		User: typingUser{ID: entry.userID, DisplayName: entry.displayName, AvatarURL: domain.NeutralAvatarPlaceholderURL},
	})
	if err != nil {
		return
	}
	for userID := range entry.recipientIDs {
		h.deliverToUser(userID, payload)
	}
}

func typingUserActive(bucket map[string]*typingEntry, userID int64) bool {
	for _, entry := range bucket {
		if entry.userID == userID {
			return true
		}
	}
	return false
}

func normalizeRecipientSet(values []int64, senderUserID int64) map[int64]struct{} {
	result := make(map[int64]struct{}, len(values))
	for _, value := range values {
		if value > 0 && value != senderUserID {
			result[value] = struct{}{}
		}
	}
	return result
}
