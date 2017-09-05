package retry

import (
	"math"
	"math/rand"
	"time"
)

const (
	numUint64Bits = 64
	zeroDuration  = time.Duration(0)
)

var (
	// DefaultBackoff defines the backoff that is used if none is provided. An
	// exponential backoff is used that starts at 0.5s and maxes out at 60s.
	DefaultBackoff = NewDecorrelatedExponentialBackoff(
		time.Millisecond*500,
		time.Second*60,
	)
)

// BackoffFactory is a factory that returns a new Backoff instance. A Retrier uses this
// factory to instantiate a Backoff for each Run. Stateless backoffs can just return
// themselves, and also implement the Backoff interface.
type BackoffFactory interface {
	New() Backoff
}

// Backoff is an interface that is called after Retrier encounters an error. Backoff is
// provided iteration which is the iteration that last failed starting at 0.
type Backoff interface {
	Backoff(iteration uint64)
}

// FunctionalBackoff is a Backoff that wraps a function.
type FunctionalBackoff func(uint64)

// New returns itself.
func (b FunctionalBackoff) New() Backoff {
	return b
}

// Backoff calls the function provided to FunctionalBackoff.
func (b FunctionalBackoff) Backoff(iteration uint64) {
	b(iteration)
}

// NoopBackoff is a Backoff that performs no operation (does not wait).
var NoopBackoff = FunctionalBackoff(func(iteration uint64) {})

// FixedBackoff is a Backoff that sleeps for a fixed amount of time after each failure.
type FixedBackoff struct {
	wait time.Duration
}

// NewFixedBackoff returns a new FixedBackoff.
func NewFixedBackoff(wait time.Duration) *FixedBackoff {
	return &FixedBackoff{wait: wait}
}

// New returns itself.
func (b *FixedBackoff) New() Backoff {
	return b
}

// Backoff sleeps for a fixed amount of time.
func (b *FixedBackoff) Backoff(iteration uint64) {
	time.Sleep(b.wait)
}

// DecorrelatedExponentialBackoff is a Backoff that sleeps for an exponentially increasing
// period of time, starting with a minimum wait, up to the maximum wait. Backoff times are
// heavily jittered to prevent thundering herd. The jitter amount is not correlated to the
// iteration number, but rather the previous sleep duration. This is the recommended backoff
// algorithm to use if you're unsure. See this article for details:
//
// https://www.awsarchitectureblog.com/2015/03/backoff.html
type DecorrelatedExponentialBackoff struct {
	minWait time.Duration
	maxWait time.Duration
}

// NewDecorrelatedExponentialBackoff returns a new DecorrelatedExponentialBackoff.
func NewDecorrelatedExponentialBackoff(minWait, maxWait time.Duration) *DecorrelatedExponentialBackoff {
	return &DecorrelatedExponentialBackoff{
		minWait: minWait,
		maxWait: maxWait,
	}
}

// New returns a new Backoff instance that can be called independently.
func (b *DecorrelatedExponentialBackoff) New() Backoff {
	return &decorrelatedExponentialBackoff{
		minWait: b.minWait,
		maxWait: b.maxWait,
	}
}

type decorrelatedExponentialBackoff struct {
	minWait  time.Duration
	maxWait  time.Duration
	lastWait time.Duration
}

// Backoff sleeps for a varying amount of time depending on the iteration number.
func (b *decorrelatedExponentialBackoff) Backoff(iteration uint64) {
	time.Sleep(b.getDuration())
}

func (b *decorrelatedExponentialBackoff) getDuration() time.Duration {
	if b.lastWait == zeroDuration {
		// For the first backoff, return the minimum wait time
		b.lastWait = b.minWait
		return b.minWait
	}

	// Each sleep is dependent on the previous sleep. Formula:
	//   sleep(n) = min(maxWait, random(minWait, sleep(n-1) * 3))
	var max int64
	if int64(b.lastWait) > int64(math.MaxInt64)/3 {
		// Would overflow our data type
		max = math.MaxInt64
	} else {
		max = int64(b.lastWait) * 3
	}

	minWaitF := float64(b.minWait)
	duration := minInt64(
		int64(b.maxWait),
		int64(rand.Float64()*(float64(max)-minWaitF)+minWaitF),
	)

	return time.Duration(duration)
}

func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// ExponentialBackoff is a Backoff that sleeps for an exponentially increasing period of
// time, starting with a minimum wait, up to the maximum wait. Some jitter is applied in
// order to prevent thundering herd.
type ExponentialBackoff struct {
	minWait time.Duration
	maxWait time.Duration
	jitter  bool
}

// NewExponentialBackoff returns a new ExponentialBackoff.
func NewExponentialBackoff(minWait, maxWait time.Duration) *ExponentialBackoff {
	return &ExponentialBackoff{minWait: minWait, maxWait: maxWait, jitter: true}
}

// New returns itself.
func (b *ExponentialBackoff) New() Backoff {
	return b
}

// Backoff sleeps for a varying amount of time depending on the iteration number.
func (b *ExponentialBackoff) Backoff(iteration uint64) {
	time.Sleep(b.getDuration(iteration))
}

func (b *ExponentialBackoff) getDuration(iteration uint64) time.Duration {
	if iteration >= numUint64Bits-1 { // would overflow
		return b.maxWait
	}

	// multiplier = 2 ^ iteration
	multiplier := int64(1)
	multiplier = multiplier << iteration

	var duration int64
	if int64(b.minWait) > int64(b.maxWait)/multiplier {
		// Would overflow - just use the max
		duration = int64(b.maxWait)
	} else {
		duration = int64(b.minWait) * multiplier
	}

	if b.jitter {
		// Apply some jitter to lessen thundering herd
		// https://en.wikipedia.org/wiki/Thundering_herd_problem

		// Final value will be in the range [duration/2, duration]
		duration = int64(float64(duration) * (0.5 * (rand.Float64() + 1.0)))
	}

	// Never sleep less than the minimum
	duration = maxInt64(duration, int64(b.minWait))

	return time.Duration(duration)
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}