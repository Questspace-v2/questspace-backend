package jwt

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/yandex/perforator/library/go/core/xerrors"

	"questspace/pkg/storage"
)

const usedAlg = "HS256"

//go:generate mockgen -source=token.go -destination mocks/token.go -package mocks
type TokenVendingMachine interface {
	TokenEncoder
	TokenParser
}

type TokenParser interface {
	ParseToken(tokenStr string) (*storage.User, error)
}

type TokenEncoder interface {
	CreateToken(user *storage.User) (string, error)
}

type questspaceClaims struct {
	Admin  bool   `json:"admin"`
	Avatar string `json:"avatar"`

	jwt.RegisteredClaims
}

type VendingMachine struct {
	secret []byte
}

func NewTokenParser(sec []byte) *VendingMachine {
	return &VendingMachine{secret: sec}
}

func (p *VendingMachine) ParseToken(tokenStr string) (*storage.User, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &questspaceClaims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method.Alg() != usedAlg {
			return nil, xerrors.Errorf("unexpected signing method: %v", t.Method.Alg())
		}

		return p.secret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*questspaceClaims); ok {
		return &storage.User{
			ID:        storage.ID(claims.ID),
			Username:  claims.Issuer,
			AvatarURL: claims.Avatar,
		}, nil
	}
	return nil, xerrors.New("invalid token")
}

func (p *VendingMachine) CreateToken(user *storage.User) (string, error) {
	claims := questspaceClaims{
		Admin:  false, // TODO(svayp11): Implement admin role
		Avatar: user.AvatarURL,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:     user.ID.String(),
			Issuer: user.Username,
		},
	}

	token := jwt.NewWithClaims(jwt.GetSigningMethod(usedAlg), claims)
	ss, err := token.SignedString(p.secret)
	if err != nil {
		return "", xerrors.Errorf("issue new token: %w", err)
	}
	return ss, nil
}

type nopParser struct {
	User  *storage.User
	Token string
}

func NewNopParser(neededUser *storage.User, neededToken string) TokenVendingMachine {
	return &nopParser{User: neededUser, Token: neededToken}
}

func (n nopParser) ParseToken(_ string) (*storage.User, error) {
	return n.User, nil
}

func (n nopParser) CreateToken(_ *storage.User) (string, error) {
	return n.Token, nil
}
