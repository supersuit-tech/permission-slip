package amadeus

import "time"

// validDate checks that s is a valid YYYY-MM-DD date string.
func validDate(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}
