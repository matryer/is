package example

import (
	"testing"

	"github.com/matryer/is"
)

func Test(t *testing.T) {
	is := is.NewRelaxed(t)

	is.Equal(add(2, 4), 6)  // simple addition
	is.Equal(add(5, -1), 4) // subtracting
	is.OK(add(10, 1) < 10)  // surely?
}
