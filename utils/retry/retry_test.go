package retry

import (
	"errors"
	"testing"
)

func makeFn(i int64) func() error {
	var count int64
	return func() error {
		if count < i {
			count++
			return errors.New("err")
		}
		return nil
	}
}

func TestRetrier_Do(t *testing.T) {
	fn0 := makeFn(3)
	retrier0 := NewRetrier(MaxAttemptsOption(3))
	err0 := retrier0.Do(fn0)
	if err0 == nil {
		t.Error("Expected retrier0 to return err")
	}

	fn1 := makeFn(3)
	retrier1 := NewRetrier(MaxAttemptsOption(4))
	err1 := retrier1.Do(fn1)
	if err1 != nil {
		t.Error("Expected retrier1 to return nil")
	}

	fn2 := makeFn(3)
	retrier2 := NewRetrier(MaxAttemptsOption(3), RetriableOption(func(error) bool { return false }))
	err2 := retrier2.Do(fn2)
	if err2.Error() != "err" {
		t.Error("Expected retrier2 to return error")
	}

	var iterations []uint64
	expected := []uint64{0, 1}
	backoffFunc := func(i uint64) {
		iterations = append(iterations, i)
	}
	backoff := FunctionalBackoff(backoffFunc)

	fn3 := makeFn(3)
	retrier3 := NewRetrier(MaxAttemptsOption(3), BackoffOption(backoff))
	err3 := retrier3.Do(fn3)
	if err3 == nil {
		t.Error("Expected retrier3 to return err")
	}

	if len(iterations) != len(expected) {
		t.Errorf("expected len(iterations) == %d, received %d", len(expected), len(iterations))
	}

	for i := 0; i < len(iterations); i++ {
		if iterations[i] != expected[i] {
			t.Errorf("expected[%d] expected %d, received %d", i, expected[i], iterations[i])
		}
	}

}

func TestNewRetrier(t *testing.T) {
	retrier0 := NewRetrier()
	if retrier0.maxAttempts != defaultMaxAttempts {
		t.Errorf("retrier0.maxAttempts != defaultMaxAttempts, got %d instead", retrier0.maxAttempts)
	}
}
