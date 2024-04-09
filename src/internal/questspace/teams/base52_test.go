package teams

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLinkIDToPath_IsShort(t *testing.T) {
	id := int64(5)
	res, err := LinkIDToPath(id)
	require.NoError(t, err)
	t.Log(res)
	assert.Less(t, len(res), 10)
	assert.GreaterOrEqual(t, len(res), 6)
}
