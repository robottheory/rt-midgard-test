package timer

import (
	"time"

	"github.com/rs/zerolog/log"
)

// Useful for debugging, prints running times to the console.
// When called with defer use: "defer timer.Console()()" (note the trailing "()")
func Console(name string) func() {
	start := time.Now()
	log.Info().Str("name", name).Msg("Timer start")
	return func() {
		var m float32 = float32(time.Since(start).Milliseconds()) / 1000
		log.Info().Str("name", name).Float32("delta", m).Msg("Timer end")
	}
}
