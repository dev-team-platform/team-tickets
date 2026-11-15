package util

import (
	"math"
	"strconv"
	"testing"
	"time"
)

func TestJitterFixedBounds(t *testing.T) {
	for _, value := range []int{0, 5, 1000} {
		got, err := Jitter(value, value)
		if err != nil || got != time.Duration(value)*time.Millisecond {
			t.Fatalf("Jitter(%d, %d) = %v, %v", value, value, got, err)
		}
	}
}

func TestJitterReturnsWholeMillisecondsWithinBounds(t *testing.T) {
	for range 1000 {
		got, err := Jitter(1000, 3000)
		if err != nil || got < time.Second || got >= 3*time.Second || got%time.Millisecond != 0 {
			t.Fatalf("unexpected jitter: %v, %v", got, err)
		}
	}
}

func TestJitterRejectsInvalidBounds(t *testing.T) {
	for _, bounds := range [][2]int{
		{-1, 1},
		{2, 1},
		{0, -1},
	} {
		if _, err := Jitter(bounds[0], bounds[1]); err == nil {
			t.Errorf("expected error for %v", bounds)
		}
	}
	if strconv.IntSize == 64 {
		overflow := int64(math.MaxInt64/int64(time.Millisecond)) + 1
		if _, err := Jitter(0, int(overflow)); err == nil {
			t.Error("expected error for unrepresentable duration")
		}
	}
}
