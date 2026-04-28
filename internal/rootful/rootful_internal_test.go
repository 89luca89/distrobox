package rootful

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetState() {
	validateOnce = sync.Once{}
	errValidate = nil
	cachedSudoCommand = ""
}

func TestValidate_Success(t *testing.T) {
	resetState()
	t.Cleanup(resetState)

	err := Validate(context.Background(), "true")
	require.NoError(t, err)
}

func TestValidate_Failure(t *testing.T) {
	resetState()
	t.Cleanup(resetState)

	err := Validate(context.Background(), "false")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `"false"`)
}

func TestValidate_CachesResult(t *testing.T) {
	resetState()
	t.Cleanup(resetState)

	require.NoError(t, Validate(context.Background(), "true"))
	require.NoError(t, Validate(context.Background(), "true"))
}

func TestValidate_MismatchedCommand(t *testing.T) {
	resetState()
	t.Cleanup(resetState)

	require.NoError(t, Validate(context.Background(), "true"))

	err := Validate(context.Background(), "false")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mismatch")
	assert.Contains(t, err.Error(), `"true"`)
	assert.Contains(t, err.Error(), `"false"`)
}
