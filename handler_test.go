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

func TestRainbow_Handler(t *testing.T) {
	opts := rainbow.Options{
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
				"testgroup": "<gt>",
			},
		},
		SpecialOverrides: &rainbow.SpecialColorOverrides{
			Time:    "<t>",
			Message: "<m>",
		},
	}
	dateRE := string(opts.SpecialOverrides.Time + "[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\\.[0-9]{3}" + opts.ResetOverride)
	logLevelDebugRE := string(opts.LevelOverrides.Debug + "\\|DBG " + opts.ResetOverride)
	logLevelInfoRE := string(opts.LevelOverrides.Info + "\\|INF " + opts.ResetOverride)
	logLevelWarnRE := string(opts.LevelOverrides.Warning + "\\|WRN " + opts.ResetOverride)
	logLevelErrRE := string(opts.LevelOverrides.Error + "\\|ERR " + opts.ResetOverride)
	messageRE := func(msg string) string {
		return string(opts.SpecialOverrides.Message) + msg + string(opts.ResetOverride)
	}
	keyRE := func(key string) string {
		col, ok := opts.KeyOverrides.KeyMap[key]
		if !ok {
			col = opts.KeyOverrides.Default
		}
		return string(col) + key + string(opts.ResetOverride) + string(opts.SymbolOverride) + "=" + string(opts.ResetOverride)
	}
	attrRE := func(attr *slog.Attr) string {
		attr.Value = attr.Value.Resolve()
		rstring := keyRE(attr.Key)
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
			return rstring + string(opts.ValueOverrides.Any) + fmt.Sprintf("%v", attr.Value.Any()) + string(opts.ResetOverride)
		default:
			return ""
		}
	}
	makeRegex := func(level slog.Leveler, message string, attrs ...slog.Attr) *regexp.Regexp {
		levelStr := ""
		switch level.Level() {
		case slog.LevelDebug:
			levelStr = logLevelDebugRE
		case slog.LevelInfo:
			levelStr = logLevelInfoRE
		case slog.LevelWarn:
			levelStr = logLevelWarnRE
		case slog.LevelError:
			levelStr = logLevelErrRE
		}
		reStr := "^" + dateRE + levelStr + messageRE(message)

		numAttrs := len(attrs)
		if numAttrs > 0 {
			reStr = reStr + string(opts.SymbolOverride) + opts.MessageAttrSeparator + string(opts.ResetOverride)
		}
		for i, extra := range attrs {
			reStr = reStr + attrRE(&extra)
			if i < numAttrs-1 {
				reStr = reStr + string(opts.SymbolOverride) + opts.AttrAttrSeparator + string(opts.ResetOverride)
			}
		}

		return regexp.MustCompile(reStr + `\n$`)
	}

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
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("handler test %d", i), func(t *testing.T) {
			t.Parallel()
			buffer := bytes.NewBuffer(make([]byte, 0))
			logger := slog.New(rainbow.New(buffer, &opts))

			logger.LogAttrs(context.Background(), tt.LogLevel.Level(), tt.Message, tt.Attrs...)
			expectedOutput := makeRegex(tt.LogLevel.Level(), tt.Message, tt.Attrs...)
			output := buffer.String()
			if !expectedOutput.Match(buffer.Bytes()) {
				t.Errorf("output \n%q did not match the expected output regex \n%s", output, expectedOutput.String())
			}
		})
	}
}
