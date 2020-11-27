// Reports averages and histograms.
// These metrics will show up at /debug/metrics as timer_* histograms
// There is a separate /debug/timers page for an overview of these metrics only.
package timer

import (
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/pascaldekloe/metrics"
)

type timer struct {
	histogram *metrics.Histogram
}

var allTimers struct {
	sync.RWMutex
	timers []timer
}

const namePrefix = "timer_"

// Timer used for bigger things like requests.
func NewMilli(name string) (ret timer) {
	ret = timer{histogram: metrics.MustHistogram(
		namePrefix+name,
		"Timing histogram for : "+name,
		0.001, 0.01, 0.05, 0.1, 0.2, 0.5, 1, 2, 4)}
	allTimers.Lock()
	allTimers.timers = append(allTimers.timers, ret)
	allTimers.Unlock()
	return ret
}

// Timer used for smaller things like commiting to db.
func NewNano(name string) (ret timer) {
	ret = timer{histogram: metrics.MustHistogram(
		namePrefix+name,
		"Timing histogram for : "+name,
		1e-9, 1e-8, 1e-7, 1e-6, 1e-5, 1e-4, 1e-3, 1e-2, 1e-2, 1)}
	allTimers.Lock()
	allTimers.timers = append(allTimers.timers, ret)
	allTimers.Unlock()
	return ret
}

// Usage, note the final ():
// defer t.One()()
func (t *timer) One() func() {
	t0 := time.Now()
	return func() {
		t.histogram.AddSince(t0)
	}
}

// Usage, note the final ():
// defer t.Batch(10)()
//
// Note: this adds just one value for the full batch. Implications:
// - the count at the summary page has to be multiplied with the average batch
//       size to get the true count.
// - If batch sizes are different than this overrepresent small batches.
func (t *timer) Batch(batchSize int) func() {
	t0 := time.Now()
	return func() {
		duration := float64(time.Now().UnixNano()-t0.UnixNano()) * 1e-9
		t.histogram.Add(duration / float64(batchSize))
	}
}

// Writes timing reports as json
func ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	allTimers.RLock()
	defer allTimers.RUnlock()
	bucketValues := make([]uint64, 0, 20)
	for _, t := range allTimers.timers {
		fmt.Fprintf(resp, "%s\n", t.histogram.Name()[len(namePrefix):])
		vals, count, sum := t.histogram.Get(bucketValues)
		bounds := t.histogram.BucketBounds
		fmt.Fprintf(resp, "    Count: %d\n", count)
		if count != 0 {
			fmt.Fprint(resp, "    Average: ")
			writeFloatTime(resp, sum/float64(count))
			fmt.Fprint(resp, "\n")
			fmt.Fprint(resp, "    Histogram: ")
			cummulative := uint64(0)
			for i := 0; i < len(vals); i++ {
				v := vals[i]
				if v != 0 {
					cummulative += v
					writeIntTime(resp, bounds[i])
					fmt.Fprintf(resp, ": %.1f%%, ", 100*float64(cummulative)/float64(count))
				}
			}
			fmt.Fprint(resp, "\n")
		}
		fmt.Fprint(resp, "\n")
	}
}

func writeIntTime(w io.Writer, durationSec float64) {
	v, unit := normalize(durationSec)
	fmt.Fprintf(w, "%d%s", int(v), unit)
}

func writeFloatTime(w io.Writer, durationSec float64) {
	v, unit := normalize(durationSec)
	// Print only 3 digits out e.g. 1.23 ; 12.3 or 123
	if v < 10 {
		fmt.Fprintf(w, "%.2f%s", v, unit)
	} else if v < 100 {
		fmt.Fprintf(w, "%.1f%s", v, unit)
	} else {
		fmt.Fprintf(w, "%.0f%s", v, unit)
	}
}

func normalize(durationSec float64) (newValue float64, unit string) {
	if 1e-3 <= durationSec {
		newValue = durationSec * 1e3
		unit = "ms"
	} else if 1e-6 <= durationSec {
		newValue = durationSec * 1e6
		unit = "μs"
	} else {
		newValue = durationSec * 1e9
		unit = "Ns"
	}
	return
}
