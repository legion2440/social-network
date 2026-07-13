package domain

import "time"

type Gender string

const (
	GenderMale   Gender = "male"
	GenderFemale Gender = "female"
)

func (g Gender) Valid() bool {
	return g == GenderMale || g == GenderFemale
}

type User struct {
	ID            int64     `json:"id"`
	Email         string    `json:"email"`
	PasswordHash  string    `json:"-"`
	FirstName     string    `json:"first_name"`
	LastName      string    `json:"last_name"`
	DateOfBirth   time.Time `json:"date_of_birth"`
	Gender        *Gender   `json:"gender,omitempty"`
	Nickname      *string   `json:"nickname,omitempty"`
	AboutMe       *string   `json:"about_me,omitempty"`
	AvatarMediaID *int64    `json:"avatar_media_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
