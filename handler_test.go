package rainbow_test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"testing"
	"time"

	"github.com/nerdwave-nick/rainbow"
)

func attrRE(attr *slog.Attr, opts *rainbow.Options, pgr string) string {
	if attr.Value.Kind() != slog.KindGroup {
		attr.Value = attr.Value.Resolve()
	}
	rstring := pgr + keyRE(attr.Key, opts)
	switch attr.Value.Kind() {
	case slog.KindString:
		return rstring + string(opts.ValueOverrides.String) + "\\\"" + attr.Value.String() + "\\\"" + string(opts.ResetOverride)
	case slog.KindInt64:
		return rstring + string(opts.ValueOverrides.Int) + fmt.Sprintf("%d", attr.Value.Int64()) + string(opts.ResetOverride)
	case slog.KindBool:
		return rstring + string(opts.ValueOverrides.Bool) + fmt.Sprintf("%t", attr.Value.Bool()) + string(opts.ResetOverride)
	case slog.KindFloat64:
		return rstring + string(opts.ValueOverrides.Float) + fmt.Sprintf("%g", attr.Value.Float64()) + string(opts.ResetOverride)
	case slog.KindDuration:
		return rstring + string(opts.ValueOverrides.Duration) + attr.Value.Duration().String() + string(opts.ResetOverride)
	case slog.KindTime:
		return rstring + string(opts.ValueOverrides.Time) + attr.Value.Time().Format("2006-01-02T15:04:05.000") + string(opts.ResetOverride)
	case slog.KindAny:
		errVal, ok := attr.Value.Any().(error)
		if ok {
			return rstring + string(opts.ValueOverrides.Error) + errVal.Error() + string(opts.ResetOverride)
		}
		return rstring + string(opts.ValueOverrides.Any) + fmt.Sprintf("%v", attr.Value.Any()) + string(opts.ResetOverride)
	case slog.KindGroup:
		group := attr.Value.Group()
		grK := pgr + grRE(attr.Key, opts)
		grS := "" // "THIS IS A GROUP WITH KEY " + attr.Key
		grLen := len(group)
		for i, attr := range group {
			grS = grS + attrRE(&attr, opts, grK)
			if i < grLen-1 {
				grS = grS + string(opts.SymbolOverride) + opts.AttrAttrSeparator + string(opts.ResetOverride)
			}
		}
		return grS

	default:
		return ""
	}
}

func messageRE(msg string, opts *rainbow.Options) string {
	return string(opts.SpecialOverrides.Message) + msg + string(opts.ResetOverride)
}

func keyRE(key string, opts *rainbow.Options) string {
	col, ok := opts.KeyOverrides.KeyMap[key]
	if !ok {
		col = opts.KeyOverrides.Default
	}
	return string(col) + key + string(opts.ResetOverride) + string(opts.SymbolOverride) + "=" + string(opts.ResetOverride)
}

func grRE(key string, opts *rainbow.Options) string {
	col, ok := opts.KeyOverrides.GroupMap[key]
	if !ok {
		col = opts.KeyOverrides.Default
	}
	return string(col) + key + string(opts.ResetOverride) + string(opts.SymbolOverride) + "." + string(opts.ResetOverride)
}

func makeRegex(opts *rainbow.Options, level slog.Leveler, preGroup []string, message string, attrs ...slog.Attr) *regexp.Regexp {
	levelStr := ""
	switch level.Level() {
	case slog.LevelDebug:
		levelStr = logLevelDebugRE(opts)
	case slog.LevelInfo:
		levelStr = logLevelInfoRE(opts)
	case slog.LevelWarn:
		levelStr = logLevelWarnRE(opts)
	case slog.LevelError:
		levelStr = logLevelErrRE(opts)
	}
	reStr := "^" + dateRE(opts) + levelStr + messageRE(message, opts)
	grStr := ""
	for _, g := range preGroup {
		grStr = grStr + grRE(g, opts)
	}

	numAttrs := len(attrs)
	if numAttrs > 0 {
		reStr = reStr + string(opts.SymbolOverride) + opts.MessageAttrSeparator + string(opts.ResetOverride)
	}
	for i, extra := range attrs {
		reStr = reStr + attrRE(&extra, opts, grStr)
		if i < numAttrs-1 {
			reStr = reStr + string(opts.SymbolOverride) + opts.AttrAttrSeparator + string(opts.ResetOverride)
		}
	}

	return regexp.MustCompile(reStr + `\n$`)
}

