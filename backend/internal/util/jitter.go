package util

import (
	"fmt"
	"math"
	"math/rand/v2"
	"time"
)

// Jitter returns a random whole-millisecond duration in [fromMS, toMS).
// Equal bounds return that fixed duration. Both bounds must be non-negative
// and representable as time.Duration values.
func Jitter(fromMS, toMS int) (time.Duration, error) {
	if fromMS < 0 || toMS < fromMS {
		return 0, fmt.Errorf("invalid jitter bounds: expected 0 <= from <= to")
	}
	if int64(toMS) > math.MaxInt64/int64(time.Millisecond) {
		return 0, fmt.Errorf("invalid jitter bounds: milliseconds exceed time.Duration range")
	}
	milliseconds := fromMS
	if fromMS != toMS {
		milliseconds += rand.IntN(toMS - fromMS)
	}
	return time.Duration(milliseconds) * time.Millisecond, nil
}
