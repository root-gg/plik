package utils

import "time"

func TruncateDuration(d time.Duration, precision time.Duration) time.Duration {
	if d == 0 {
		return time.Duration(0)
	}
	p := float64(precision)
	n := float64(int(float64(d)/p)) * p
	return time.Duration(n)
}