func dateRE(opts *rainbow.Options) string {
	return string(opts.SpecialOverrides.Time + "[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\\.[0-9]{3}" + opts.ResetOverride)
}

func logLevelDebugRE(opts *rainbow.Options) string {
	return string(opts.LevelOverrides.Debug + "\\|DBG " + opts.ResetOverride)
}

func logLevelInfoRE(opts *rainbow.Options) string {
	return string(opts.LevelOverrides.Info + "\\|INF " + opts.ResetOverride)
}

func logLevelWarnRE(opts *rainbow.Options) string {
	return string(opts.LevelOverrides.Warning + "\\|WRN " + opts.ResetOverride)
}

func logLevelErrRE(opts *rainbow.Options) string {
	return string(opts.LevelOverrides.Error + "\\|ERR " + opts.ResetOverride)
}

var opts = rainbow.Options{
	Level:                slog.LevelDebug,
	NoColor:              false,
	MessageAttrSeparator: "<mas>",
	AttrAttrSeparator:    "<aas>",
	ResetOverride:        "<ro>",
	SymbolOverride:       "<so>",
	LevelOverrides: &rainbow.LevelColorOverrides{
		Error:   "<le>",
		Warning: "<lw>",
		Debug:   "<ld>",
		Info:    "<li>",
	},
	ValueOverrides: &rainbow.ValueColorOverrides{
		String:   "<vs>",
		Int:      "<vi>",
		Float:    "<vf>",
		Uint:     "<vu>",
		Error:    "<ve>",
		Time:     "<vt>",
		Bool:     "<vb>",
		Duration: "<vd>",
		Any:      "<va>",
	},
	KeyOverrides: &rainbow.KeyColorOverrides{
		Default: "<kd>",
		KeyMap: map[string]rainbow.AnsiMod{
			"err":   "<ke>",
			"error": "<keo>",
		},
		GroupMap: map[string]rainbow.AnsiMod{
			"gr": "<tgr>",
		},
	},
	SpecialOverrides: &rainbow.SpecialColorOverrides{
		Time:    "<t>",
		Message: "<m>",
	},
}

func TestRainbow_Handler(t *testing.T) {
	tests := []struct {
		LogLevel slog.Leveler
		Message  string
		Attrs    []slog.Attr
	}{
		{
			LogLevel: slog.LevelDebug,
			Message:  "Test Debug Message",
		},
		{
			LogLevel: slog.LevelInfo,
			Message:  "Test Info Message",
		},
		{
			LogLevel: slog.LevelWarn,
			Message:  "Test Warn Message",
		},
		{
			LogLevel: slog.LevelError,
			Message:  "Test Error Message",
		},
		{
			LogLevel: slog.LevelError,
			Message:  "Testing Attributes",
			Attrs: []slog.Attr{
				slog.String("some", "attribute"),
				slog.Int64("i64k", int64(23)),
				slog.Int("ik", 23),
				slog.Bool("bk", true),
				slog.Float64("fk", 324.2),
				slog.Duration("dk", 12*time.Second),
				slog.Time("tk", time.Unix(1, 1000000)),
				slog.Any("err", fmt.Errorf("err")),
				slog.Any("rk", struct {
					a int
					b string
				}{1, "2"}),
			},
		},
		{
			LogLevel: slog.LevelError,
			Message:  "Testing Attributes",
			Attrs: []slog.Attr{
				slog.Group("gr",
					slog.String("some", "attribute"),
					slog.Int64("i64k", int64(23)),
					slog.Int("ik", 23),
					slog.Bool("bk", true),
					slog.Float64("fk", 324.2),
					slog.Duration("dk", 12*time.Second),
					slog.Time("tk", time.Unix(1, 1000000)),
					slog.Any("err", fmt.Errorf("err")),
					slog.Any("rk", struct {
						a int
						b string
					}{1, "2"}),
				),
			},
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("handler test %d", i), func(t *testing.T) {
			t.Parallel()
			buffer := bytes.NewBuffer(make([]byte, 0))
			logger := slog.New(rainbow.New(buffer, &opts))

			logger.LogAttrs(context.Background(), tt.LogLevel.Level(), tt.Message, tt.Attrs...)
			expectedOutput := makeRegex(&opts, tt.LogLevel.Level(), []string{}, tt.Message, tt.Attrs...)
			output := buffer.String()
			if !expectedOutput.Match(buffer.Bytes()) {
				t.Errorf("output \n%q did not match the expected output regex \n%s", output, expectedOutput.String())
			}
		})
	}
}

