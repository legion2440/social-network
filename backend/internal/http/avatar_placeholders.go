package http

import "embed"

//go:embed static/avatars/*.svg
var avatarPlaceholderFiles embed.FS
