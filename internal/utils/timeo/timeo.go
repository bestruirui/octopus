package timeo

import "time"

func ToMidnight(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}
func GetMidnight() time.Time {
	return ToMidnight(time.Now())
}
