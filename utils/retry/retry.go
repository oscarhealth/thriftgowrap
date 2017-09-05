// Package retry provides types and functions used to retry functions.
package retry

const defaultMaxAttempts uint64 = 1

// AnyErr returns true if err != nil.
func AnyErr(err error) bool {
	return err != nil
}

// Retrier performs a function a specified amount of times.
type Retrier struct {
	maxAttempts    uint64
	isRetriable    func(error) bool
	backoffFactory BackoffFactory
}

// Option is an optional argument to NewRetrier.
type Option func(retrier *Retrier)

// MaxAttemptsOption is the maximum number of attempts Retrier will make.
func MaxAttemptsOption(attempts uint64) Option {
	return func(retrier *Retrier) {
		retrier.maxAttempts = attempts
	}
}

// RetriableOption provides a function for checking whether an error should truly be considered an error.
func RetriableOption(isRetriable func(error) bool) Option {
	return func(retrier *Retrier) {
		retrier.isRetriable = isRetriable
	}
}

// BackoffOption provides the Backoff strategy to be used by Retrier.
func BackoffOption(backoff BackoffFactory) Option {
	return func(retrier *Retrier) {
		retrier.backoffFactory = backoff
	}
}

// NewRetrier returns a new Retrier constructed with optional arguments.
func NewRetrier(options ...Option) *Retrier {
	r := &Retrier{
		maxAttempts:    defaultMaxAttempts,
		isRetriable:    AnyErr,
		backoffFactory: DefaultBackoff,
	}

	for _, option := range options {
		option(r)
	}

	return r
}

// Do calls fn a specified amount of times. It will either succeed and return nil,
// or return the last error returned by fn.
func (r *Retrier) Do(fn func() error) error {
	var err error
	var i uint64
	backoff := r.backoffFactory.New()

	for ; i < r.maxAttempts; i++ {
		err = fn()
		if !r.isRetriable(err) {
			return err
		}
		// ensure we will still make another attempt before backing off
		if i+1 < r.maxAttempts {
			backoff.Backoff(i)
		}
	}
	return err
}
