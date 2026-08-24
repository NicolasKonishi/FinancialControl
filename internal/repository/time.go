package repository

import (
	"fmt"
	"time"
)

const (
	dateLayout     = "2006-01-02"
	dateTimeLayout = time.RFC3339
)

func formatDate(t time.Time) string {
	return t.UTC().Format(dateLayout)
}

func formatDateTime(t time.Time) string {
	return t.UTC().Format(dateTimeLayout)
}

func parseDate(value string) (time.Time, error) {
	t, err := time.Parse(dateLayout, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse date %q: %w", value, err)
	}
	return t.UTC(), nil
}

func parseDateTime(value string) (time.Time, error) {
	t, err := time.Parse(dateTimeLayout, value)
	if err != nil {
		// SQLite may also return values without timezone in some tools.
		t, err = time.ParseInLocation("2006-01-02 15:04:05", value, time.UTC)
		if err != nil {
			return time.Time{}, fmt.Errorf("parse datetime %q: %w", value, err)
		}
	}
	return t.UTC(), nil
}
