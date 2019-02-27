// Copyright (c) 2019 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package run_test

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/bborbe/run"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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
					return nil
				case <-time.NewTicker(time.Minute).C:
					return nil
				}
			})
		Expect(err).To(BeNil())
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
			Expect(run.Sequential(ctx, f)).To(BeNil())
		}()
		cancel()
		wg.Wait()
	})
	It("does not call function if context is canceled", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		r1 := new(testRunnable)
		err := run.Sequential(ctx, r1.Run)
		Expect(err).To(BeNil())
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
		Expect(err).NotTo(HaveOccurred())
		Expect(r1.Counter()).To(Equal(0))
	})
})