func TestRainbow_HandlerWithAttrs(t *testing.T) {
	tests := []struct {
		LogLevel slog.Leveler
		Message  string
		Attrs    []slog.Attr
	}{
		{
			LogLevel: slog.LevelDebug,
			Message:  "Test Debug Message",
		},
		{
			LogLevel: slog.LevelInfo,
			Message:  "Test Info Message",
		},
		{
			LogLevel: slog.LevelWarn,
			Message:  "Test Warn Message",
		},
		{
			LogLevel: slog.LevelError,
			Message:  "Test Error Message",
		},
		{
			LogLevel: slog.LevelError,
			Message:  "Testing Attributes",
			Attrs: []slog.Attr{
				slog.String("some", "attribute"),
				slog.Int64("i64k", int64(23)),
				slog.Int("ik", 23),
				slog.Bool("bk", true),
				slog.Float64("fk", 324.2),
				slog.Duration("dk", 12*time.Second),
				slog.Time("tk", time.Unix(1, 1000000)),
				slog.Any("err", fmt.Errorf("err")),
				slog.Any("rk", struct {
					a int
					b string
				}{1, "2"}),
			},
		},
		{
			LogLevel: slog.LevelError,
			Message:  "Testing Attributes",
			Attrs: []slog.Attr{
				slog.Group("gr",
					slog.String("some", "attribute"),
					slog.Int64("i64k", int64(23)),
					slog.Int("ik", 23),
					slog.Bool("bk", true),
					slog.Float64("fk", 324.2),
					slog.Duration("dk", 12*time.Second),
					slog.Time("tk", time.Unix(1, 1000000)),
					slog.Any("err", fmt.Errorf("err")),
					slog.Any("rk", struct {
						a int
						b string
					}{1, "2"}),
				),
			},
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("handler with attr test %d", i), func(t *testing.T) {
			t.Parallel()
			buffer := bytes.NewBuffer(make([]byte, 0))
			logger := slog.New(rainbow.New(buffer, &opts)).With(slog.String("wa", "wav"))

			logger.LogAttrs(context.Background(), tt.LogLevel.Level(), tt.Message, tt.Attrs...)

			tt.Attrs = append([]slog.Attr{slog.String("wa", "wav")}, tt.Attrs...)
			expectedOutput := makeRegex(&opts, tt.LogLevel.Level(), []string{}, tt.Message, tt.Attrs...)
			output := buffer.String()
			if !expectedOutput.Match(buffer.Bytes()) {
				t.Errorf("output \n%q did not match the expected output regex \n%s", output, expectedOutput.String())
			}
		})
	}
}

