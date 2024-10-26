package rainbow

import (
	"strings"
)

type AnsiMod string

type AnsiAttr string

const escape = '\x1b'

var Fmt = struct {
	Reset      AnsiAttr
	Bold       AnsiAttr
	Faint      AnsiAttr
	Italic     AnsiAttr
	Underline  AnsiAttr
	Blink      AnsiAttr
	CrossedOut AnsiAttr
}{
	Reset:      "0",
	Bold:       "1",
	Faint:      "2",
	Italic:     "3",
	Underline:  "4",
	Blink:      "5",
	CrossedOut: "9",
}

var Fg = struct {
	Black     AnsiAttr
	Red       AnsiAttr
	Green     AnsiAttr
	Yellow    AnsiAttr
	Blue      AnsiAttr
	Magenta   AnsiAttr
	Cyan      AnsiAttr
	White     AnsiAttr
	HiBlack   AnsiAttr
	HiRed     AnsiAttr
	HiGreen   AnsiAttr
	HiYellow  AnsiAttr
	HiBlue    AnsiAttr
	HiMagenta AnsiAttr
	HiCyan    AnsiAttr
	HiWhite   AnsiAttr
}{
	Black:     "30",
	Red:       "31",
	Green:     "32",
	Yellow:    "33",
	Blue:      "34",
	Magenta:   "35",
	Cyan:      "36",
	White:     "37",
	HiBlack:   "90",
	HiRed:     "91",
	HiGreen:   "92",
	HiYellow:  "93",
	HiBlue:    "94",
	HiMagenta: "95",
	HiCyan:    "96",
	HiWhite:   "97",
}

var Bg = struct {
	Black     AnsiAttr
	Red       AnsiAttr
	Green     AnsiAttr
	Yellow    AnsiAttr
	Blue      AnsiAttr
	Magenta   AnsiAttr
	Cyan      AnsiAttr
	White     AnsiAttr
	HiBlack   AnsiAttr
	HiRed     AnsiAttr
	HiGreen   AnsiAttr
	HiYellow  AnsiAttr
	HiBlue    AnsiAttr
	HiMagenta AnsiAttr
	HiCyan    AnsiAttr
	HiWhite   AnsiAttr
}{
	Black:     "40",
	Red:       "41",
	Green:     "42",
	Yellow:    "43",
	Blue:      "44",
	Magenta:   "45",
	Cyan:      "46",
	White:     "47",
	HiBlack:   "100",
	HiRed:     "101",
	HiGreen:   "102",
	HiYellow:  "103",
	HiBlue:    "104",
	HiMagenta: "105",
	HiCyan:    "106",
	HiWhite:   "107",
}

func Mod(attrs ...AnsiAttr) AnsiMod {
	if len(attrs) == 0 {
		return AnsiMod("")
	}
	sb := strings.Builder{}
	// 2 for the starters, then len(attr) * 2 since they're probably around 2 long and len(attr) for separator/ending
	sb.Grow(2 + len(attrs)*3)
	sb.WriteByte(escape)
	sb.WriteString("[")
	if len(attrs) > 0 {
		sb.WriteString(string(attrs[0]))
		for _, attr := range attrs[1:] {
			sb.WriteString(";")
			sb.WriteString(string(attr))
		}
	}
	sb.WriteString("m")
	return AnsiMod(sb.String())
}
