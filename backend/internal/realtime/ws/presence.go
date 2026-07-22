package ws

import "encoding/json"

type presenceInitEnvelope struct {
	Type          string  `json:"type"`
	OnlineUserIDs []int64 `json:"online_user_ids"`
}

type presenceUpdateEnvelope struct {
	Type   string `json:"type"`
	UserID int64  `json:"user_id"`
	Online bool   `json:"online"`
}

type presenceRemoveEnvelope struct {
	Type   string `json:"type"`
	UserID int64  `json:"user_id"`
}

func marshalPresenceInit(userIDs []int64) ([]byte, error) {
	return json.Marshal(presenceInitEnvelope{Type: "presence:init", OnlineUserIDs: userIDs})
}

func marshalPresenceUpdate(userID int64, online bool) ([]byte, error) {
	return json.Marshal(presenceUpdateEnvelope{Type: "presence:update", UserID: userID, Online: online})
}

func marshalPresenceRemove(userID int64) ([]byte, error) {
	return json.Marshal(presenceRemoveEnvelope{Type: "presence:remove", UserID: userID})
}

func (h *Hub) setPresencePeers(userID int64, peerIDs []int64) {
	peers := make(map[int64]struct{}, len(peerIDs))
	for _, peerID := range peerIDs {
		if peerID > 0 && peerID != userID {
			peers[peerID] = struct{}{}
		}
	}
	h.presencePeers[userID] = peers
}

func (h *Hub) notifyPresenceTransition(userID int64, online bool) {
	payload, err := marshalPresenceUpdate(userID, online)
	if err != nil {
		return
	}
	for peerID := range h.presencePeers[userID] {
		h.deliverToUser(peerID, payload)
	}
}

func (h *Hub) relationshipChanged(command relationshipCommand) {
	h.presenceGen[command.firstUserID]++
	h.presenceGen[command.secondUserID]++
	firstPeers := h.presencePeers[command.firstUserID]
	if firstPeers == nil {
		firstPeers = make(map[int64]struct{})
		h.presencePeers[command.firstUserID] = firstPeers
	}
	secondPeers := h.presencePeers[command.secondUserID]
	if secondPeers == nil {
		secondPeers = make(map[int64]struct{})
		h.presencePeers[command.secondUserID] = secondPeers
	}
	if command.eligible {
		firstPeers[command.secondUserID] = struct{}{}
		secondPeers[command.firstUserID] = struct{}{}
		if payload, err := marshalPresenceUpdate(command.secondUserID, h.userOnline(command.secondUserID)); err == nil {
			h.deliverToUser(command.firstUserID, payload)
		}
		if payload, err := marshalPresenceUpdate(command.firstUserID, h.userOnline(command.firstUserID)); err == nil {
			h.deliverToUser(command.secondUserID, payload)
		}
		return
	}
	delete(firstPeers, command.secondUserID)
	delete(secondPeers, command.firstUserID)
	if payload, err := marshalPresenceRemove(command.secondUserID); err == nil {
		h.deliverToUser(command.firstUserID, payload)
	}
	if payload, err := marshalPresenceRemove(command.firstUserID); err == nil {
		h.deliverToUser(command.secondUserID, payload)
	}
	h.clearDirectTyping(command.firstUserID, command.secondUserID)
}

func (h *Hub) deliverToUser(userID int64, payload []byte) {
	ids := make([]string, 0)
	for id, client := range h.clientsByUser[userID] {
		if h.clientCanReceive(client) {
			ids = append(ids, id)
		}
	}
	for _, id := range ids {
		client := h.clients[id]
		if client != nil && !h.enqueue(client, payload) {
			h.deactivateClient(id, deactivateSlow)
		}
	}
}
