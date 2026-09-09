package utils

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func withSecret(t *testing.T, secret string) {
	t.Helper()
	t.Setenv("JWT_SECRET", secret)
}

func TestGenerateAndValidateToken(t *testing.T) {
	withSecret(t, "test-secret")

	token, err := GenerateToken(42, 7)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	userID, version, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if userID != 42 {
		t.Errorf("userID = %d, want 42", userID)
	}
	if version != 7 {
		t.Errorf("tokenVersion = %d, want 7", version)
	}
}

func TestValidateTokenRejectsBadInput(t *testing.T) {
	withSecret(t, "test-secret")

	valid, err := GenerateToken(1, 0)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	tests := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"garbage", "not-a-token"},
		{"wrong segment count", "a.b"},
		{"tampered signature", valid[:len(valid)-4] + "AAAA"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := ValidateToken(tc.token); err == nil {
				t.Fatalf("ValidateToken(%q) succeeded, want error", tc.name)
			}
		})
	}
}

// A token signed with a different key must not validate.
func TestValidateTokenRejectsForeignSignature(t *testing.T) {
	withSecret(t, "secret-a")
	token, err := GenerateToken(1, 0)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}

	withSecret(t, "secret-b")
	if _, _, err := ValidateToken(token); err == nil {
		t.Fatal("token signed with a different secret was accepted")
	}
}

// alg:none is the classic JWT bypass: the parser must not treat an unsigned
// token as valid just because the header asks it to.
func TestValidateTokenRejectsAlgNone(t *testing.T) {
	withSecret(t, "test-secret")

	enc := func(v interface{}) string {
		b, _ := json.Marshal(v)
		return base64.RawURLEncoding.EncodeToString(b)
	}
	forged := enc(map[string]string{"alg": "none", "typ": "JWT"}) + "." +
		enc(map[string]interface{}{"user_id": 1, "exp": time.Now().Add(time.Hour).Unix()}) + "."

	if _, _, err := ValidateToken(forged); err == nil {
		t.Fatal("alg:none token was accepted")
	}
}

// Without WithExpirationRequired, jwt.Parse only checks exp when it is present,
// so a validly signed token that omits it would never expire.
func TestValidateTokenRequiresExpiry(t *testing.T) {
	withSecret(t, "test-secret")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":       1,
		"token_version": 0,
		// deliberately no "exp"
	})
	signed, err := token.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("signing: %v", err)
	}

	if _, _, err := ValidateToken(signed); err == nil {
		t.Fatal("token without an exp claim was accepted")
	}
}

func TestValidateTokenRejectsExpired(t *testing.T) {
	withSecret(t, "test-secret")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UserID: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
		},
	})
	signed, err := token.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("signing: %v", err)
	}

	if _, _, err := ValidateToken(signed); err == nil {
		t.Fatal("expired token was accepted")
	}
}

func TestTokenFunctionsRequireSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "")

	if _, err := GenerateToken(1, 0); err == nil {
		t.Error("GenerateToken succeeded without JWT_SECRET")
	}
	if _, _, err := ValidateToken("anything"); err == nil {
		t.Error("ValidateToken succeeded without JWT_SECRET")
	}
}
