package midlog

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func LogCommandLine() {
	fmt.Printf("Command: %s\n", strings.Join(os.Args, " "))
}

var GlobalLogger Logger
var exitFunction func()

func SetGlobalOutput(w io.Writer) {
	log.Logger = log.Output(
		zerolog.ConsoleWriter{
			Out:        w,
			TimeFormat: "2006-01-02 15:04:05",
			PartsOrder: []string{"level", "time", "caller", "message"},
		},
	)
	GlobalLogger.zlog = log.Logger
	refreshSubloggers()
}

func init() {
	SetGlobalOutput(os.Stdout)
}

var subloggers = map[string]*Logger{}

func newSublogger(module string) Logger {
	return Logger{GlobalLogger.zlog.With().Str("module", module).Logger()}
}

func SubLogger(module string) *Logger {
	l := newSublogger(module)
	subloggers[module] = &l
	return &l
}

func refreshSubloggers() {
	for module, l := range subloggers {
		*l = newSublogger(module)
	}
}

func SetExitFunctionForTest(f func()) {
	exitFunction = f
}

//////////////////// Tags
// Tags are additional fields added to the logs.
// These are usually called Fields, but F in InfoF already refers to format string,
// therefore we refer to them as Tags and call the function InfoT.

type Tag interface {
	apply(logEvent *zerolog.Event)
}

type tagInt64 struct {
	key   string
	value int64
}

func (t tagInt64) apply(logEvent *zerolog.Event) {
	logEvent.Int64(t.key, t.value)
}

func Int64(key string, value int64) Tag {
	return tagInt64{key, value}
}

type tagStr struct {
	key   string
	value string
}

func (t tagStr) apply(logEvent *zerolog.Event) {
	logEvent.Str(t.key, t.value)
}

func Str(key string, value string) Tag {
	return tagStr{key, value}
}

type multiTag struct {
	tags []Tag
}

func (x multiTag) apply(logEvent *zerolog.Event) {
	for _, v := range x.tags {
		v.apply(logEvent)
	}
}

func Tags(tags ...Tag) Tag {
	return multiTag{tags: tags}
}

//////////////// Commands

type Logger struct {
	zlog zerolog.Logger
}

func (l Logger) InfoT(t Tag, msg string) {
	logEvent := l.zlog.Info()
	t.apply(logEvent)
	logEvent.Msg(msg)
}

func (l Logger) Info(msg string) {
	l.zlog.Info().Msg(msg)
}

func (l Logger) InfoF(format string, v ...interface{}) {
	l.zlog.Info().Msgf(format, v...)
}

func (l Logger) Warn(msg string) {
	l.zlog.Warn().Msg(msg)
}

func (l Logger) WarnF(format string, v ...interface{}) {
	l.zlog.Warn().Msgf(format, v...)
}

func (l Logger) WarnT(t Tag, msg string) {
	e := l.zlog.Warn()
	t.apply(e)
	e.Msg(msg)
}

func (l Logger) WarnTF(t Tag, format string, v ...interface{}) {
	e := l.zlog.Warn()
	t.apply(e)
	e.Msgf(format, v...)
}

func (l Logger) Error(msg string) {
	l.zlog.Error().Msg(msg)
}

func (l Logger) ErrorE(err error, msg string) {
	l.zlog.Error().Err(err).Msg(msg)
}

func (l Logger) ErrorF(format string, v ...interface{}) {
	l.zlog.Error().Msgf(format, v...)
}

func (l Logger) Fatal(msg string) {
	l.zlog.Fatal().Msg(msg)
}

func (l Logger) FatalE(err error, msg string) {
	if exitFunction == nil {
		l.zlog.Fatal().Err(err).Msg(msg)
	} else {
		l.zlog.Error().Err(err).Msg(msg)
		exitFunction()
	}
}

func (l Logger) FatalF(format string, v ...interface{}) {
	l.zlog.Fatal().Msgf(format, v...)
}

func (l Logger) FatalEF(err error, format string, v ...interface{}) {
	l.zlog.Fatal().Err(err).Msgf(format, v...)
}

///////////////////// Global utility functions

func InfoT(t Tag, msg string) {
	GlobalLogger.InfoT(t, msg)
}

func Info(msg string) {
	GlobalLogger.Info(msg)
}

func InfoF(format string, v ...interface{}) {
	GlobalLogger.InfoF(format, v...)
}

func Warn(msg string) {
	GlobalLogger.Warn(msg)
}

func WarnF(format string, v ...interface{}) {
	GlobalLogger.WarnF(format, v...)
}

func WarnT(t Tag, msg string) {
	GlobalLogger.WarnT(t, msg)
}

func WarnTF(t Tag, format string, v ...interface{}) {
	GlobalLogger.WarnTF(t, format, v...)
}

func Error(msg string) {
	GlobalLogger.Error(msg)
}

func ErrorE(err error, msg string) {
	GlobalLogger.ErrorE(err, msg)
}

func ErrorF(format string, v ...interface{}) {
	GlobalLogger.ErrorF(format, v...)
}

func Fatal(msg string) {
	GlobalLogger.Fatal(msg)
}

func FatalE(err error, msg string) {
	GlobalLogger.FatalE(err, msg)
}

func FatalF(format string, v ...interface{}) {
	GlobalLogger.FatalF(format, v...)
}

func FatalEF(err error, format string, v ...interface{}) {
	GlobalLogger.FatalEF(err, format, v...)
}
