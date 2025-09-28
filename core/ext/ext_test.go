package ext_test

import (
	"fmt"
	"testing"

	"github.com/m0090-dev/eec-go/core/ext"
)

func TestExtLogger(t *testing.T) {
	l := ext.NewDefaultLogger()
	l.Debug().Err(fmt.Errorf("Error!")).Msg("Test Message ")
}
