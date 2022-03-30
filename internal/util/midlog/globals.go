package midlog

import (
	"io"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var GlobalLogger Logger
var exitFunction func()
var subloggers = map[string]*Logger{}

func init() {
	SetGlobalOutput(os.Stdout)
}

func SetExitFunctionForTest(f func()) {
	exitFunction = f
}

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

func SubLogger(module string) *Logger {
	l := newSublogger(module)
	subloggers[module] = &l
	return &l
}

func newSublogger(module string) Logger {
	return Logger{GlobalLogger.zlog.With().Str("module", module).Logger()}
}

func refreshSubloggers() {
	for module, l := range subloggers {
		*l = newSublogger(module)
	}
}
