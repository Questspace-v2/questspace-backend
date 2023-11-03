package hasher

import "hash"

func HashString(h hash.Hash, str string) string {
	h.Write([]byte(str))
	sum := h.Sum(nil)
	h.Reset()
	return string(sum)
}
