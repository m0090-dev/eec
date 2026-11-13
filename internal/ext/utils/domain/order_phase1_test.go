package domain_test

import (
	"testing"

	"github.com/m0090-dev/eec/internal/ext/types"
	"github.com/m0090-dev/eec/internal/ext/utils/domain"
)

func TestResolveRunOptions_InvalidInlineConfig_PropagatesError(t *testing.T) {
	rt := newMockRuntime()
	opts := types.RunOptions{
		InlineConfigToml: "[envs\nthis is not valid toml =",
	}
	_, _, _, _, _, err := domain.ResolveRunOptions(opts, types.TagData{}, rt)
	if err == nil {
		t.Error("expected inline config parse error to be propagated, got nil")
	}
}
