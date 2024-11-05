package rainbow

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
)

type TextHandler struct {
	// TODO: state for WithGroup and WithAttrs
	lock *sync.Mutex
	out  io.Writer

	level slog.Leveler

	levelColors   *LevelColorOverrides
	valueColors   *ValueColorOverrides
	keyColors     *KeyColorOverrides
	specialColors *SpecialColorOverrides

	resetMod  AnsiMod
	symbolMod AnsiMod

	baseState handleState

	messageAttrSeparator string
	attrAttrSeparator    string
}

type LevelColorOverrides struct {
	Error   AnsiMod
	Warning AnsiMod
	Debug   AnsiMod
	Info    AnsiMod
}

type SpecialColorOverrides struct {
	Time    AnsiMod
	Message AnsiMod
}

type ValueColorOverrides struct {
	String   AnsiMod
	Int      AnsiMod
	Float    AnsiMod
	Uint     AnsiMod
	Error    AnsiMod
	Time     AnsiMod
	Bool     AnsiMod
	Duration AnsiMod
	Any      AnsiMod
}

type KeyColorOverrides struct {
	Default  AnsiMod
	KeyMap   map[string]AnsiMod
	GroupMap map[string]AnsiMod
}

type Options struct {
	// Level reports the minimum level to log.
	// Levels with lower levels are discarded.
	// If nil, the Handler uses [slog.LevelInfo].
	Level slog.Leveler
	// Use this to disable/enable color. Defaults
	// to false, meaning color on.
	// Can be used for example with isatty.IsTermanal
	// ignored if NO_COLOR env var is set to ANYTHING
	NoColor bool

	MessageAttrSeparator string
	AttrAttrSeparator    string

	LevelOverrides   *LevelColorOverrides
	ValueOverrides   *ValueColorOverrides
	KeyOverrides     *KeyColorOverrides
	SpecialOverrides *SpecialColorOverrides

	SymbolOverride AnsiMod
	ResetOverride  AnsiMod
}

func (h *TextHandler) clone() *TextHandler {
	h2 := &TextHandler{
		lock: h.lock,
		out:  h.out,

		level: h.level,

		levelColors:   h.levelColors,
		valueColors:   h.valueColors,
		keyColors:     h.keyColors,
		specialColors: h.specialColors,

		resetMod:  h.resetMod,
		symbolMod: h.symbolMod,

		baseState: *h.baseState.clone(),

		attrAttrSeparator:    h.attrAttrSeparator,
		messageAttrSeparator: h.messageAttrSeparator,
	}
	return h2
}

func New(out io.Writer, opts *Options) slog.Handler {
	if opts == nil {
		opts = &Options{
			Level:   slog.LevelInfo,
			NoColor: false,
		}
	}

	// could be null
	if opts.Level == nil {
		opts.Level = slog.LevelInfo
	}
	levelColors := getOrDefaultLevelColorOverrides(opts.LevelOverrides)
	valueColors := getOrDefaultValueColorOverrides(opts.ValueOverrides)
	keyColors := getOrDefaultKeyColorOverrides(opts.KeyOverrides)
	specialColors := getOrDefaultSpecialOverrides(opts.SpecialOverrides)
	resetMod := Mod(Fmt.Reset)
	if opts.ResetOverride != "" {
		resetMod = opts.ResetOverride
	}
	symbolMod := Mod(Fmt.Faint, Fg.HiWhite)
	if opts.SymbolOverride != "" {
		symbolMod = opts.SymbolOverride
	}
	withColor := !opts.NoColor && os.Getenv("NO_COLOR") == ""

	if !withColor {
		levelColors = &LevelColorOverrides{}
		valueColors = &ValueColorOverrides{}
		specialColors = &SpecialColorOverrides{}
		keyColors = &KeyColorOverrides{
			GroupMap: make(map[string]AnsiMod),
			KeyMap:   make(map[string]AnsiMod),
		}
		resetMod = ""
		symbolMod = ""
	}

	messageAttrSeparator := "\n\t"
	if opts.MessageAttrSeparator != "" {
		messageAttrSeparator = opts.MessageAttrSeparator
	}

	attrAttrSeparator := "\n\t"
	if opts.AttrAttrSeparator != "" {
		attrAttrSeparator = opts.AttrAttrSeparator
	}

	h := &TextHandler{
		out:           out,
		lock:          &sync.Mutex{},
		level:         opts.Level,
		levelColors:   levelColors,
		valueColors:   valueColors,
		keyColors:     keyColors,
		specialColors: specialColors,
		resetMod:      resetMod,
		symbolMod:     symbolMod,

		messageAttrSeparator: messageAttrSeparator,
		attrAttrSeparator:    attrAttrSeparator,
	}

	return h
}

