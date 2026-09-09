package utils

import (
	"errors"
	"testing"
)

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr error
	}{
		{"plain ten digits", "9876543210", "9876543210", nil},
		{"country code with plus", "+919876543210", "9876543210", nil},
		{"country code without plus", "919876543210", "9876543210", nil},
		{"national trunk prefix", "09876543210", "9876543210", nil},
		{"dashes stripped", "987-654-3210", "9876543210", nil},
		{"spaces stripped", "98765 43210", "9876543210", nil},
		{"mixed separators with country code", "+91 98765-43210", "9876543210", nil},
		{"parentheses stripped", "(98765)43210", "9876543210", nil},
		{"surrounding whitespace", "  9876543210  ", "9876543210", nil},

		{"empty", "", "", ErrInvalidPhone},
		{"too short", "98765", "", ErrInvalidPhone},
		{"too long", "98765432101", "", ErrInvalidPhone},
		{"letters rejected", "98765abcde", "", ErrInvalidPhone},
		{"bare plus", "+", "", ErrInvalidPhone},
		{"plus alone with country code", "+91", "", ErrInvalidPhone},
		{"absurdly long input", "+9199999999999999999999999999999999999", "", ErrInvalidPhone},
		// A leading 0 on an already-10-digit number is significant, not a trunk
		// prefix: stripping it unconditionally used to corrupt the number.
		{"leading zero on ten digits is kept", "0987654321", "0987654321", nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizePhone(tc.input)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("NormalizePhone(%q) error = %v, want %v", tc.input, err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("NormalizePhone(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// Normalizing an already-normalized number must not change it, or repeated
// edits would slowly rewrite stored data.
func TestNormalizePhoneIsIdempotent(t *testing.T) {
	for _, input := range []string{"+91 98765-43210", "09876543210", "9876543210"} {
		once, err := NormalizePhone(input)
		if err != nil {
			t.Fatalf("first pass on %q: %v", input, err)
		}
		twice, err := NormalizePhone(once)
		if err != nil {
			t.Fatalf("second pass on %q: %v", once, err)
		}
		if once != twice {
			t.Errorf("not idempotent for %q: %q then %q", input, once, twice)
		}
	}
}
