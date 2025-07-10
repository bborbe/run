// Copyright (c) 2019 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package run_test

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/run"
)

type testRunnable struct {
	counter int
	result  error
	mutex   sync.Mutex
}

func (t *testRunnable) Run(context.Context) error {
	defer t.mutex.Unlock()
	t.mutex.Lock()
	t.counter++
	return t.result
}

func (t *testRunnable) Counter() int {
	defer t.mutex.Unlock()
	t.mutex.Lock()
	return t.counter
}

var _ = Describe("CancelOnFirstFinish", func() {
	It("TestCancelOnFirstFinishRunNothing", func() {
		err := run.CancelOnFirstFinish(context.Background())
		Expect(err).To(BeNil())
	})
	It("TestCancelOnFirstFinishReturnOnContextCancel", func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
		defer cancel()
		err := run.CancelOnFirstFinish(ctx,
			func(ctx context.Context) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.NewTicker(time.Minute).C:
					return nil
				}
			})
		Expect(err).To(Equal(context.DeadlineExceeded))
	})
	It("TestCancelOnFirstFinishRun", func() {
		r1 := new(testRunnable)
		err := run.CancelOnFirstFinish(context.Background(), r1.Run)
		Expect(err).To(BeNil())
		Expect(r1.Counter()).To(Equal(1))
	})
	It("TestCancelOnFirstFinishRunThree", func() {
		r1 := new(testRunnable)
		err := run.CancelOnFirstFinish(context.Background(), r1.Run, r1.Run, r1.Run)
		Expect(err).To(BeNil())
		Expect(r1.Counter()).To(BeNumerically(">=", 1))
	})
	It("TestCancelOnFirstFinishRunFail", func() {
		r1 := new(testRunnable)
		r1.result = errors.New("fail")
		err := run.CancelOnFirstFinish(context.Background(), r1.Run)
		Expect(err).NotTo(BeNil())
		Expect(r1.Counter()).To(Equal(1))
	})
})

var _ = Describe("CancelOnFirstFinishWait", func() {
	It("run nothing", func() {
		err := run.CancelOnFirstFinishWait(context.Background())
		Expect(err).To(BeNil())
	})
	It("run single function success", func() {
		r1 := new(testRunnable)
		err := run.CancelOnFirstFinishWait(context.Background(), r1.Run)
		Expect(err).To(BeNil())
		Expect(r1.Counter()).To(Equal(1))
	})
	It("run single function with error", func() {
		r1 := new(testRunnable)
		r1.result = errors.New("fail")
		err := run.CancelOnFirstFinishWait(context.Background(), r1.Run)
		Expect(err).NotTo(BeNil())
		Expect(r1.Counter()).To(Equal(1))
	})
	It("run multiple functions and wait for all", func() {
		r1 := new(testRunnable)
		r2 := new(testRunnable)
		r3 := new(testRunnable)
		err := run.CancelOnFirstFinishWait(context.Background(), r1.Run, r2.Run, r3.Run)
		Expect(err).To(BeNil())
		// At least one should have run, but due to cancellation, not all may complete
		Expect(r1.Counter() + r2.Counter() + r3.Counter()).To(BeNumerically(">=", 1))
	})
	It("collects all errors from completed functions", func() {
		errorFunc := func(ctx context.Context) error {
			return errors.New("test error")
		}
		successFunc := func(ctx context.Context) error {
			return nil
		}

		err := run.CancelOnFirstFinishWait(context.Background(), errorFunc, successFunc)
		Expect(err).To(HaveOccurred())
	})
	It("returns on context cancel", func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
		defer cancel()
		err := run.CancelOnFirstFinishWait(ctx,
			func(ctx context.Context) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.NewTicker(time.Minute).C:
					return nil
				}
			})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("context deadline exceeded"))
	})
})

var _ = Describe("CancelOnFirstError", func() {
	It("run nothing", func() {
		err := run.CancelOnFirstError(context.Background())
		Expect(err).To(BeNil())
	})
	It("run all", func() {
		r1 := new(testRunnable)
		err := run.CancelOnFirstError(context.Background(), r1.Run, r1.Run, r1.Run)
		Expect(err).To(BeNil())
		Expect(r1.Counter()).To(Equal(3))
	})
	It("returns error", func() {
		r1 := new(testRunnable)
		r2 := new(testRunnable)
		r2.result = errors.New("banana")
		r3 := new(testRunnable)
		err := run.CancelOnFirstError(context.Background(), r1.Run, r2.Run, r3.Run)
		Expect(err).To(HaveOccurred())
	})
	It("returns on context cancel", func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
		defer cancel()
		err := run.CancelOnFirstError(
			ctx,
			func(ctx context.Context) error {
				select {
				case <-time.NewTicker(time.Minute).C:
				case <-ctx.Done():
				}
				return nil
			},
		)
		Expect(err).To(BeNil())
	})
})

var _ = Describe("CancelOnFirstErrorWait", func() {
	It("run nothing", func() {
		err := run.CancelOnFirstErrorWait(context.Background())
		Expect(err).To(BeNil())
	})
	It("run all successfully", func() {
		r1 := new(testRunnable)
		err := run.CancelOnFirstErrorWait(context.Background(), r1.Run, r1.Run, r1.Run)
		Expect(err).To(BeNil())
		Expect(r1.Counter()).To(Equal(3))
	})
	It("cancels on first error and collects all errors", func() {
		r1 := new(testRunnable)
		r2 := new(testRunnable)
		r2.result = errors.New("banana")
		r3 := new(testRunnable)

		err := run.CancelOnFirstErrorWait(context.Background(), r1.Run, r2.Run, r3.Run)
		Expect(err).To(HaveOccurred())
		// Should have at least one error
		Expect(err.Error()).To(ContainSubstring("banana"))
	})
	It("returns single error", func() {
		errorFunc := func(ctx context.Context) error {
			return errors.New("test error")
		}

		err := run.CancelOnFirstErrorWait(context.Background(), errorFunc)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("test error"))
	})
	It("returns on context cancel", func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
		defer cancel()
		err := run.CancelOnFirstErrorWait(
			ctx,
			func(ctx context.Context) error {
				select {
				case <-time.NewTicker(time.Minute).C:
				case <-ctx.Done():
				}
				return nil
			},
		)
		Expect(err).To(BeNil())
	})
})

