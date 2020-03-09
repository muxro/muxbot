package bot

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseCmdString(t *testing.T) {
	tests := []struct {
		cmd  string
		args []string
		pos  []int
	}{
		{
			cmd:  `hello world`,
			args: []string{"hello", "world"},
			pos:  []int{0, 6},
		},
		{
			cmd:  `"hello world" "arg 2"`,
			args: []string{"hello world", "arg 2"},
			pos:  []int{0, 14},
		},
		{
			cmd:  `"'hello world"`,
			args: []string{"'hello world"},
			pos:  []int{0},
		},
		{
			cmd:  `"\"hello world"`,
			args: []string{"\"hello world"},
			pos:  []int{0},
		},

		{
			cmd:  `hello      world`,
			args: []string{"hello", "world"},
			pos:  []int{0, 11},
		},
	}

	for _, tc := range tests {
		args, pos := parseCmdString(tc.cmd)
		require.Equal(t, tc.args, args)
		require.Equal(t, tc.pos, pos)
	}
}
