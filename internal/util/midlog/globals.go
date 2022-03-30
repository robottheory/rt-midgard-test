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
