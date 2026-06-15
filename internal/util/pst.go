package util

import "time"

var pstLoc *time.Location

func init() {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		loc = time.FixedZone("PST", -8*3600)
	}
	pstLoc = loc
}

// SecondsUntilNextPSTMidnight returns seconds until the next 00:00 US/Pacific.
func SecondsUntilNextPSTMidnight() float64 {
	now := time.Now().In(pstLoc)
	midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, pstLoc)
	next := midnight.Add(24 * time.Hour)
	return next.Sub(now).Seconds()
}

// ShouldRunSinceLastPSTDay reports whether the last run was on a different PST calendar day.
func ShouldRunSinceLastPSTDay(lastUnix float64) bool {
	if lastUnix <= 0 {
		return true
	}
	now := time.Now().In(pstLoc)
	last := time.Unix(int64(lastUnix), 0).In(pstLoc)
	ny, nm, nd := now.Date()
	ly, lm, ld := last.Date()
	return ny != ly || nm != lm || nd != ld
}

// NowPSTUnix returns the current time as a Unix timestamp (PST wall-clock aware).
func NowPSTUnix() float64 {
	return float64(time.Now().In(pstLoc).Unix())
}