func getOrDefaultValueColorOverrides(valueColorOverrides *ValueColorOverrides) *ValueColorOverrides {
	if valueColorOverrides != nil {
		return valueColorOverrides
	}
	return &ValueColorOverrides{
		String:   Mod(),
		Int:      Mod(Fg.Yellow),
		Float:    Mod(Fg.Yellow),
		Uint:     Mod(Fg.Yellow),
		Error:    Mod(Fg.Red),
		Bool:     Mod(Fg.Green),
		Time:     Mod(Fmt.Italic),
		Duration: Mod(Fg.Cyan),
		Any:      Mod(),
	}
}

func getOrDefaultLevelColorOverrides(levelColorOverrides *LevelColorOverrides) *LevelColorOverrides {
	if levelColorOverrides != nil {
		return levelColorOverrides
	}
	return &LevelColorOverrides{
		Debug:   Mod(Fg.Green),
		Info:    Mod(Fg.Blue),
		Warning: Mod(Fg.Yellow),
		Error:   Mod(Fg.Red),
	}
}

func getOrDefaultSpecialOverrides(specialColorOverrides *SpecialColorOverrides) *SpecialColorOverrides {
	if specialColorOverrides != nil {
		return specialColorOverrides
	}
	return &SpecialColorOverrides{
		Time:    Mod(Fmt.Faint, Fg.HiBlack),
		Message: Mod(),
	}
}

func getOrDefaultKeyColorOverrides(keyColorOverrides *KeyColorOverrides) *KeyColorOverrides {
	if keyColorOverrides != nil {
		return keyColorOverrides
	}
	return &KeyColorOverrides{
		Default: Mod(Fmt.Faint, Fg.HiWhite, Fmt.Italic, Fmt.Faint),
		KeyMap: map[string]AnsiMod{
			"error": Mod(Fg.Red, Fmt.Faint),
			"err":   Mod(Fg.Red, Fmt.Faint),
		},
		GroupMap: map[string]AnsiMod{},
	}
}

