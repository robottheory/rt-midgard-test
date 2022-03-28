package midlog

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func LogStartCommand() {
	fmt.Printf("Command: %s\n", strings.Join(os.Args, " "))
}

func Init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
}

func Info(msg string) {
	log.Info().Msg(msg)
}

func InfoF(format string, v ...interface{}) {
	log.Info().Msgf(format, v)
}

func Warn(msg string) {
	log.Warn().Msg(msg)
}

func Error(msg string) {
	log.Error().Msg(msg)
}

func ErrorE(err error, msg string) {
	log.Error().Err(err).Msg(msg)
}

func ErrorF(format string, v ...interface{}) {
	log.Error().Msgf(format, v)
}

func Fatal(msg string) {
	log.Fatal().Msg(msg)
}

func FatalE(err error, msg string) {
	log.Fatal().Err(err).Msg(msg)
}

func FatalF(format string, v ...interface{}) {
	log.Fatal().Msgf(format, v)
}
