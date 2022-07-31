package midlog

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func LogCommandLine() {
	fmt.Printf("Command: %s\n", strings.Join(os.Args, " "))
}

func Init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
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