var _ = Describe("All", func() {

	It("returns not errors", func() {
		err := run.All(context.Background())
		Expect(err).To(BeNil())
	})
	It("run one", func() {
		r1 := new(testRunnable)
		err := run.All(context.Background(), r1.Run)
		Expect(err).To(BeNil())
		Expect(r1.Counter()).To(Equal(1))
	})
	It("with errorr", func() {
		r1 := new(testRunnable)
		r1.result = errors.New("fail")
		r2 := new(testRunnable)
		err := run.All(context.Background(), r1.Run, r2.Run)
		Expect(err).NotTo(BeNil())
		Expect(r1.Counter()).To(Equal(1))
		Expect(r2.Counter()).To(Equal(1))
	})
	It("run three", func() {
		r1 := new(testRunnable)
		err := run.All(context.Background(), r1.Run, r1.Run, r1.Run)
		Expect(err).To(BeNil())
		Expect(r1.Counter()).To(BeNumerically(">=", 1))
	})
})

var _ = Describe("Sequential", func() {
	It("returns if all funcs completed", func() {
		r1 := new(testRunnable)
		err := run.Sequential(context.Background(), r1.Run)
		Expect(err).To(BeNil())
	})
	It("returns nil if empty list", func() {
		err := run.Sequential(context.Background())
		Expect(err).To(BeNil())
	})
	It("cancels after first failed", func() {
		r1 := new(testRunnable)
		r2 := new(testRunnable)
		r2.result = errors.New("fail")
		r3 := new(testRunnable)
		err := run.Sequential(context.Background(), r1.Run, r2.Run, r3.Run)
		Expect(err).NotTo(BeNil())
		Expect(r1.Counter()).To(Equal(1))
		Expect(r2.Counter()).To(Equal(1))
		Expect(r3.Counter()).To(Equal(0))
	})
	It("returns if context is canceled", func() {
		f := func(ctx context.Context) error {
			<-ctx.Done()
			return errors.New("banana")
		}
		ctx, cancel := context.WithCancel(context.Background())
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			Expect(run.Sequential(ctx, f)).To(Equal(context.Canceled))
		}()
		cancel()
		wg.Wait()
	})
	It("does not call function if context is canceled", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		r1 := new(testRunnable)
		err := run.Sequential(ctx, r1.Run)
		Expect(err).To(Equal(context.Canceled))
		Expect(r1.Counter()).To(Equal(0))
	})
})

var _ = Describe("Delayed", func() {

	It("calls the given function", func() {
		r1 := new(testRunnable)
		fn := run.Delayed(r1.Run, time.Nanosecond)
		err := fn(context.Background())
		Expect(err).NotTo(HaveOccurred())
		Expect(r1.Counter()).To(Equal(1))
	})

	It("returns errors of the called functio", func() {
		r1 := new(testRunnable)
		r1.result = errors.New("banana")
		fn := run.Delayed(r1.Run, time.Nanosecond)
		err := fn(context.Background())
		Expect(err).To(Equal(r1.result))
		Expect(r1.Counter()).To(Equal(1))
	})

	It("does not call function if cancel in delay", func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
		defer cancel()
		r1 := new(testRunnable)
		fn := run.Delayed(r1.Run, time.Minute)
		err := fn(ctx)
		Expect(err).To(Equal(context.DeadlineExceeded))
		Expect(r1.Counter()).To(Equal(0))
	})
})

var _ = Describe("Run", func() {
	var errCh <-chan error
	var errList []error
	var funcs []run.Func
	JustBeforeEach(func() {
		errList = []error{}
		errCh = run.Run(context.Background(), funcs...)
		if errCh != nil {
			for err := range errCh {
				errList = append(errList, err)
			}
		}
	})
	Context("empty list", func() {
		BeforeEach(func() {
			funcs = []run.Func{}
		})
		It("returns nil if empty list", func() {
			Expect(errCh).To(BeNil())
		})
	})
	Context("one func", func() {
		BeforeEach(func() {
			funcs = []run.Func{
				func(ctx context.Context) error {
					return errors.New("banana")
				},
			}
		})
		It("returns channel", func() {
			Expect(errCh).NotTo(BeNil())
			Expect(errList).To(HaveLen(1))
		})
	})
	Context("num cpu funcs", func() {
		BeforeEach(func() {
			funcs = []run.Func{}
			for i := 0; i < runtime.NumCPU(); i++ {
				funcs = append(funcs, func(ctx context.Context) error {
					return errors.New("banana")
				})
			}
		})
		It("returns channel", func() {
			Expect(errCh).NotTo(BeNil())
			Expect(errList).To(HaveLen(runtime.NumCPU()))
		})
	})
	Context("num cpu funcs + 1", func() {
		BeforeEach(func() {
			funcs = []run.Func{}
			for i := 0; i < runtime.NumCPU()+1; i++ {
				funcs = append(funcs, func(ctx context.Context) error {
					return errors.New("banana")
				})
			}
		})
		It("returns channel", func() {
			Expect(errCh).NotTo(BeNil())
			Expect(errList).To(HaveLen(runtime.NumCPU() + 1))
		})
	})
})
