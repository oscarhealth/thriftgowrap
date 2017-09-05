package retry

import (
	"math"
	"testing"
	"time"
)

func TestDecorrelatedExponentialBackoff(t *testing.T) {
	backoff := NewDecorrelatedExponentialBackoff(100, 30000).New().(*decorrelatedExponentialBackoff)

	// Ensure the first duration is equal to the min
	if int64(backoff.getDuration()) != 100 {
		t.Error("expected first duration to be min")
	}

	// Ensure sequential durations fall within the expected bounds
	lastDuration := backoff.minWait
	for idx := 0; idx < 200; idx++ {
		duration := backoff.getDuration()

		if duration > lastDuration*3 || duration < backoff.minWait || duration > backoff.maxWait {
			t.Error("unexpected duration")
		}
		lastDuration = duration
	}
}

func TestExponentialBackoff(t *testing.T) {
	// Test max value limit
	backoff0 := NewExponentialBackoff(100, 30000).New().(*ExponentialBackoff)
	expectedVals0 := []time.Duration{100, 200, 400, 800, 1600, 3200, 6400, 12800, 25600, 30000, 30000}
	for i, expected := range expectedVals0 {
		res := backoff0.getDuration(uint64(i))
		if res < expected/2 || res > expected {
			t.Errorf("Expected %d, received %d", expected, res)
		}
	}

	backoff1 := &ExponentialBackoff{minWait: 100, maxWait: 30000, jitter: false}
	expectedVals1 := []time.Duration{100, 200, 400, 800, 1600, 3200, 6400, 12800, 25600, 30000, 30000}
	for i, expected := range expectedVals1 {
		res := backoff1.getDuration(uint64(i))
		if res != expected {
			t.Errorf("Expected %d, received %d", expected, res)
		}
	}

	// Test overflow defaults to max due to multiplication
	backoff2 := &ExponentialBackoff{minWait: 100, maxWait: math.MaxInt64, jitter: false}
	res2 := backoff2.getDuration(62)
	var exp2 time.Duration = math.MaxInt64
	if res2 != exp2 {
		t.Errorf("Expected %d, received %d", exp2, res2)
	}

	// Test no overflow
	backoff3 := &ExponentialBackoff{minWait: 1, maxWait: math.MaxInt64, jitter: false}
	res3 := backoff3.getDuration(62)
	var exp3 time.Duration = 1
	exp3 = exp3 << 62
	if res3 != exp3 {
		t.Errorf("Expected %d, received %d", exp3, res3)
	}

	// Test overflow defaults to max due to exponentiation
	backoff4 := &ExponentialBackoff{minWait: 100, maxWait: math.MaxInt64, jitter: false}
	res4 := backoff4.getDuration(63)
	var exp4 time.Duration = math.MaxInt64
	if res4 != exp4 {
		t.Errorf("Expected %d, received %d", exp4, res4)
	}
}

func TestFixedBackoff(t *testing.T) {
	backoff := NewFixedBackoff(1000 * time.Millisecond).New()
	start := time.Now()
	backoff.Backoff(500) // 500 should not affect anything
	dur := time.Since(start).Seconds()

	if dur < 1. || dur > 1.05 {
		t.Error("FixedBackoff incorrect sleep")
	}
}
