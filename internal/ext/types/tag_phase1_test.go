package types_test

import (
	"testing"

	"github.com/m0090-dev/eec/internal/ext/types"
)

func TestValidateTagName(t *testing.T) {
	valid := []string{"dev", "win-dev", "my.tag.v2", "tag_1"}
	for _, name := range valid {
		if err := types.ValidateTagName(name); err != nil {
			t.Errorf("ValidateTagName(%q) = %v, want nil", name, err)
		}
	}
	invalid := []string{"", "..", "../evil", "a/b", `a\b`, "a..b", "/abs", "x/"}
	for _, name := range invalid {
		if err := types.ValidateTagName(name); err == nil {
			t.Errorf("ValidateTagName(%q) = nil, want error", name)
		}
	}
}

func TestTagDataWrite_InvalidName_ReturnsError(t *testing.T) {
	rt := newMockRuntime()
	tag := types.TagData{Program: "prog"}
	if err := tag.Write(rt, "../evil"); err == nil {
		t.Error("expected error writing tag with traversal name, got nil")
	}
}

func TestReadTagData_InvalidName_ReturnsError(t *testing.T) {
	rt := newMockRuntime()
	if _, err := types.ReadTagData(rt, "../evil"); err == nil {
		t.Error("expected error reading tag with traversal name, got nil")
	}
	if _, err := types.ReadTagData(rt, ""); err == nil {
		t.Error("expected error reading tag with empty name, got nil")
	}
}
