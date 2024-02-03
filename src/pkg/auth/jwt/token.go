package jwt

import (
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/xerrors"

	"questspace/pkg/storage"
)

const usedAlg = "HS256"

type Parser interface {
	ParseToken(tokenStr string) (*storage.User, error)
	CreateToken(user *storage.User) (string, error)
}

type questspaceClaims struct {
	Admin  bool   `json:"admin"`
	Avatar string `json:"avatar"`

	jwt.RegisteredClaims
}

type parser struct {
	secret []byte
}

func NewParser(sec []byte) Parser {
	return &parser{secret: sec}
}

func (p parser) ParseToken(tokenStr string) (*storage.User, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &questspaceClaims{}, func(t *jwt.Token) (interface{}, error) {
		if t.Method.Alg() != usedAlg {
			return nil, xerrors.Errorf("unexpected signing method: %v", t.Method.Alg())
		}

		return p.secret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(questspaceClaims); ok {
		return &storage.User{
			Id:        claims.ID,
			Username:  claims.Issuer,
			AvatarURL: claims.Avatar,
		}, nil
	}
	return nil, xerrors.New("invalid token")
}

func (p parser) CreateToken(user *storage.User) (string, error) {
	claims := questspaceClaims{
		Admin:  false, // TODO(svayp11): Implement admin role
		Avatar: user.AvatarURL,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:     user.Id,
			Issuer: user.Username,
		},
	}

	token := jwt.NewWithClaims(jwt.GetSigningMethod(usedAlg), claims)
	ss, err := token.SignedString(p.secret)
	if err != nil {
		return "", xerrors.Errorf("failed to issue new token: %w", err)
	}
	return ss, nil
}

type nopParser struct {
	User  *storage.User
	Token string
}

func NewNopParser(neededUser *storage.User, neededToken string) Parser {
	return &nopParser{User: neededUser, Token: neededToken}
}

func (n nopParser) ParseToken(_ string) (*storage.User, error) {
	return n.User, nil
}

func (n nopParser) CreateToken(_ *storage.User) (string, error) {
	return n.Token, nil
}
