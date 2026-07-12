package service

import "golang.org/x/crypto/bcrypt"

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

type BcryptHasher struct{}

func (BcryptHasher) Hash(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func (BcryptHasher) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func HashPassword(password string) (string, error) {
	return (BcryptHasher{}).Hash(password)
}

func VerifyPassword(hash, password string) error {
	return (BcryptHasher{}).Compare(hash, password)
}
