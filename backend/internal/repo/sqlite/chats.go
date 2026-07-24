package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"social-network/backend/internal/domain"
	"social-network/backend/internal/repo"

	githubsqlite "github.com/mattn/go-sqlite3"
)

type ChatRepo struct {
	db sqlExecutor
}

func (r *ChatRepo) EnsureUserState(ctx context.Context, userID int64) error {
	if r == nil || r.db == nil || userID <= 0 {
		return fmt.Errorf("invalid chat user state")
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO chat_user_states (user_id, revision)
		VALUES (?, 0)
		ON CONFLICT(user_id) DO NOTHING
	`, userID)
	return err
}

func (r *ChatRepo) CurrentRevision(ctx context.Context, userID int64) (int64, error) {
	if r == nil || r.db == nil || userID <= 0 {
		return 0, repo.ErrNotFound
	}
	var revision int64
	err := r.db.QueryRowContext(ctx, `
		SELECT revision
		FROM chat_user_states
		WHERE user_id = ?
	`, userID).Scan(&revision)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, repo.ErrNotFound
	}
	return revision, err
}

func (r *ChatRepo) BumpRevision(ctx context.Context, userID int64) (int64, error) {
	if r == nil || r.db == nil || userID <= 0 {
		return 0, repo.ErrNotFound
	}
	result, err := r.db.ExecContext(ctx, `
		UPDATE chat_user_states
		SET revision = revision + 1
		WHERE user_id = ?
	`, userID)
	if err != nil {
		return 0, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if affected != 1 {
		return 0, repo.ErrNotFound
	}
	return r.CurrentRevision(ctx, userID)
}

func (r *ChatRepo) TotalUnreadCount(ctx context.Context, userID int64) (int64, error) {
	if r == nil || r.db == nil || userID <= 0 {
		return 0, repo.ErrNotFound
	}
	var count int64
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COALESCE((
				SELECT SUM(state.unread_count)
				FROM direct_chat_read_states state
				WHERE state.user_id = ?
			), 0)
			+
			COALESCE((
				SELECT SUM(state.unread_count)
				FROM group_chat_read_states state
				JOIN group_memberships membership ON membership.id = state.membership_id
				WHERE membership.user_id = ?
					AND membership.status IN ('owner', 'member')
			), 0)
	`, userID, userID).Scan(&count)
	return count, err
}

func (r *ChatRepo) GetDirectConversation(ctx context.Context, userLowID, userHighID int64) (*domain.DirectConversation, error) {
	if r == nil || r.db == nil || userLowID <= 0 || userHighID <= userLowID {
		return nil, repo.ErrNotFound
	}
	return scanDirectConversation(r.db.QueryRowContext(ctx, `
		SELECT id, user_low_id, user_high_id, created_at
		FROM direct_conversations
		WHERE user_low_id = ? AND user_high_id = ?
	`, userLowID, userHighID))
}

func (r *ChatRepo) LatestDirectMessageID(ctx context.Context, conversationID int64) (*int64, error) {
	return r.latestMessageID(ctx, "direct_conversation_id", conversationID)
}

func (r *ChatRepo) LatestGroupMessageID(ctx context.Context, groupID int64) (*int64, error) {
	return r.latestMessageID(ctx, "group_id", groupID)
}

