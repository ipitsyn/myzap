package myzap

import (
	"fmt"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

// MyZap : structure wrapping up zaplogger to have it customized
type MyZap struct {
	Logger *zap.Logger
	Sugar  *zap.SugaredLogger
	Atom   *zap.AtomicLevel
}

// Foreground colors.
const (
	Black uint8 = iota + 30
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White
)

// Color represents a text color.
type fColor struct {
	color    uint8
	isBright bool
}

// adds the coloring to the given string.
func (c fColor) add(s string) string {
	if c.isBright {
		return fmt.Sprintf("\x1b[%d;1m%s\x1b[0m", uint8(c.color), s)
	}
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", uint8(c.color), s)
}

// Variables to hold level-to-color mapping
var (
	_levelToColor = map[zapcore.Level]fColor{
		zapcore.DebugLevel:  {Magenta, false},
		zapcore.InfoLevel:   {Cyan, false},
		zapcore.WarnLevel:   {Yellow, false},
		zapcore.ErrorLevel:  {Red, false},
		zapcore.DPanicLevel: {Red, true},
		zapcore.PanicLevel:  {Red, true},
		zapcore.FatalLevel:  {Red, true},
	}
	_unknownLevelColor         = fColor{Red, true}
	_levelToCapitalColorString = make(map[zapcore.Level]string, len(_levelToColor))
)

func mySugarLevelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	// See source at https://github.com/uber-go/zap/blob/v1.16.0/zapcore/encoder.go#L65
	s, ok := _levelToCapitalColorString[l]
	if !ok {
		s = _unknownLevelColor.add(l.CapitalString())
	}
	enc.AppendString(s)
}

func mySugarCallerEncoder(ec zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	// See source at https://github.com/uber-go/zap/blob/v1.16.0/zapcore/entry.go#L88
	if !ec.Defined {
		enc.AppendString("undefined")
		return
	}
	// Find the last separator.
	idx := strings.LastIndexByte(ec.File, '/')
	if idx == -1 {
		enc.AppendString(ec.FullPath())
		return
	}
	buf := buffer.NewPool().Get()
	// Keep everything after the last separator.
	buf.AppendByte('[')
	buf.AppendString(ec.File[idx+1:])
	buf.AppendByte(':')
	buf.AppendInt(int64(ec.Line))
	buf.AppendByte(']')
	caller := buf.String()
	buf.Free()

	enc.AppendString(caller)
}

// New : Create and initialize MyZAP logger using custom format and passed logging level
func New(lvl zapcore.Level) *MyZap {
	atom := zap.NewAtomicLevel()

	// Use Production Config as template
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = ""

	// Set up custom levels coloring
	for level, color := range _levelToColor {
		_levelToCapitalColorString[level] = color.add(level.CapitalString())
	}
	encoderCfg.EncodeLevel = mySugarLevelEncoder
	encoderCfg.EncodeCaller = mySugarCallerEncoder

	// Create logger
	logger := zap.New(
		zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderCfg),
			zapcore.Lock(os.Stdout),
			atom,
		),
		zap.AddCaller(),
	)
	sugar := logger.Sugar()

	// Adjust logging level
	atom.SetLevel(lvl)

	sugar.Debug("Initialized logging")
	sugar.Debugf("Set logging level to %s", strings.ToUpper(atom.String()))

	return &MyZap{
		Logger: logger,
		Sugar:  sugar,
		Atom:   &atom,
	}
}

// NewFileLogger : logs to file
func NewFileLogger(lvl zapcore.Level, fileName string) (*zap.Logger, error) {
	cfg := zap.Config{
		Level:             zap.NewAtomicLevelAt(lvl),
		Development:       false,
		DisableCaller:     true,
		DisableStacktrace: true,
		Sampling:          nil,
		Encoding:          "console",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:    "T",
			MessageKey: "M",
			EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
				enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
			},
		},
		OutputPaths: []string{
			fileName,
		},
	}
	return cfg.Build()
}
