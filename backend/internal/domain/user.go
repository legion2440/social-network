package domain

import (
	"errors"
	"time"
)

const DateOfBirthLayout = "02-01-2006"

var (
	ErrInvalidDateOfBirth = errors.New("invalid date_of_birth")
	ErrInvalidGender      = errors.New("invalid gender")
)

type Gender string

const (
	GenderMale   Gender = "male"
	GenderFemale Gender = "female"
)

func (g Gender) Valid() bool {
	return g == GenderMale || g == GenderFemale
}

func ValidDateOfBirth(value string) bool {
	if len(value) != len(DateOfBirthLayout) {
		return false
	}
	date, err := time.Parse(DateOfBirthLayout, value)
	return err == nil && date.Year() > 0 && date.Format(DateOfBirthLayout) == value
}

type User struct {
	ID            int64     `json:"id"`
	Email         string    `json:"email"`
	PasswordHash  string    `json:"-"`
	FirstName     string    `json:"first_name"`
	LastName      string    `json:"last_name"`
	DateOfBirth   string    `json:"date_of_birth"`
	Gender        *Gender   `json:"gender,omitempty"`
	Nickname      *string   `json:"nickname,omitempty"`
	AboutMe       *string   `json:"about_me,omitempty"`
	AvatarMediaID *int64    `json:"avatar_media_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
