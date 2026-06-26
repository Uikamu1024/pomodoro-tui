package config

import (
	"flag"
	"time"
)

type Config struct {
	WorkDuration      time.Duration
	BreakDuration     time.Duration
	LongBreakDuration time.Duration
	CyclesBeforeLong  int
}

func Parse() Config {
	work := flag.Duration("work", 25*time.Minute, "work session duration")
	short := flag.Duration("break", 5*time.Minute, "short break duration")
	long := flag.Duration("long-break", 15*time.Minute, "long break duration")
	cycles := flag.Int("cycles", 4, "number of work sessions before a long break")
	flag.Parse()

	return Config{
		WorkDuration:      *work,
		BreakDuration:     *short,
		LongBreakDuration: *long,
		CyclesBeforeLong:  *cycles,
	}
}