func (r *ChatRepo) latestMessageID(ctx context.Context, column string, targetID int64) (*int64, error) {
	if r == nil || r.db == nil || targetID <= 0 || (column != "direct_conversation_id" && column != "group_id") {
		return nil, repo.ErrNotFound
	}
	var id int64
	err := r.db.QueryRowContext(ctx, `
		SELECT id
		FROM chat_messages
		WHERE `+column+` = ?
		ORDER BY created_at DESC, id DESC
		LIMIT 1
	`, targetID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func (r *ChatRepo) GetDirectMessage(ctx context.Context, conversationID, messageID int64) (*domain.ChatMessage, error) {
	return r.getTargetMessage(ctx, "direct_conversation_id", conversationID, messageID, domain.ChatDirect)
}

func (r *ChatRepo) GetGroupMessage(ctx context.Context, groupID, messageID int64) (*domain.ChatMessage, error) {
	return r.getTargetMessage(ctx, "group_id", groupID, messageID, domain.ChatGroup)
}

func (r *ChatRepo) getTargetMessage(
	ctx context.Context,
	column string,
	targetID, messageID int64,
	kind domain.ChatKind,
) (*domain.ChatMessage, error) {
	if r == nil || r.db == nil || targetID <= 0 || messageID <= 0 ||
		(column != "direct_conversation_id" && column != "group_id") {
		return nil, repo.ErrNotFound
	}
	var message domain.ChatMessage
	var createdAt int64
	err := r.db.QueryRowContext(ctx, `
		SELECT id, sender_user_id, client_message_id, body, created_at
		FROM chat_messages
		WHERE id = ? AND `+column+` = ?
	`, messageID, targetID).Scan(
		&message.ID, &message.SenderUserID, &message.ClientMessageID, &message.Body, &createdAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repo.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	message.CreatedAt = unixToTime(createdAt)
	message.Chat = domain.ChatRef{Kind: kind, TargetID: targetID}
	if kind == domain.ChatDirect {
		message.DirectConversationID = &targetID
	} else {
		message.GroupID = &targetID
	}
	return &message, nil
}

func (r *ChatRepo) EnsureDirectReadState(
	ctx context.Context,
	userID, conversationID int64,
	markerID *int64,
	updatedAt time.Time,
) error {
	if r == nil || r.db == nil || userID <= 0 || conversationID <= 0 || updatedAt.IsZero() {
		return fmt.Errorf("invalid direct chat read state")
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO direct_chat_read_states (
			user_id, direct_conversation_id, last_read_message_id, unread_count, updated_at
		) VALUES (?, ?, ?, 0, ?)
		ON CONFLICT(user_id, direct_conversation_id) DO NOTHING
	`, userID, conversationID, markerID, timeToUnix(updatedAt))
	return err
}

func (r *ChatRepo) EnsureGroupReadState(
	ctx context.Context,
	membershipID int64,
	markerID *int64,
	updatedAt time.Time,
) error {
	if r == nil || r.db == nil || membershipID <= 0 || updatedAt.IsZero() {
		return fmt.Errorf("invalid group chat read state")
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO group_chat_read_states (
			membership_id, last_read_message_id, unread_count, updated_at
		) VALUES (?, ?, 0, ?)
		ON CONFLICT(membership_id) DO NOTHING
	`, membershipID, markerID, timeToUnix(updatedAt))
	return err
}

func (r *ChatRepo) IncrementDirectUnread(
	ctx context.Context,
	userID, conversationID int64,
	updatedAt time.Time,
) error {
	return r.incrementUnread(ctx, `
		UPDATE direct_chat_read_states
		SET unread_count = unread_count + 1, updated_at = ?
		WHERE user_id = ? AND direct_conversation_id = ?
	`, timeToUnix(updatedAt), userID, conversationID)
}

func (r *ChatRepo) IncrementGroupUnread(ctx context.Context, membershipID int64, updatedAt time.Time) error {
	return r.incrementUnread(ctx, `
		UPDATE group_chat_read_states
		SET unread_count = unread_count + 1, updated_at = ?
		WHERE membership_id = ?
	`, timeToUnix(updatedAt), membershipID)
}

func (r *ChatRepo) incrementUnread(ctx context.Context, query string, args ...any) error {
	if r == nil || r.db == nil {
		return repo.ErrNotFound
	}
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return repo.ErrNotFound
	}
	return nil
}

func (r *ChatRepo) DirectUnreadState(
	ctx context.Context,
	userID, conversationID int64,
) (*domain.ChatUnreadState, error) {
	if r == nil || r.db == nil || userID <= 0 || conversationID <= 0 {
		return nil, repo.ErrNotFound
	}
	var state domain.ChatUnreadState
	var marker sql.NullInt64
	err := r.db.QueryRowContext(ctx, `
		SELECT
			CASE WHEN conversation.user_low_id = ? THEN conversation.user_high_id ELSE conversation.user_low_id END,
			state.unread_count,
			user_state.revision,
			state.last_read_message_id
		FROM direct_chat_read_states state
		JOIN direct_conversations conversation ON conversation.id = state.direct_conversation_id
		JOIN chat_user_states user_state ON user_state.user_id = state.user_id
		WHERE state.user_id = ? AND state.direct_conversation_id = ?
	`, userID, userID, conversationID).Scan(
		&state.Chat.TargetID, &state.ChatUnreadCount, &state.Revision, &marker,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repo.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	state.Chat.Kind = domain.ChatDirect
	if marker.Valid {
		value := marker.Int64
		state.ReadThroughMessageID = &value
	}
	state.UnreadCount, err = r.TotalUnreadCount(ctx, userID)
	return &state, err
}

func (r *ChatRepo) GroupUnreadState(
	ctx context.Context,
	userID, membershipID, groupID int64,
) (*domain.ChatUnreadState, error) {
	if r == nil || r.db == nil || userID <= 0 || membershipID <= 0 || groupID <= 0 {
		return nil, repo.ErrNotFound
	}
	state := &domain.ChatUnreadState{Chat: domain.ChatRef{Kind: domain.ChatGroup, TargetID: groupID}}
	var marker sql.NullInt64
	err := r.db.QueryRowContext(ctx, `
		SELECT state.unread_count, user_state.revision, state.last_read_message_id
		FROM group_chat_read_states state
		JOIN group_memberships membership ON membership.id = state.membership_id
		JOIN chat_user_states user_state ON user_state.user_id = membership.user_id
		WHERE state.membership_id = ? AND membership.user_id = ? AND membership.group_id = ?
			AND membership.status IN ('owner', 'member')
	`, membershipID, userID, groupID).Scan(&state.ChatUnreadCount, &state.Revision, &marker)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repo.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if marker.Valid {
		value := marker.Int64
		state.ReadThroughMessageID = &value
	}
	state.UnreadCount, err = r.TotalUnreadCount(ctx, userID)
	return state, err
}

func (r *ChatRepo) AdvanceDirectRead(
	ctx context.Context,
	userID, conversationID, messageID int64,
	updatedAt time.Time,
) (bool, error) {
	return r.advanceRead(ctx, `
		UPDATE direct_chat_read_states
		SET
			last_read_message_id = ?,
			unread_count = (
				SELECT COUNT(*)
				FROM chat_messages newer
				JOIN chat_messages candidate ON candidate.id = ?
				WHERE newer.direct_conversation_id = ?
					AND newer.sender_user_id != ?
					AND (
						newer.created_at > candidate.created_at OR
						(newer.created_at = candidate.created_at AND newer.id > candidate.id)
					)
			),
			updated_at = ?
		WHERE user_id = ? AND direct_conversation_id = ?
			AND (
				last_read_message_id IS NULL OR EXISTS (
					SELECT 1
					FROM chat_messages candidate
					JOIN chat_messages current ON current.id = direct_chat_read_states.last_read_message_id
					WHERE candidate.id = ?
						AND (
							candidate.created_at > current.created_at OR
							(candidate.created_at = current.created_at AND candidate.id > current.id)
						)
				)
			)
	`, messageID, messageID, conversationID, userID, timeToUnix(updatedAt), userID, conversationID, messageID)
}

func (r *ChatRepo) AdvanceGroupRead(
	ctx context.Context,
	membershipID, groupID, messageID int64,
	updatedAt time.Time,
) (bool, error) {
	return r.advanceRead(ctx, `
		UPDATE group_chat_read_states
		SET
			last_read_message_id = ?,
			unread_count = (
				SELECT COUNT(*)
				FROM chat_messages newer
				JOIN chat_messages candidate ON candidate.id = ?
				JOIN group_memberships membership ON membership.id = ?
				WHERE newer.group_id = ?
					AND newer.sender_user_id != membership.user_id
					AND (
						newer.created_at > candidate.created_at OR
						(newer.created_at = candidate.created_at AND newer.id > candidate.id)
					)
			),
			updated_at = ?
		WHERE membership_id = ?
			AND (
				last_read_message_id IS NULL OR EXISTS (
					SELECT 1
					FROM chat_messages candidate
					JOIN chat_messages current ON current.id = group_chat_read_states.last_read_message_id
					WHERE candidate.id = ?
						AND (
							candidate.created_at > current.created_at OR
							(candidate.created_at = current.created_at AND candidate.id > current.id)
						)
				)
			)
	`, messageID, messageID, membershipID, groupID, timeToUnix(updatedAt), membershipID, messageID)
}

func (r *ChatRepo) advanceRead(ctx context.Context, query string, args ...any) (bool, error) {
	if r == nil || r.db == nil {
		return false, repo.ErrNotFound
	}
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	return affected > 0, err
}

func (r *ChatRepo) EnsureDirectConversation(ctx context.Context, userLowID, userHighID int64, createdAt time.Time) (*domain.DirectConversation, error) {
	if r == nil || r.db == nil || userLowID <= 0 || userHighID <= userLowID || createdAt.IsZero() {
		return nil, fmt.Errorf("invalid direct conversation")
	}
	if _, err := r.db.ExecContext(ctx, `
		INSERT INTO direct_conversations (user_low_id, user_high_id, created_at)
		VALUES (?, ?, ?)
		ON CONFLICT(user_low_id, user_high_id) DO NOTHING
	`, userLowID, userHighID, timeToUnix(createdAt)); err != nil {
		return nil, err
	}
	return r.GetDirectConversation(ctx, userLowID, userHighID)
}

func scanDirectConversation(row rowScanner) (*domain.DirectConversation, error) {
	var conversation domain.DirectConversation
	var createdAt int64
	if err := row.Scan(&conversation.ID, &conversation.UserLowID, &conversation.UserHighID, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repo.ErrNotFound
		}
		return nil, err
	}
	conversation.CreatedAt = unixToTime(createdAt)
	return &conversation, nil
}

func (r *ChatRepo) CreateMessage(ctx context.Context, message *domain.ChatMessage) (int64, error) {
	if r == nil || r.db == nil || message == nil || message.SenderUserID <= 0 || strings.TrimSpace(message.ClientMessageID) == "" || strings.TrimSpace(message.Body) == "" || message.CreatedAt.IsZero() {
		return 0, fmt.Errorf("invalid chat message")
	}
	if (message.DirectConversationID == nil) == (message.GroupID == nil) {
		return 0, fmt.Errorf("chat message must have exactly one target")
	}
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO chat_messages (
			direct_conversation_id, group_id, sender_user_id,
			client_message_id, body, created_at
		) VALUES (?, ?, ?, ?, ?, ?)
	`, message.DirectConversationID, message.GroupID, message.SenderUserID, message.ClientMessageID, message.Body, timeToUnix(message.CreatedAt))
	if err != nil {
		var sqliteErr githubsqlite.Error
		if errors.As(err, &sqliteErr) && (sqliteErr.ExtendedCode == githubsqlite.ErrConstraintUnique || sqliteErr.ExtendedCode == githubsqlite.ErrConstraintPrimaryKey) {
			return 0, repo.ErrConflict
		}
		return 0, err
	}
	return result.LastInsertId()
}

const chatMessageSelect = `
	SELECT
		m.id, m.direct_conversation_id, m.group_id, m.sender_user_id,
		m.client_message_id, m.body, m.created_at,
		CASE WHEN m.direct_conversation_id IS NOT NULL THEN 'direct' ELSE 'group' END,
		CASE
			WHEN m.direct_conversation_id IS NOT NULL AND dc.user_low_id = ? THEN dc.user_high_id
			WHEN m.direct_conversation_id IS NOT NULL THEN dc.user_low_id
			ELSE m.group_id
		END,
		sender.id, sender.first_name, sender.last_name, sender.nickname,
		CASE
			WHEN sender.avatar_media_id IS NULL THEN NULL
			WHEN sender.id = ? OR sender.is_private = 0 OR EXISTS (
				SELECT 1 FROM follows sender_avatar_follow
				WHERE sender_avatar_follow.follower_user_id = ?
					AND sender_avatar_follow.followed_user_id = sender.id
					AND sender_avatar_follow.status = 'accepted'
			) THEN sender.avatar_media_id
			ELSE NULL
		END,
		sender.is_private
	FROM chat_messages m
	LEFT JOIN direct_conversations dc ON dc.id = m.direct_conversation_id
	JOIN users sender ON sender.id = m.sender_user_id
`

func (r *ChatRepo) GetMessageByClientID(ctx context.Context, senderUserID int64, clientMessageID string) (*domain.ChatMessage, error) {
	if r == nil || r.db == nil || senderUserID <= 0 || strings.TrimSpace(clientMessageID) == "" {
		return nil, repo.ErrNotFound
	}
	query := chatMessageSelect + ` WHERE m.sender_user_id = ? AND m.client_message_id = ?`
	return scanChatMessage(r.db.QueryRowContext(ctx, query, senderUserID, senderUserID, senderUserID, senderUserID, clientMessageID))
}

func (r *ChatRepo) ListDirectMessages(ctx context.Context, viewerUserID, targetUserID int64, cursor *domain.ChatMessageCursor, limit int) ([]*domain.ChatMessage, error) {
	if r == nil || r.db == nil || viewerUserID <= 0 || targetUserID <= 0 || limit <= 0 {
		return []*domain.ChatMessage{}, nil
	}
	low, high := normalizedPair(viewerUserID, targetUserID)
	query := chatMessageSelect + `
		WHERE dc.user_low_id = ? AND dc.user_high_id = ?
	`
	args := []any{viewerUserID, viewerUserID, viewerUserID, low, high}
	query, args = appendChatMessageCursor(query, args, cursor)
	query += ` ORDER BY m.created_at DESC, m.id DESC LIMIT ?`
	args = append(args, limit)
	return r.listMessages(ctx, query, args...)
}

func (r *ChatRepo) ListGroupMessages(ctx context.Context, viewerUserID, groupID int64, cursor *domain.ChatMessageCursor, limit int) ([]*domain.ChatMessage, error) {
	if r == nil || r.db == nil || viewerUserID <= 0 || groupID <= 0 || limit <= 0 {
		return []*domain.ChatMessage{}, nil
	}
	query := chatMessageSelect + ` WHERE m.group_id = ?`
	args := []any{viewerUserID, viewerUserID, viewerUserID, groupID}
	query, args = appendChatMessageCursor(query, args, cursor)
	query += ` ORDER BY m.created_at DESC, m.id DESC LIMIT ?`
	args = append(args, limit)
	return r.listMessages(ctx, query, args...)
}

func appendChatMessageCursor(query string, args []any, cursor *domain.ChatMessageCursor) (string, []any) {
	if cursor == nil {
		return query, args
	}
	timestamp := timeToUnix(cursor.CreatedAt)
	query += ` AND (m.created_at < ? OR (m.created_at = ? AND m.id < ?))`
	return query, append(args, timestamp, timestamp, cursor.ID)
}

func (r *ChatRepo) listMessages(ctx context.Context, query string, args ...any) ([]*domain.ChatMessage, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	messages := make([]*domain.ChatMessage, 0)
	for rows.Next() {
		message, err := scanChatMessage(rows)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, rows.Err()
}

func scanChatMessage(row rowScanner) (*domain.ChatMessage, error) {
	var (
		message              domain.ChatMessage
		directConversationID sql.NullInt64
		groupID              sql.NullInt64
		createdAt            int64
		kind                 string
		sender               domain.User
		nickname             sql.NullString
		avatarMediaID        sql.NullInt64
		isPrivate            int
	)
	if err := row.Scan(
		&message.ID, &directConversationID, &groupID, &message.SenderUserID,
		&message.ClientMessageID, &message.Body, &createdAt, &kind, &message.Chat.TargetID,
		&sender.ID, &sender.FirstName, &sender.LastName, &nickname, &avatarMediaID, &isPrivate,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repo.ErrNotFound
		}
		return nil, err
	}
	message.Chat.Kind = domain.ChatKind(kind)
	if !message.Chat.Kind.Valid() || message.Chat.TargetID <= 0 {
		return nil, fmt.Errorf("invalid chat target")
	}
	if directConversationID.Valid {
		value := directConversationID.Int64
		message.DirectConversationID = &value
	}
	if groupID.Valid {
		value := groupID.Int64
		message.GroupID = &value
	}
	message.CreatedAt = unixToTime(createdAt)
	sender.Nickname = stringFromNullString(nickname)
	if avatarMediaID.Valid {
		value := avatarMediaID.Int64
		sender.AvatarMediaID = &value
	}
	if isPrivate != 0 && isPrivate != 1 {
		return nil, fmt.Errorf("invalid sender privacy")
	}
	sender.IsPrivate = isPrivate == 1
	message.Sender = &sender
	return &message, nil
}

func (r *ChatRepo) ListDirectPeerIDs(ctx context.Context, userID int64) ([]int64, error) {
	if r == nil || r.db == nil || userID <= 0 {
		return []int64{}, nil
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT DISTINCT CASE
			WHEN follower_user_id = ? THEN followed_user_id
			ELSE follower_user_id
		END AS peer_id
		FROM follows
		WHERE status = 'accepted'
			AND (follower_user_id = ? OR followed_user_id = ?)
		ORDER BY peer_id ASC
	`, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *ChatRepo) ListChats(ctx context.Context, viewerUserID int64, cursor *domain.ChatListCursor, limit int) ([]*domain.ChatSummary, error) {
	if r == nil || r.db == nil || viewerUserID <= 0 || limit <= 0 {
		return []*domain.ChatSummary{}, nil
	}
	query := `
		WITH chat_rows AS (
			SELECT
				'direct' AS kind, 0 AS kind_rank, dc.id AS entity_id,
				CASE WHEN dc.user_low_id = ? THEN dc.user_high_id ELSE dc.user_low_id END AS target_id,
				NULL AS group_id,
				COALESCE(last_message.created_at, dc.created_at) AS activity_at,
				last_message.id AS last_message_id,
				NULL AS membership_status,
				COALESCE(read_state.unread_count, 0) AS unread_count
			FROM direct_conversations dc
			LEFT JOIN direct_chat_read_states read_state
				ON read_state.direct_conversation_id = dc.id AND read_state.user_id = ?
			LEFT JOIN chat_messages last_message ON last_message.id = (
				SELECT newest.id FROM chat_messages newest
				WHERE newest.direct_conversation_id = dc.id
				ORDER BY newest.created_at DESC, newest.id DESC LIMIT 1
			)
			WHERE dc.user_low_id = ? OR dc.user_high_id = ?

			UNION ALL

			SELECT
				'group' AS kind, 1 AS kind_rank, g.id AS entity_id,
				g.id AS target_id, g.id AS group_id,
				COALESCE(last_message.created_at, membership.updated_at) AS activity_at,
				last_message.id AS last_message_id,
				membership.status AS membership_status,
				COALESCE(read_state.unread_count, 0) AS unread_count
			FROM group_memberships membership
			JOIN groups g ON g.id = membership.group_id
			LEFT JOIN group_chat_read_states read_state ON read_state.membership_id = membership.id
			LEFT JOIN chat_messages last_message ON last_message.id = (
				SELECT newest.id FROM chat_messages newest
				WHERE newest.group_id = g.id
				ORDER BY newest.created_at DESC, newest.id DESC LIMIT 1
			)
			WHERE membership.user_id = ? AND membership.status IN ('owner', 'member')
		)
		SELECT
			row.kind, row.kind_rank, row.entity_id, row.target_id, row.activity_at, row.membership_status,
			row.unread_count,
			peer.id, peer.first_name, peer.last_name, peer.nickname,
			CASE
				WHEN peer.avatar_media_id IS NULL THEN NULL
				WHEN peer.id = ? OR peer.is_private = 0 OR EXISTS (
					SELECT 1 FROM follows peer_avatar_follow
					WHERE peer_avatar_follow.follower_user_id = ?
						AND peer_avatar_follow.followed_user_id = peer.id
						AND peer_avatar_follow.status = 'accepted'
				) THEN peer.avatar_media_id ELSE NULL
			END,
			peer.is_private,
			g.id, g.owner_user_id, g.title, g.description, g.created_at,
			CASE WHEN g.id IS NULL THEN NULL ELSE (
				SELECT COUNT(*) FROM group_memberships counted
				WHERE counted.group_id = g.id AND counted.status IN ('owner', 'member')
			) END,
			owner.id, owner.first_name, owner.last_name, owner.nickname,
			CASE
				WHEN owner.avatar_media_id IS NULL THEN NULL
				WHEN owner.id = ? OR owner.is_private = 0 OR EXISTS (
					SELECT 1 FROM follows owner_avatar_follow
					WHERE owner_avatar_follow.follower_user_id = ?
						AND owner_avatar_follow.followed_user_id = owner.id
						AND owner_avatar_follow.status = 'accepted'
				) THEN owner.avatar_media_id ELSE NULL
			END,
			owner.is_private,
			last.id, last.direct_conversation_id, last.group_id, last.sender_user_id,
			last.client_message_id, last.body, last.created_at,
			sender.id, sender.first_name, sender.last_name, sender.nickname,
			CASE
				WHEN sender.avatar_media_id IS NULL THEN NULL
				WHEN sender.id = ? OR sender.is_private = 0 OR EXISTS (
					SELECT 1 FROM follows sender_avatar_follow
					WHERE sender_avatar_follow.follower_user_id = ?
						AND sender_avatar_follow.followed_user_id = sender.id
						AND sender_avatar_follow.status = 'accepted'
				) THEN sender.avatar_media_id ELSE NULL
			END,
			sender.is_private
		FROM chat_rows row
		LEFT JOIN users peer ON row.kind = 'direct' AND peer.id = row.target_id
		LEFT JOIN groups g ON row.kind = 'group' AND g.id = row.group_id
		LEFT JOIN users owner ON owner.id = g.owner_user_id
		LEFT JOIN chat_messages last ON last.id = row.last_message_id
		LEFT JOIN users sender ON sender.id = last.sender_user_id
		WHERE 1 = 1
	`
	args := []any{
		viewerUserID, viewerUserID, viewerUserID, viewerUserID, viewerUserID,
		viewerUserID, viewerUserID,
		viewerUserID, viewerUserID,
		viewerUserID, viewerUserID,
	}
	if cursor != nil {
		timestamp := timeToUnix(cursor.ActivityAt)
		query += ` AND (
			row.activity_at < ? OR
			(row.activity_at = ? AND row.kind_rank > ?) OR
			(row.activity_at = ? AND row.kind_rank = ? AND row.entity_id < ?)
		)`
		args = append(args, timestamp, timestamp, cursor.KindRank, timestamp, cursor.KindRank, cursor.EntityID)
	}
	query += ` ORDER BY row.activity_at DESC, row.kind_rank ASC, row.entity_id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	summaries := make([]*domain.ChatSummary, 0)
	for rows.Next() {
		summary, err := scanChatSummary(rows)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}
	return summaries, rows.Err()
}

type nullableChatUser struct {
	id            sql.NullInt64
	firstName     sql.NullString
	lastName      sql.NullString
	nickname      sql.NullString
	avatarMediaID sql.NullInt64
	isPrivate     sql.NullInt64
}

func (u *nullableChatUser) destinations() []any {
	return []any{&u.id, &u.firstName, &u.lastName, &u.nickname, &u.avatarMediaID, &u.isPrivate}
}

func (u *nullableChatUser) value() (*domain.User, error) {
	if !u.id.Valid {
		return nil, nil
	}
	if !u.firstName.Valid || !u.lastName.Valid || !u.isPrivate.Valid || (u.isPrivate.Int64 != 0 && u.isPrivate.Int64 != 1) {
		return nil, fmt.Errorf("invalid chat user")
	}
	user := &domain.User{ID: u.id.Int64, FirstName: u.firstName.String, LastName: u.lastName.String, IsPrivate: u.isPrivate.Int64 == 1}
	user.Nickname = stringFromNullString(u.nickname)
	if u.avatarMediaID.Valid {
		value := u.avatarMediaID.Int64
		user.AvatarMediaID = &value
	}
	return user, nil
}

func scanChatSummary(row rowScanner) (*domain.ChatSummary, error) {
	var (
		summary           domain.ChatSummary
		kind              string
		kindRank          int
		activityAt        int64
		membershipStatus  sql.NullString
		peer              nullableChatUser
		groupID           sql.NullInt64
		groupOwnerID      sql.NullInt64
		groupTitle        sql.NullString
		groupDescription  sql.NullString
		groupCreatedAt    sql.NullInt64
		groupMembersCount sql.NullInt64
		owner             nullableChatUser
		messageID         sql.NullInt64
		messageDirectID   sql.NullInt64
		messageGroupID    sql.NullInt64
		messageSenderID   sql.NullInt64
		clientMessageID   sql.NullString
		messageBody       sql.NullString
		messageCreatedAt  sql.NullInt64
		sender            nullableChatUser
	)
	destinations := []any{
		&kind, &kindRank, &summary.EntityID, &summary.TargetID, &activityAt, &membershipStatus,
		&summary.UnreadCount,
	}
	destinations = append(destinations, peer.destinations()...)
	destinations = append(destinations,
		&groupID, &groupOwnerID, &groupTitle, &groupDescription, &groupCreatedAt, &groupMembersCount,
	)
	destinations = append(destinations, owner.destinations()...)
	destinations = append(destinations,
		&messageID, &messageDirectID, &messageGroupID, &messageSenderID,
		&clientMessageID, &messageBody, &messageCreatedAt,
	)
	destinations = append(destinations, sender.destinations()...)
	if err := row.Scan(destinations...); err != nil {
		return nil, err
	}
	summary.Kind = domain.ChatKind(kind)
	if !summary.Kind.Valid() || kindRank < 0 || kindRank > 1 || summary.EntityID <= 0 || summary.TargetID <= 0 || summary.UnreadCount < 0 {
		return nil, fmt.Errorf("invalid chat summary")
	}
	summary.ActivityAt = unixToTime(activityAt)
	var err error
	if summary.Kind == domain.ChatDirect {
		summary.User, err = peer.value()
		if err != nil {
			return nil, fmt.Errorf("invalid direct chat user: %w", err)
		}
		if summary.User == nil {
			return nil, fmt.Errorf("invalid direct chat user")
		}
	} else {
		if !groupID.Valid || !groupOwnerID.Valid || !groupTitle.Valid || !groupDescription.Valid || !groupCreatedAt.Valid || !groupMembersCount.Valid || !membershipStatus.Valid {
			return nil, fmt.Errorf("invalid group chat")
		}
		status := domain.GroupMembershipStatus(membershipStatus.String)
		if status != domain.GroupOwner && status != domain.GroupMember {
			return nil, fmt.Errorf("invalid group chat membership")
		}
		groupOwner, ownerErr := owner.value()
		if ownerErr != nil {
			return nil, fmt.Errorf("invalid group owner: %w", ownerErr)
		}
		if groupOwner == nil {
			return nil, fmt.Errorf("invalid group owner")
		}
		summary.Group = &domain.Group{
			ID: groupID.Int64, OwnerUserID: groupOwnerID.Int64, Owner: groupOwner,
			Title: groupTitle.String, Description: groupDescription.String,
			CreatedAt: unixToTime(groupCreatedAt.Int64), MembersCount: groupMembersCount.Int64,
			ViewerStatus: &status,
		}
	}
	if messageID.Valid {
		if !messageSenderID.Valid || !clientMessageID.Valid || !messageBody.Valid || !messageCreatedAt.Valid {
			return nil, fmt.Errorf("invalid last chat message")
		}
		messageSender, senderErr := sender.value()
		if senderErr != nil {
			return nil, fmt.Errorf("invalid last message sender: %w", senderErr)
		}
		if messageSender == nil {
			return nil, fmt.Errorf("invalid last message sender")
		}
		message := &domain.ChatMessage{
			ID: messageID.Int64, SenderUserID: messageSenderID.Int64,
			ClientMessageID: clientMessageID.String, Body: messageBody.String,
			CreatedAt: unixToTime(messageCreatedAt.Int64), Sender: messageSender,
			Chat: domain.ChatRef{Kind: summary.Kind, TargetID: summary.TargetID},
		}
		if messageDirectID.Valid {
			value := messageDirectID.Int64
			message.DirectConversationID = &value
		}
		if messageGroupID.Valid {
			value := messageGroupID.Int64
			message.GroupID = &value
		}
		summary.LastMessage = message
	}
	return &summary, nil
}

func normalizedPair(first, second int64) (int64, int64) {
	if first < second {
		return first, second
	}
	return second, first
}
