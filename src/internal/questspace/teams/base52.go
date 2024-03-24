package teams

import (
	"strings"

	"golang.org/x/xerrors"
)

var (
	alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

func LinkIDToPath(id int64) (string, error) {
	var b strings.Builder
	b.Grow(10)
	for ; id > 0; id /= int64(len(alphabet)) {
		idx := id % int64(len(alphabet))
		if err := b.WriteByte(alphabet[idx]); err != nil {
			return "", xerrors.Errorf("write %q: %w", string(alphabet[idx]), err)
		}
	}
	return b.String(), nil
}
