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

func InfoT(t Tag, msg string) {
	logEvent := log.Info()
	t.apply(logEvent)
	logEvent.Msg(msg)
}

func Info(msg string) {
	log.Info().Msg(msg)
}

func InfoF(format string, v ...interface{}) {
	log.Info().Msgf(format, v...)
}

func Warn(msg string) {
	log.Warn().Msg(msg)
}

func WarnF(format string, v ...interface{}) {
	log.Warn().Msgf(format, v...)
}

func WarnT(t Tag, msg string) {
	e := log.Warn()
	t.apply(e)
	e.Msg(msg)
}

func WarnTF(t Tag, format string, v ...interface{}) {
	e := log.Warn()
	t.apply(e)
	e.Msgf(format, v...)
}

func Error(msg string) {
	log.Error().Msg(msg)
}

func ErrorE(err error, msg string) {
	log.Error().Err(err).Msg(msg)
}

func ErrorF(format string, v ...interface{}) {
	log.Error().Msgf(format, v...)
}

func Fatal(msg string) {
	log.Fatal().Msg(msg)
}

func FatalE(err error, msg string) {
	log.Fatal().Err(err).Msg(msg)
}

func FatalF(format string, v ...interface{}) {
	log.Fatal().Msgf(format, v...)
}
