package run

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	. "github.com/bborbe/assert"
	"github.com/golang/glog"
)

func TestMain(m *testing.M) {
	exit := m.Run()
	glog.Flush()
	os.Exit(exit)
}

func TestRunNothing(t *testing.T) {
	err := CancelOnFirstFinish()
	if err := AssertThat(err, NilValue()); err != nil {
		t.Fatal(err)
	}
}

func TestRun(t *testing.T) {
	r1 := new(testRunnable)
	err := CancelOnFirstFinish(r1.Run)
	if err := AssertThat(err, NilValue()); err != nil {
		t.Fatal(err)
	}
	if err := AssertThat(r1.counter, Is(1)); err != nil {
		t.Fatal(err)
	}
}

func TestRunThree(t *testing.T) {
	r1 := new(testRunnable)
	err := CancelOnFirstFinish(r1.Run, r1.Run, r1.Run)
	if err := AssertThat(err, NilValue()); err != nil {
		t.Fatal(err)
	}
	if err := AssertThat(r1.counter, Ge(1)); err != nil {
		t.Fatal(err)
	}
}

func TestRunFail(t *testing.T) {
	r1 := new(testRunnable)
	r1.result = errors.New("fail")
	err := CancelOnFirstFinish(r1.Run)
	if err := AssertThat(err, NotNilValue()); err != nil {
		t.Fatal(err)
	}
	if err := AssertThat(r1.counter, Is(1)); err != nil {
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
