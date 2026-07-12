package http

import "context"

type currentUserContextKey struct{}

type CurrentUser struct {
	ID           int64
	SessionToken string
}

func withCurrentUser(ctx context.Context, user CurrentUser) context.Context {
	return context.WithValue(ctx, currentUserContextKey{}, user)
}

func CurrentUserFromContext(ctx context.Context) (CurrentUser, bool) {
	user, ok := ctx.Value(currentUserContextKey{}).(CurrentUser)
	return user, ok && user.ID > 0
}
