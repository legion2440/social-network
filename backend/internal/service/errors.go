package service

import "errors"

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrNotFound           = errors.New("not found")
	ErrEmailTaken         = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidMediaType   = errors.New("invalid media type")
	ErrMediaTooBig        = errors.New("media is too big")
)
