package domain_test

import (
	"strings"
	"testing"

	"github.com/m0090-dev/eec/internal/ext/utils/domain"
)

func TestGenUnixTagUtilsScript_NotEmptyAndContainsTag(t *testing.T) {
	script := domain.GenUnixTagUtilsScript("dev")
	if script == "" {
		t.Fatal("GenUnixTagUtilsScript returned an empty script")
	}
	for _, want := range []string{"#!/bin/sh", "--tag dev", "--program", "--program-args"} {
		if !strings.Contains(script, want) {
			t.Errorf("script should contain %q, got:\n%s", want, script)
		}
	}
	if strings.Contains(script, "%TAGNAME%") {
		t.Error("tag placeholder was not replaced")
	}
}

func TestGenUnixTagUtilsScriptProgramFixed_NotEmptyAndContainsTag(t *testing.T) {
	script := domain.GenUnixTagUtilsScriptProgramFixed("dev")
	if script == "" {
		t.Fatal("GenUnixTagUtilsScriptProgramFixed returned an empty script")
	}
	for _, want := range []string{"#!/bin/sh", "--tag dev", "--program-args"} {
		if !strings.Contains(script, want) {
			t.Errorf("script should contain %q, got:\n%s", want, script)
		}
	}
	if strings.Contains(script, "%TAGNAME%") {
		t.Error("tag placeholder was not replaced")
	}
}
