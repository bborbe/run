package run_test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	. "github.com/bborbe/assert"
	"github.com/bborbe/run"
	"github.com/golang/glog"
)

func TestMain(m *testing.M) {
	exit := m.Run()
	glog.Flush()
	os.Exit(exit)
}

func TestCancelOnFirstFinishRunNothing(t *testing.T) {
	err := run.CancelOnFirstFinish(context.Background())
	if err := AssertThat(err, NilValue()); err != nil {
		t.Fatal(err)
	}
}

func TestCancelOnFirstFinishReturnOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-time.NewTicker(100 * time.Millisecond).C
		cancel()
	}()
	err := run.CancelOnFirstFinish(ctx,
		func(ctx context.Context) error {
			<-time.NewTicker(time.Minute).C
			return nil
		})
	if err := AssertThat(err, NilValue()); err != nil {
		t.Fatal(err)
	}
}

func TestCancelOnFirstFinishRun(t *testing.T) {
	r1 := new(testRunnable)
	err := run.CancelOnFirstFinish(context.Background(), r1.Run)
	if err := AssertThat(err, NilValue()); err != nil {
		t.Fatal(err)
	}
	if err := AssertThat(r1.counter, Is(1)); err != nil {
		t.Fatal(err)
	}
}

func TestCancelOnFirstFinishRunThree(t *testing.T) {
	r1 := new(testRunnable)
	err := run.CancelOnFirstFinish(context.Background(), r1.Run, r1.Run, r1.Run)
	if err := AssertThat(err, NilValue()); err != nil {
		t.Fatal(err)
	}
	if err := AssertThat(r1.counter, Ge(1)); err != nil {
		t.Fatal(err)
	}
}

func TestCancelOnFirstFinishRunFail(t *testing.T) {
	r1 := new(testRunnable)
	r1.result = errors.New("fail")
	err := run.CancelOnFirstFinish(context.Background(), r1.Run)
	if err := AssertThat(err, NotNilValue()); err != nil {
		t.Fatal(err)
	}
	if err := AssertThat(r1.counter, Is(1)); err != nil {
		t.Fatal(err)
	}
}

func TestCancelOnFirstErrorRunNothing(t *testing.T) {
	err := run.CancelOnFirstError(context.Background())
	if err := AssertThat(err, NilValue()); err != nil {
		t.Fatal(err)
	}
}

func TestCancelOnFirstErrorReturnOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-time.NewTicker(100 * time.Millisecond).C
		cancel()
	}()
	err := run.CancelOnFirstError(ctx,
		func(ctx context.Context) error {
			<-time.NewTicker(time.Minute).C
			return nil
		})
	if err := AssertThat(err, NilValue()); err != nil {
		t.Fatal(err)
	}
}

type testRunnable struct {
	counter int
	result  error
	mutex   sync.Mutex
}

func (t *testRunnable) Run(context.Context) error {
	t.mutex.Lock()
	t.counter++
	t.mutex.Unlock()
	return t.result
}

func TestAllRunNothing(t *testing.T) {
	err := run.All(context.Background())
	if err := AssertThat(err, NilValue()); err != nil {
		t.Fatal(err)
	}
}

func TestAllRunOne(t *testing.T) {
	r1 := new(testRunnable)
	err := run.All(context.Background(), r1.Run)
	if err := AssertThat(err, NilValue()); err != nil {
		t.Fatal(err)
	}
	if err := AssertThat(r1.counter, Is(1)); err != nil {
		t.Fatal(err)
	}
}

func TestAllWithError(t *testing.T) {
	r1 := new(testRunnable)
	r1.result = errors.New("fail")
	r2 := new(testRunnable)
	err := run.All(context.Background(), r1.Run, r2.Run)
	if err := AssertThat(err, NotNilValue()); err != nil {
		t.Fatal(err)
	}
	if err := AssertThat(r1.counter, Is(1)); err != nil {
		t.Fatal(err)
	}
	if err := AssertThat(r2.counter, Is(1)); err != nil {
		t.Fatal(err)
	}
}

func TestAllRunThree(t *testing.T) {
	r1 := new(testRunnable)
	err := run.All(context.Background(), r1.Run, r1.Run, r1.Run)
	if err := AssertThat(err, NilValue()); err != nil {
		t.Fatal(err)
	}
	if err := AssertThat(r1.counter, Ge(1)); err != nil {
		t.Fatal(err)
	}
}

func TestSequential(t *testing.T) {
	r1 := new(testRunnable)
	r2 := new(testRunnable)
	r2.result = errors.New("fail")
	r3 := new(testRunnable)
	err := run.Sequential(context.Background(), r1.Run, r2.Run, r3.Run)
	if err := AssertThat(err, NotNilValue()); err != nil {
		t.Fatal(err)
	}
	if err := AssertThat(r1.counter, Eq(1)); err != nil {
		t.Fatal(err)
	}
	if err := AssertThat(r2.counter, Eq(1)); err != nil {
		t.Fatal(err)
	}
	if err := AssertThat(r3.counter, Eq(0)); err != nil {
		t.Fatal(err)
	}
}

func TestSequentialCancelsOnContextCancel(t *testing.T) {
	f := func(ctx context.Context) error {
		<-ctx.Done()
		return errors.New("banana")
	}
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := AssertThat(run.Sequential(ctx, f), NilValue()); err != nil {
			t.Fatal(err)
		}
	}()
	cancel()
	wg.Wait()
}

func TestSequentialDoesNotCallFunctionIfContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r1 := new(testRunnable)
	if err := AssertThat(run.Sequential(ctx, r1.Run), NilValue()); err != nil {
		t.Fatal(err)
	}
	if err := AssertThat(r1.counter, Eq(0)); err != nil {
		t.Fatal(err)
	}
}
