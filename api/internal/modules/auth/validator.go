package auth

import (
	"errors"
	"fmt"
	"net/mail"
	"unicode"
)

const (
	minPasswordLength = 8
	maxPasswordLength = 128
	maxEmailLength    = 255
)

// ValidateRegisterRequest validates the registration payload.
func ValidateRegisterRequest(req RegisterRequest) error {
	if err := validateEmail(req.Email); err != nil {
		return err
	}
	return validatePassword(req.Password)
}

// ValidateLoginRequest validates the login payload.
func ValidateLoginRequest(req LoginRequest) error {
	if req.Email == "" {
		return errors.New("email is required")
	}
	if req.Password == "" {
		return errors.New("password is required")
	}
	return nil
}

func validateEmail(email string) error {
	if email == "" {
		return errors.New("email is required")
	}
	if len(email) > maxEmailLength {
		return fmt.Errorf("email must not exceed %d characters", maxEmailLength)
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return errors.New("invalid email format")
	}
	return nil
}

func validatePassword(password string) error {
	if password == "" {
		return errors.New("password is required")
	}
	if len(password) < minPasswordLength {
		return fmt.Errorf("password must be at least %d characters", minPasswordLength)
	}
	if len(password) > maxPasswordLength {
		return fmt.Errorf("password must not exceed %d characters", maxPasswordLength)
	}

	var hasUpper, hasLower, hasDigit bool
	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit {
		return errors.New("password must contain at least one uppercase letter, one lowercase letter, and one digit")
	}

	return nil
}
