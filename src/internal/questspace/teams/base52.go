package teams

import (
	"math"
	"strings"

	"github.com/yandex/perforator/library/go/core/xerrors"
)

const minLength = 6

var (
	alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

func LinkIDToPath(id int64) (string, error) {
	var b strings.Builder
	b.Grow(10)
	// NOTE(svayp11): Cheap way to increase minimum length
	modifier := getStartConst()
	id += modifier

	for ; id > 0; id /= int64(len(alphabet)) {
		idx := id % int64(len(alphabet))
		if err := b.WriteByte(alphabet[idx]); err != nil {
			return "", xerrors.Errorf("write %q: %w", string(alphabet[idx]), err)
		}
	}
	return b.String(), nil
}

func getStartConst() int64 {
	initialModifier := int64(math.Pow(float64(len(alphabet)), minLength-2))
	initialModifier = initialModifier*int64(len(alphabet)*5/6) + initialModifier*int64(len(alphabet))/9
	return initialModifier * 31
}