func (h *TextHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

type handleState struct {
	CurrentGroupName       string
	PreformattedAttributes string
}

func (hs *handleState) clone() *handleState {
	hsc := &handleState{}
	hsc.CurrentGroupName = strings.Clone(hs.CurrentGroupName)
	hsc.PreformattedAttributes = strings.Clone(hs.PreformattedAttributes)
	return hsc
}

func (h *TextHandler) Handle(ctx context.Context, r slog.Record) error {
	// get buffer, dereference, and then reassign before freeing
	// see https://github.com/golang/example/blob/master/slog-handler-guide/README.md#speed
	bufp := allocBuf()
	buf := *bufp
	defer func() {
		*bufp = buf
		freeBuf(bufp)
	}()

	hs := h.baseState.clone()

	if !r.Time.IsZero() {
		buf = h.appendRecordTime(buf, r.Time.Round(0))
	}
	buf = h.appendRecordLevel(buf, r.Level, hs)
	buf = fmt.Appendf(buf, "%s%s%s", h.specialColors.Message, r.Message, h.resetMod)
	if hs.PreformattedAttributes != "" || r.NumAttrs() > 0 {
		buf = fmt.Appendf(buf, "%s%s%s", h.symbolMod, h.messageAttrSeparator, h.resetMod)
	}
	if hs.PreformattedAttributes != "" {
		buf = append(buf, hs.PreformattedAttributes...)
		if r.NumAttrs() > 0 {
			buf = fmt.Appendf(buf, "%s%s%s", h.symbolMod, h.attrAttrSeparator, h.resetMod)
		}
	}

	numAttrs := r.NumAttrs()
	curAttrs := 0
	r.Attrs(func(a slog.Attr) bool {
		curAttrs++
		a.Value = a.Value.Resolve()
		buf = h.appendAttr(buf, a, hs)
		if curAttrs < numAttrs {
			buf = fmt.Appendf(buf, "%s%s%s", h.symbolMod, h.attrAttrSeparator, h.resetMod)
		}
		return true
	})
	buf = append(buf, '\n')

	h.lock.Lock()
	defer h.lock.Unlock()
	_, err := h.out.Write(buf)
	return err
}

func (h *TextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	h2 := h.clone()

	bufp := allocBuf()
	buf := *bufp
	defer func() {
		*bufp = buf
		freeBuf(bufp)
	}()

	separator := ""
	if h2.baseState.PreformattedAttributes != "" {
		separator = fmt.Sprintf("%s%s%s", h.symbolMod, h.attrAttrSeparator, h.resetMod)
	}
	// write attributes to buffer
	length := len(attrs)
	for i, attr := range attrs {
		attr.Value = attr.Value.Resolve()
		buf = h2.appendAttr(buf, attr, &h2.baseState)
		if i < length-1 {
			buf = fmt.Appendf(buf, "%s%s%s", h.symbolMod, h.attrAttrSeparator, h.resetMod)
		}
	}
	h2.baseState.PreformattedAttributes = h2.baseState.PreformattedAttributes + separator + string(buf)
	return h2
}

func (h *TextHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	h2 := h.clone()
	h2.baseState.CurrentGroupName = h.appendCurrentGroupName(h2.baseState.CurrentGroupName, name)
	return h2
}

func (h *TextHandler) appendRecordTime(buf []byte, time time.Time) []byte {
	formattedTime := time.Format("2006-01-02T15:04:05.000")
	buf = fmt.Appendf(buf, "%s%s%s", h.specialColors.Time, formattedTime, h.resetMod)
	return buf
}

func (h *TextHandler) appendRecordLevel(buf []byte, level slog.Level, _ *handleState) []byte {
	switch level {
	case slog.LevelDebug:
		return fmt.Appendf(buf, "%s|DBG %s", h.levelColors.Debug, h.resetMod)
	case slog.LevelInfo:
		return fmt.Appendf(buf, "%s|INF %s", h.levelColors.Info, h.resetMod)
	case slog.LevelWarn:
		return fmt.Appendf(buf, "%s|WRN %s", h.levelColors.Warning, h.resetMod)
	case slog.LevelError:
		return fmt.Appendf(buf, "%s|ERR %s", h.levelColors.Error, h.resetMod)
	default:
		return fmt.Appendf(buf, "|INVALID ")
	}
}

func (h *TextHandler) appendAttr(buf []byte, a slog.Attr, hs *handleState) []byte {
	// Ignore empty Attrs.
	if a.Equal(slog.Attr{}) {
		return buf
	}
	kind := a.Value.Kind()

	if kind != slog.KindGroup {
		keyCol, ok := h.keyColors.KeyMap[a.Key]
		if !ok {
			keyCol = h.keyColors.Default
		}
		buf = fmt.Appendf(buf, "%s%s%s%s%s=%s", hs.CurrentGroupName, keyCol, a.Key, h.resetMod, h.symbolMod, h.resetMod)
	}
	switch a.Value.Kind() {
	case slog.KindInt64:
		buf = fmt.Appendf(buf, "%s%d%s", h.valueColors.Int, a.Value.Int64(), h.resetMod)
	case slog.KindFloat64:
		buf = fmt.Appendf(buf, "%s%g%s", h.valueColors.Float, a.Value.Float64(), h.resetMod)
	case slog.KindUint64:
		buf = fmt.Appendf(buf, "%s%d%s", h.valueColors.Uint, a.Value.Uint64(), h.resetMod)
	case slog.KindString:
		buf = fmt.Appendf(buf, "%s%q%s", h.valueColors.String, a.Value.String(), h.resetMod)
	case slog.KindBool:
		buf = fmt.Appendf(buf, "%s%t%s", h.valueColors.Bool, a.Value.Bool(), h.resetMod)
	case slog.KindTime:
		// Write times in a standard way
		formattedTime := a.Value.Time().Format("2006-01-02T15:04:05.000")
		buf = fmt.Appendf(buf, "%s%s%s", h.valueColors.Time, formattedTime, h.resetMod)
	case slog.KindDuration:
		formattedDuration := a.Value.Duration().String()
		buf = fmt.Appendf(buf, "%s%s%s", h.valueColors.Duration, formattedDuration, h.resetMod)
	case slog.KindGroup:
		attrs := a.Value.Group()
		// Ignore empty groups.
		if len(attrs) == 0 {
			return buf
		}
		hss := hs.clone()
		hss.CurrentGroupName = h.appendCurrentGroupName(hss.CurrentGroupName, a.Key)
		for _, ga := range attrs {
			buf = h.appendAttr(buf, ga, hss)
		}
	case slog.KindAny:
		errVal, ok := a.Value.Any().(error)
		if ok {
			buf = fmt.Appendf(buf, "%s%s%s", h.valueColors.Error, errVal.Error(), h.resetMod)
		} else {
			buf = fmt.Appendf(buf, "%s%v%s", h.valueColors.Any, a.Value.Any(), h.resetMod)
		}
	default:
		buf = fmt.Appendf(buf, "%v", a.Value)
	}
	return buf
}

func (h *TextHandler) appendCurrentGroupName(currentGroupName, newGroupName string) string {
	col, ok := h.keyColors.GroupMap[newGroupName]
	if !ok {
		col = h.keyColors.Default
	}
	return fmt.Sprintf("%s%s%s%s%s.%s", currentGroupName, col, newGroupName, h.resetMod, h.symbolMod, h.resetMod)
}

// see https://github.com/golang/example/blob/master/slog-handler-guide/README.md#speed
// have a pool of memory to log into
var bufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, 1024)
		return &b
	},
}

// grab a buffer from the pool and type assert it
func allocBuf() *[]byte {
	return bufPool.Get().(*[]byte)
}

func freeBuf(b *[]byte) {
	// To reduce peak allocation, return only smaller buffers to the pool.
	// otherwise the big buffers might be kept in the pool for way too long.
	// big buffers should be deallocated
	const maxBufferSize = 16 << 10
	if cap(*b) <= maxBufferSize {
		*b = (*b)[:0]
		bufPool.Put(b)
	}
}
