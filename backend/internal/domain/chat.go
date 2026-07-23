package domain

import "time"

type ChatKind string

const (
	ChatDirect ChatKind = "direct"
	ChatGroup  ChatKind = "group"
)

func (k ChatKind) Valid() bool {
	return k == ChatDirect || k == ChatGroup
}

type ChatRef struct {
	Kind     ChatKind `json:"kind"`
	TargetID int64    `json:"target_id"`
}

type DirectConversation struct {
	ID         int64
	UserLowID  int64
	UserHighID int64
	CreatedAt  time.Time
}

type ChatMessage struct {
	ID                   int64
	DirectConversationID *int64
	GroupID              *int64
	SenderUserID         int64
	ClientMessageID      string
	Chat                 ChatRef
	Sender               *User
	Body                 string
	CreatedAt            time.Time
}

type ChatMessageCursor struct {
	CreatedAt time.Time
	ID        int64
}

type ChatListCursor struct {
	ActivityAt time.Time
	KindRank   int
	EntityID   int64
}

type ChatSummary struct {
	EntityID    int64
	Kind        ChatKind
	TargetID    int64
	User        *User
	Group       *Group
	LastMessage *ChatMessage
	ActivityAt  time.Time
	UnreadCount int64
}

type ChatUnreadState struct {
	Chat                 ChatRef
	ChatUnreadCount      int64
	UnreadCount          int64
	Revision             int64
	ReadThroughMessageID *int64
}

type ChatRecipient struct {
	UserID       int64
	MembershipID *int64
}
