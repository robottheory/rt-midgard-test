package miderr

import "github.com/rs/zerolog/log"

// This is not thread safe.
// It has to be set only by the main before everybody reads it.
var failOnError bool

func SetFailOnError(v bool) {
	if v {
		log.Warn().Msg("Development mode, will fail on any error")
	}
	failOnError = v
}

func Printf(format string, v ...interface{}) {
	log.Info().Msgf(format, v...)
	if failOnError {
		panic("Error in developement mode")
	}
}
