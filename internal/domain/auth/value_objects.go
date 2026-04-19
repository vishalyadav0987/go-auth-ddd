package auth

import (
	"errors"
	"regexp"
	"unicode"
)

type Email string
type Mobile string
type Password string
type MPIN string

func NewEmail(value string) (Email, error) {
	if value == "" {
		return "", errors.New("email required")
	}

	regex := `^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`
	matched, _ := regexp.MatchString(regex, value)

	if !matched {
		return "", errors.New("invalid email format")
	}

	return Email(value), nil
}

func NewMobile(value string) (Mobile, error) {
	if value == "" {
		return "", errors.New("mobile number required")
	}

	if len(value) < 10 {
		return "", errors.New("invalid number")
	}

	regex := `^[+]?[(]?[0-9]{1,4}[)]?[-\s./0-9]*$`
	matched, _ := regexp.MatchString(regex, value)

	if !matched {
		return "", errors.New("invalid number format")
	}

	return Mobile(value), nil
}

func NewPassword(value string) (Password, error) {
	if value == "" {
		return Password(""), errors.New("password is required")
	}

	if len(value) < 8 {
		return Password(""), errors.New("password must be at least 8 characters long")
	}

	var hasUpper, hasLower, hasNumber, hasSpecial bool

	for _, r := range value {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasNumber = true
		case !unicode.IsLetter(r) && !unicode.IsDigit(r):
			hasSpecial = true
		}
	}

	if !hasUpper || !hasLower || !hasNumber || !hasSpecial {
		return Password(""), errors.New(
			"password must include at least one uppercase letter, one lowercase letter, one number, and one special character",
		)
	}

	return Password(value), nil
}

func NewMPIN(value string) (MPIN, error) {
	if len(value) != 4 {
		return "", errors.New("MPIN must be exactly 4 digits")
	}

	// ensure only digits
	for _, r := range value {
		if r < '0' || r > '9' {
			return "", errors.New("MPIN must contain only digits")
		}
	}

	// prevent all digits same (1111, 2222)
	if value[0] == value[1] && value[1] == value[2] && value[2] == value[3] {
		return "", errors.New("MPIN cannot be repeating digits")
	}

	// prevent ascending or descending sequences
	ascending := true
	descending := true

	for i := 1; i < 4; i++ {
		if value[i] != value[i-1]+1 {
			ascending = false
		}
		if value[i] != value[i-1]-1 {
			descending = false
		}
	}

	if ascending || descending {
		return "", errors.New("MPIN cannot be sequential digits")
	}

	return MPIN(value), nil
}
