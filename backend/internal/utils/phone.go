package utils

import (
	"errors"
	"strings"
)

var ErrInvalidPhone = errors.New("invalid phone number")
var ErrDuplicatePhone = errors.New("duplicate phone")

func NormalizePhone(phone string) (string, error) {
	phone = strings.TrimSpace(phone)
	phone = strings.ReplaceAll(phone, "-", "")
	phone = strings.ReplaceAll(phone, " ", "")
	if strings.HasPrefix(phone, "+91") {
		phone = phone[3:]
	} else if len(phone) == 12 && strings.HasPrefix(phone, "91") {
		phone = phone[2:]
	} else if len(phone) == 11 && strings.HasPrefix(phone, "0") {
		phone = phone[1:]
	}
	if len(phone) != 10 {
		return "", ErrInvalidPhone
	}
	return phone, nil
}
