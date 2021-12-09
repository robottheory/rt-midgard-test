package miderr

import (
	"github.com/rs/zerolog/log"
	"gitlab.com/thorchain/midgard/config"
)

func Printf(format string, v ...interface{}) {
	log.Info().Msgf(format, v...)
	if config.Global.FailOnError {
		panic("Error in developement mode")
	}
}
