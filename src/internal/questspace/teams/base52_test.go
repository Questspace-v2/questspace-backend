package teams

import (
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

var prevAlph = alphabet

func setTestDecimalAlphabet(t *testing.T) {
	alphabet = "0123456789"
	t.Cleanup(func() { alphabet = prevAlph })
}

func reverse(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for len(s) > 0 {
		r, sz := utf8.DecodeLastRuneInString(s)
		_, _ = b.WriteRune(r)
		s = s[:len(s)-sz]
	}
	return b.String()
}

func TestLinkIDToPath_DecimalAlphabet(t *testing.T) {
	setTestDecimalAlphabet(t)

	id := int64(1337)
	expected := reverse(strconv.Itoa(int(id)))
	res, err := LinkIDToPath(id)

	require.NoError(t, err)
	require.Equal(t, expected, res)
}

func TestLinkIDToPath_IsShort(t *testing.T) {
	id := int64(1 << 33)
	res, err := LinkIDToPath(id)
	require.NoError(t, err)
	require.Less(t, len(res), 10)
}
