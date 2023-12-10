package hasher

import (
	"hash"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/xerrors"
)

type Hasher interface {
	HashString(string) (string, error)
	HasSameHash(pw, hash string) bool
}

var _ Hasher = BCryptHasher{}

type BCryptHasher struct {
	cost int
}

func NewBCryptHasher(cost int) BCryptHasher {
	return BCryptHasher{cost: cost}
}

func (h BCryptHasher) HashString(s string) (string, error) {
	inBytes := []byte(s)
	passwordHash, err := bcrypt.GenerateFromPassword(inBytes, h.cost)
	if err != nil {
		return "", xerrors.Errorf("failed to hash password: %w", err)
	}
	return string(passwordHash), nil
}

func (h BCryptHasher) HasSameHash(pw, hash string) bool {
	pwBytes, hashBytes := []byte(pw), []byte(hash)
	return bcrypt.CompareHashAndPassword(hashBytes, pwBytes) == nil
}

var _ Hasher = NopHasher{}

type NopHasher struct {
}

func NewNopHasher() NopHasher {
	return NopHasher{}
}

func (h NopHasher) HashString(s string) (string, error) {
	return s, nil
}

func (h NopHasher) HasSameHash(pw, hash string) bool {
	return pw == hash
}

func HashString(h hash.Hash, str string) string {
	h.Write([]byte(str))
	sum := h.Sum(nil)
	h.Reset()
	return string(sum)
}