func TestRainbow_HandlerWithGroup(t *testing.T) {
	tests := []struct {
		LogLevel slog.Leveler
		Message  string
		Attrs    []slog.Attr
	}{
		{
			LogLevel: slog.LevelError,
			Message:  "Testing Attributes",
			Attrs: []slog.Attr{
				slog.String("some", "attribute"),
				slog.Int64("i64k", int64(23)),
				slog.Int("ik", 23),
				slog.Bool("bk", true),
				slog.Float64("fk", 324.2),
				slog.Duration("dk", 12*time.Second),
				slog.Time("tk", time.Unix(1, 1000000)),
				slog.Any("err", fmt.Errorf("err")),
				slog.Any("rk", struct {
					a int
					b string
				}{1, "2"}),
			},
		},
		{
			LogLevel: slog.LevelError,
			Message:  "Testing Attributes",
			Attrs: []slog.Attr{
				slog.Group("gr",
					slog.String("some", "attribute"),
					slog.Int64("i64k", int64(23)),
					slog.Int("ik", 23),
					slog.Bool("bk", true),
					slog.Float64("fk", 324.2),
					slog.Duration("dk", 12*time.Second),
					slog.Time("tk", time.Unix(1, 1000000)),
					slog.Any("err", fmt.Errorf("err")),
					slog.Any("rk", struct {
						a int
						b string
					}{1, "2"}),
				),
			},
		},
		{
			LogLevel: slog.LevelError,
			Message:  "Testing Attributes",
			Attrs: []slog.Attr{
				slog.String("some", "attribute"),
				slog.Group("gr",
					slog.Duration("dk", 12*time.Second),
					slog.Duration("dk", 12*time.Second),
				),
			},
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("handler with group test %d", i), func(t *testing.T) {
			t.Parallel()
			buffer := bytes.NewBuffer(make([]byte, 0))
			logger := slog.New(rainbow.New(buffer, &opts)).WithGroup("wg")

			logger.LogAttrs(context.Background(), tt.LogLevel.Level(), tt.Message, tt.Attrs...)
			expectedOutput := makeRegex(&opts, tt.LogLevel.Level(), []string{"wg"}, tt.Message, tt.Attrs...)
			output := buffer.String()
			if !expectedOutput.Match(buffer.Bytes()) {
				t.Errorf("output \n%q did not match the expected output regex \n%s", output, expectedOutput.String())
			}
		})
	}
}

func TestRainbow_HandlerWithGroupManual(t *testing.T) {
	tests := []struct {
		LogLevel             slog.Leveler
		Message              string
		Attrs                []slog.Attr
		ManualExpectedRegexp *regexp.Regexp
	}{
		{
			LogLevel: slog.LevelError,
			Message:  "Testing Attributes",
			Attrs: []slog.Attr{
				slog.String("some", "attribute"),
				slog.Int64("i64k", int64(23)),
				slog.Int("ik", 23),
				slog.Bool("bk", true),
				slog.Float64("fk", 324.2),
				slog.Duration("dk", 12*time.Second),
				slog.Group("gr",
					slog.Time("tk", time.Unix(1, 1000000)),
					slog.Any("err", fmt.Errorf("err")),
					slog.Any("rk", struct {
						a int
						b string
					}{1, "2"}),
				),
			},
			ManualExpectedRegexp: regexp.MustCompile(`^<t>[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{3}<ro><le>\|ERR <ro><m>Testing Attributes<ro><so><mas><ro><kd>wg<ro><so>\.<ro><kd>some<ro><so>=<ro><vs>"attribute"<ro><so><aas><ro><kd>wg<ro><so>\.<ro><kd>i64k<ro><so>=<ro><vi>23<ro><so><aas><ro><kd>wg<ro><so>\.<ro><kd>ik<ro><so>=<ro><vi>23<ro><so><aas><ro><kd>wg<ro><so>\.<ro><kd>bk<ro><so>=<ro><vb>true<ro><so><aas><ro><kd>wg<ro><so>\.<ro><kd>fk<ro><so>=<ro><vf>324\.2<ro><so><aas><ro><kd>wg<ro><so>\.<ro><kd>dk<ro><so>=<ro><vd>12s<ro><so><aas><ro><kd>wg<ro><so>\.<ro><tgr>gr<ro><so>\.<ro><kd>tk<ro><so>=<ro><vt>1970-01-01T01:00:01\.001<ro><so><aas><ro><kd>wg<ro><so>\.<ro><tgr>gr<ro><so>\.<ro><ke>err<ro><so>=<ro><ve>err<ro><so><aas><ro><kd>wg<ro><so>\.<ro><tgr>gr<ro><so>\.<ro><kd>rk<ro><so>=<ro><va>\{1 2\}<ro>\n$`),
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("handler with group test %d", i), func(t *testing.T) {
			t.Parallel()
			buffer := bytes.NewBuffer(make([]byte, 0))
			logger := slog.New(rainbow.New(buffer, &opts)).WithGroup("wg")

			logger.LogAttrs(context.Background(), tt.LogLevel.Level(), tt.Message, tt.Attrs...)
			output := buffer.String()
			if !tt.ManualExpectedRegexp.Match(buffer.Bytes()) {
				t.Errorf("output \n%q did not match the expected output regex \n%s", output, tt.ManualExpectedRegexp.String())
			}
		})
	}
}

