package clock

import "time"

// Самые простые часы
type SystemClock struct{}

func (SystemClock) Now() time.Time {
	return time.Now()
}
