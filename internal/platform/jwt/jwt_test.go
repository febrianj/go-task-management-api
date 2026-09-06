package jwt

import (
	"strings"
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testSecret = "32-char-long-secret-hahahahahaha"

func TestIssueAndVerify(t *testing.T) {
	iss := NewIssuer(testSecret, time.Hour)
	userID, teamID := uuid.New(), uuid.New()

	token, expiresAt, err := iss.Issue(userID, teamID)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if !expiresAt.After(time.Now()) {
		t.Fatalf("expiry must be later than current time")
	}

	claims, err := iss.Verify(token)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if claims.UserID != userID {
		t.Fatalf("user id: got %v instead of %v", claims.UserID, userID)
	}
	if claims.TeamID != teamID {
		t.Fatalf("team id: got %v instead of %v", claims.TeamID, teamID)
	}
}

func TestVerifyRejectsTamperedToken(t *testing.T) {
	iss := NewIssuer(testSecret, time.Hour)
	token, _, _ := iss.Issue(uuid.New(), uuid.New())

	parts := strings.Split(token, ".")
	parts[1] = "X" + parts[1][1:]

	if _, err := iss.Verify(strings.Join(parts, ".")); err == nil {
		t.Fatal("tampered token must be rejected")
	}
}

func TestVerifyRejectsExpiredToken(t *testing.T) {
	iss := NewIssuer(testSecret, -time.Hour)
	token, _, _ := iss.Issue(uuid.New(), uuid.New())

	if _, err := iss.Verify(token); err == nil {
		t.Fatal("expired token must be rejected")
	}
}

func TestVerifyRejectsWrongSecret(t *testing.T) {
	token, _, _ := NewIssuer(testSecret, time.Hour).Issue(uuid.New(), uuid.New())
	other := NewIssuer("32-char-long-secret-hahahahahahi", time.Hour)

	if _, err := other.Verify(token); err == nil {
		t.Fatal("different secret must be rejected")
	}
}

func TestVerifyRejectsUnexpectedAlgorithm(t *testing.T) {
	claims := jwtlib.MapClaims{
		"sub":     uuid.New().String(),
		"team_id": uuid.New().String(),
		"iat":     time.Now().Unix(),
		"exp":     time.Now().Add(time.Hour).Unix(),
	}

	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS512, claims)
	signed, err := token.SignedString([]byte(testSecret))

	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if _, err := NewIssuer(testSecret, time.Hour).Verify(signed); err == nil {
		t.Fatal("token signed with different alg must be rejected")
	}
}
