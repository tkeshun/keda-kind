package timeutil

import "time"

var jstLocation = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err == nil {
		return loc
	}
	return time.FixedZone("Asia/Tokyo", 9*60*60)
}()

func JSTLocation() *time.Location {
	return jstLocation
}

func ToJST(t time.Time) time.Time {
	return t.In(jstLocation)
}
