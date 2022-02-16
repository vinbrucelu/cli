package text

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWrapWordsFromString(t *testing.T) {
	for i, c := range []struct {
		words    string
		afterN   int
		tab      bool
		expected string
	}{
		{
			"a b  c d e \n ",
			2,
			false,
			"a b\nc d\ne",
		},
		{
			"a b c d e f",
			3,
			false,
			"a b c\nd e f",
		},
		{
			"a b c d e f",
			6,
			false,
			"a b c d e f",
		},
		{
			"a b",
			3,
			false,
			"a b",
		},
		{
			"a b c d e f",
			2,
			true,
			"a b\n\tc d\n\te f",
		},
	} {
		require.Equal(t, c.expected, WrapWordsFromString(c.words, c.afterN, c.tab), i)
	}
}
