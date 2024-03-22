package jwt

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"questspace/pkg/storage"
)

func TestTokenParser_NewlyCreatedIsValid(t *testing.T) {
	user := storage.User{
		Username: "svayp11",
	}
	secret := []byte{1, 2, 3}
	parser := NewTokenParser(secret)
	tk, err := parser.CreateToken(&user)
	require.NoError(t, err)
	got, err := parser.ParseToken(tk)
	require.NoError(t, err)
	assert.Equal(t, user, *got)
}
