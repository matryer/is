package example

// CAUTION: DO NOT EDIT
// Tests in this project rely on specific lines numbers
// throughout this file.

import (
	"testing"
)

func TestSomething(t *testing.T) {
	// this comment will be extracted
}

func TestSomethingElse(t *testing.T) {
	a, b := 1, 2
	getB = func() int {
		return b
	}
	is.True(a == getB()) // should be the same
}

func TestSomethingElseTpp(t *testing.T) {
	a, b := 1, 2
	getB = func() int {
		return b
	}
	is.True(a == getB())
}
