package domain

import (
	"os"
	"strings"
	"time"
)

const defaultAppTimeZone = "Europe/Rome"

// CurrentBusinessDay returns the current YYYY-MM-DD string in the configured app timezone.
func CurrentBusinessDay() string {
	return BusinessDayForTime(time.Now(), os.Getenv("APP_TIMEZONE"))
}

// BusinessDayForTime formats a timestamp using the configured app timezone.
// Invalid or empty timezones fall back to Europe/Rome.
func BusinessDayForTime(now time.Time, tz string) string {
	tz = strings.TrimSpace(tz)
	if tz == "" {
		tz = defaultAppTimeZone
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc, _ = time.LoadLocation(defaultAppTimeZone)
	}
	return now.In(loc).Format("2006-01-02")
}
