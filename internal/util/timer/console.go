package timer

import (
	"time"

	"github.com/rs/zerolog/log"
)

type milliCounter time.Time

func MilliCounter() milliCounter {
	return milliCounter(time.Now())
}

func (m milliCounter) SecondsElapsed() float32 {
	return float32(time.Since(time.Time(m)).Milliseconds()) / 1000
}

// Useful for debugging, prints running times to the console.
// When called with defer use: "defer timer.Console()()" (note the trailing "()")
func Console(name string) func() {
	start := MilliCounter()
	log.Debug().Str("name", name).Msg("Timer start")
	return func() {
		log.Debug().Str("name", name).Float32("duration", start.SecondsElapsed()).Msg("Timer end")
	}
}
