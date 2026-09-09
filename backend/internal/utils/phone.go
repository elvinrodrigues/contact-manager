package utils

import (
	"errors"
	"strings"
	"unicode"
)

var (
	// ErrInvalidPhone is returned for anything that does not reduce to a
	// 10-digit subscriber number under the rules in NormalizePhone.
	ErrInvalidPhone = errors.New("phone must be a 10-digit number, optionally prefixed with +91, 91 or 0")
	// ErrDuplicatePhone signals a unique-constraint violation on (user_id, phone).
	ErrDuplicatePhone = errors.New("duplicate phone")
)

// maxPhoneInput bounds the work done on hostile input before validation.
const maxPhoneInput = 32

// NormalizePhone reduces a user-entered number to a bare 10-digit national
// number. Separators are stripped, then a +91 / 91 / leading-0 prefix is
// removed. Numbers outside this scheme are rejected rather than truncated.
func NormalizePhone(phone string) (string, error) {
	phone = strings.TrimSpace(phone)
	if phone == "" || len(phone) > maxPhoneInput {
		return "", ErrInvalidPhone
	}

	// Keep a single leading '+' and every digit; drop spaces, dashes, dots,
	// parentheses and any other separator.
	var b strings.Builder
	for i, r := range phone {
		switch {
		case unicode.IsDigit(r):
			b.WriteRune(r)
		case r == '+' && i == 0:
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '.' || r == '(' || r == ')':
			// separator, ignore
		default:
			return "", ErrInvalidPhone
		}
	}
	phone = b.String()

	switch {
	case strings.HasPrefix(phone, "+91") && len(phone) == 13:
		phone = phone[3:]
	case strings.HasPrefix(phone, "91") && len(phone) == 12:
		phone = phone[2:]
	case strings.HasPrefix(phone, "0") && len(phone) == 11:
		phone = phone[1:]
	}

	if len(phone) != 10 {
		return "", ErrInvalidPhone
	}
	for _, r := range phone {
		if !unicode.IsDigit(r) {
			return "", ErrInvalidPhone
		}
	}
	return phone, nil
}
