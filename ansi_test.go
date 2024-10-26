package rainbow_test

import (
	"fmt"
	"testing"

	"github.com/nerdwave-nick/rainbow"
)

func TestRainbow_NewAnsiMods(t *testing.T) {
	tests := []struct {
		Inputs []rainbow.AnsiAttr
		Output string
	}{
		{
			Inputs: []rainbow.AnsiAttr{rainbow.Fg.Black},
			Output: "\x1b[30m",
		},
		{
			Inputs: []rainbow.AnsiAttr{rainbow.Fg.White},
			Output: "\x1b[37m",
		},
		{
			Inputs: []rainbow.AnsiAttr{rainbow.Fg.Red, rainbow.Fmt.Bold, rainbow.Bg.HiCyan},
			Output: "\x1b[31;1;106m",
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("ansi mod test %d", i), func(t *testing.T) {
			retVal := rainbow.Mod(tt.Inputs...)
			if string(retVal) != tt.Output {
				t.Errorf("output %q did not match the expected output %q", retVal, tt.Output)
			}
		})
	}
}
