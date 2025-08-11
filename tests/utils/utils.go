package utils

import "time"

func Parse(value string) (time.Time, error) {
	// Reference time format
	layout := "2006-01-02 15:04:05"

	location := time.Now().Location()

	return time.ParseInLocation(layout, value, location)
}
