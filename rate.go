// Experiments with doing X at a given rate and duration or count
// and comparaison between uber's ratelimit and a simple one
package main

import (
	"flag"
	"math"
	"time"

	"fortio.org/cli"
	"fortio.org/log"
	"go.uber.org/ratelimit"
)

type simpleLimiter struct {
	rate  float64
	start time.Time
	count int64
}

const warnThreshold = -10 * time.Millisecond

func (l *simpleLimiter) Take() time.Time {
	now := time.Now()
	elapsed := now.Sub(l.start)
	l.count++
	targetElapsedInSec := float64(l.count) / l.rate
	targetElapsedDuration := time.Duration(int64(targetElapsedInSec * 1e9))

	sleepTime := targetElapsedDuration - elapsed
	log.LogVf("Elapsed %v, Expected for %d: %v, Sleeping for %v", elapsed, l.count, targetElapsedDuration, sleepTime)
	if sleepTime < warnThreshold {
		log.Warnf("Falling behind by %v at %d, rate %v too high?", -sleepTime, l.count, l.rate)
		return now
	}
	time.Sleep(sleepTime)
	return now
}

func DurationBased(rl ratelimit.Limiter, duration time.Duration) {
	log.Printf("Duration based: will run for %v", duration)
	start := time.Now()
	end := start.Add(duration)
	i := 0
	for time.Until(end) > 0 {
		rl.Take()
		i++
		log.LogVf("iter %v", i)
	}
	elapsed := time.Since(start)
	actualRate := float64(i) / elapsed.Seconds()
	log.Printf("Done after %v - did %d iterations actual rate %.3f", elapsed, i, actualRate)
}

func IterBased(rl ratelimit.Limiter, iterations int) {
	log.Printf("Iterations based: will run for %v", iterations)
	start := time.Now()
	i := 0
	for i < iterations {
		rl.Take()
		i++
		log.LogVf("iter %v", i)
	}
	elapsed := time.Since(start)
	actualRate := float64(i) / elapsed.Seconds()
	log.Printf("Done after %v - did %d iterations actual rate %.3f", elapsed, i, actualRate)
}

func main() {
	rateFlag := flag.Float64("rate", 1000.0, "desired rate per second (needs to be int for uber)")
	durationFlag := flag.Duration("duration", 500*time.Millisecond, "desired duration")
	exactlyFlag := flag.Int("exactly", 0, "when set, superseeds duration")
	useUber := flag.Bool("uber", false, "use uber limiter instead of simple built in one")
	cli.Main()
	rate := *rateFlag
	duration := *durationFlag
	exactly := *exactlyFlag
	var rl ratelimit.Limiter
	if *useUber {
		r := int(math.Round(rate))
		log.Printf("Using UBER with rate of %d / sec", r)
		rl = ratelimit.New(r) // per second
	} else {
		log.Printf("Using Simple rate %.3f / sec", rate)
		rl = &simpleLimiter{
			rate:  rate,
			start: time.Now(),
		}
	}
	if exactly > 0 {
		IterBased(rl, exactly)
	} else {
		DurationBased(rl, duration)
	}
}
