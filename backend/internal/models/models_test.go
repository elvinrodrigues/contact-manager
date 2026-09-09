package models

import (
	"errors"
	"strings"
	"testing"
)

func ptr[T any](v T) *T { return &v }

func TestContactNormalizeAndValidate(t *testing.T) {
	tests := []struct {
		name      string
		contact   Contact
		wantErr   error
		checkFunc func(*testing.T, Contact)
	}{
		{
			name:    "trims surrounding whitespace",
			contact: Contact{Name: "  Ada Lovelace  "},
			checkFunc: func(t *testing.T, c Contact) {
				if c.Name != "Ada Lovelace" {
					t.Errorf("name = %q, want %q", c.Name, "Ada Lovelace")
				}
			},
		},
		{
			name:    "rejects empty name",
			contact: Contact{Name: ""},
			wantErr: ErrNameRequired,
		},
		{
			// The update path used to accept this where create rejected it.
			name:    "rejects whitespace-only name",
			contact: Contact{Name: "   "},
			wantErr: ErrNameRequired,
		},
		{
			name:    "rejects an over-long name",
			contact: Contact{Name: strings.Repeat("x", MaxNameLen+1)},
			wantErr: ErrNameTooLong,
		},
		{
			name:    "accepts a name at the limit",
			contact: Contact{Name: strings.Repeat("x", MaxNameLen)},
		},
		{
			name:    "lowercases and trims email",
			contact: Contact{Name: "Ada", Email: ptr("  Ada@Example.COM ")},
			checkFunc: func(t *testing.T, c Contact) {
				if c.Email == nil || *c.Email != "ada@example.com" {
					t.Errorf("email = %v, want ada@example.com", c.Email)
				}
			},
		},
		{
			name:    "rejects a malformed email",
			contact: Contact{Name: "Ada", Email: ptr("not-an-email")},
			wantErr: ErrEmailInvalid,
		},
		{
			// An empty string is a missing value, not a value. Storing it as ""
			// made NULL and "no email" indistinguishable.
			name:    "blank email becomes nil",
			contact: Contact{Name: "Ada", Email: ptr("   ")},
			checkFunc: func(t *testing.T, c Contact) {
				if c.Email != nil {
					t.Errorf("email = %v, want nil", *c.Email)
				}
			},
		},
		{
			name:    "nil email stays nil",
			contact: Contact{Name: "Ada", Email: nil},
			checkFunc: func(t *testing.T, c Contact) {
				if c.Email != nil {
					t.Errorf("email = %v, want nil", *c.Email)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := tc.contact
			err := c.NormalizeAndValidate()
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, want %v", err, tc.wantErr)
			}
			if tc.checkFunc != nil {
				tc.checkFunc(t, c)
			}
		})
	}
}

func TestUpdateContactInputIsEmpty(t *testing.T) {
	if !(UpdateContactInput{}).IsEmpty() {
		t.Error("a fully nil input should be empty")
	}
	if (UpdateContactInput{Name: ptr("x")}).IsEmpty() {
		t.Error("an input with a name should not be empty")
	}
	// A pointer to the empty string means "clear this field", which is a change.
	if (UpdateContactInput{Email: ptr("")}).IsEmpty() {
		t.Error("an input clearing the email should not be empty")
	}
}