func TestRainbow_HandlerWithGroupManualNoColor(t *testing.T) {
	opts := opts

	opts.NoColor = true
	tests := []struct {
		LogLevel             slog.Leveler
		Message              string
		Attrs                []slog.Attr
		ManualExpectedRegexp *regexp.Regexp
	}{
		{
			LogLevel: slog.LevelError,
			Message:  "Testing Attributes",
			Attrs: []slog.Attr{
				slog.String("some", "attribute"),
				slog.Int64("i64k", int64(23)),
				slog.Int("ik", 23),
				slog.Bool("bk", true),
				slog.Float64("fk", 324.2),
				slog.Duration("dk", 12*time.Second),
				slog.Group("gr",
					slog.Time("tk", time.Unix(1, 1000000)),
					slog.Any("err", fmt.Errorf("err")),
					slog.Any("rk", struct {
						a int
						b string
					}{1, "2"}),
				),
			},
			ManualExpectedRegexp: regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{3}\|ERR Testing Attributes<mas>wg\.some="attribute"<aas>wg\.i64k=23<aas>wg\.ik=23<aas>wg\.bk=true<aas>wg\.fk=324\.2<aas>wg\.dk=12s<aas>wg\.gr\.tk=1970-01-01T01:00:01\.001<aas>wg\.gr\.err=err<aas>wg\.gr\.rk=\{1 2\}\n$`),
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("handler with group test %d", i), func(t *testing.T) {
			t.Parallel()
			buffer := bytes.NewBuffer(make([]byte, 0))
			logger := slog.New(rainbow.New(buffer, &opts)).WithGroup("wg")

			logger.LogAttrs(context.Background(), tt.LogLevel.Level(), tt.Message, tt.Attrs...)
			output := buffer.String()
			if !tt.ManualExpectedRegexp.Match(buffer.Bytes()) {
				t.Errorf("output \n%q did not match the expected output regex \n%s", output, tt.ManualExpectedRegexp.String())
			}
		})
	}
}

// ERR Testing Attributes<mas>wg.some=\"attribute\"<aas>wg.i64k=23<aas>wg.ik=23<aas>wg.bk=true<aas>wg.fk=324.2<aas>wg.dk=12s<aas>wg.gr.tk=1970-01-01T01:00:01.001<aas>wg.gr.err=err<aas>wg.gr.rk={1 2}\n
// ERR Testing Attributes<mas>wg.some=\"attribute\"<aas>wg.i64k=23<aas>wg.ik=23<aas>wg.bk=true<aas>wg.fk=324.2<aas>wg.dk=12s<aas>wg.gr.tk=1970-01-01T01:00:01.001wg.gr.err=errwg.gr.rk={1 2}\n
