package teams

import (
	"math"
	"strings"

	"golang.org/x/xerrors"
)

const minLength = 6

var (
	alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

func LinkIDToPath(id int64) (string, error) {
	var b strings.Builder
	b.Grow(10)
	// NOTE(svayp11): Cheap way to increase minimum length
	modifier := int64(math.Pow(float64(len(alphabet)), minLength))
	id += modifier

	for ; id > 0; id /= int64(len(alphabet)) {
		idx := id % int64(len(alphabet))
		if err := b.WriteByte(alphabet[idx]); err != nil {
			return "", xerrors.Errorf("write %q: %w", string(alphabet[idx]), err)
		}
	}
	return b.String(), nil
}
