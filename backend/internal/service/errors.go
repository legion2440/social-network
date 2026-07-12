package service

import "errors"

var (
	ErrInvalidInput     = errors.New("invalid input")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrNotFound         = errors.New("not found")
	ErrInvalidMediaType = errors.New("invalid media type")
	ErrMediaTooBig      = errors.New("media is too big")
)
