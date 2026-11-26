package is

import (
	"bytes"
	"strings"
	"testing"
)

// TestFormatStringEscapeInArguments checks that format verbs in the assertion
// source are not interpreted when the assertion is printed.
func TestFormatStringEscapeInArguments(t *testing.T) {
	tt := &mockT{}
	is := NewRelaxed(tt)
	var buf bytes.Buffer
	is.out = &buf
	is.colorful = false
	is.True("x%s" == "") // assertion with a format verb in the argument
	actual := buf.String()
	if strings.Contains(actual, `%!`) {
		t.Errorf("format verb was interpreted: %s", actual)
	}
	if !strings.Contains(actual, `"x%s" == ""`) {
		t.Errorf("assertion arguments were not printed verbatim: %s", actual)
	}
	if !tt.failed {
		t.Error("assertion did not fail")
	}
}
