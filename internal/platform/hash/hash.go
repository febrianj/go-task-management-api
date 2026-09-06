package hash

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type Bcrypt struct {
	cost int
}

func NewBcrypt(cost int) *Bcrypt {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		cost = bcrypt.DefaultCost
	}
	return &Bcrypt{cost: cost}
}

func (b *Bcrypt) Hash(plain string) (string, error) {
	if len(plain) > 72 {
		return "", fmt.Errorf("password exceed 72 bytes")
	}
	out, err := bcrypt.GenerateFromPassword([]byte(plain), b.cost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}

	return string(out), nil
}

// return nil when password match
func (b *Bcrypt) Compare(hashed, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain))
}
