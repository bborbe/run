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
	t.mutex.Lock()
	t.counter++
	t.mutex.Unlock()
	return t.result
}

var _ = Describe("Run", func() {
	It("TestCancelOnFirstFinishRunNothing", func() {
		err := run.CancelOnFirstFinish(context.Background())
		Expect(err).To(BeNil())
	})
	It("TestCancelOnFirstFinishReturnOnContextCancel", func() {
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
		Expect(err).To(BeNil())
	})
	It("TestCancelOnFirstFinishRun", func() {
		r1 := new(testRunnable)
		err := run.CancelOnFirstFinish(context.Background(), r1.Run)
		Expect(err).To(BeNil())
		Expect(r1.counter).To(Equal(1))
	})
	It("TestCancelOnFirstFinishRunThree", func() {
		r1 := new(testRunnable)
		err := run.CancelOnFirstFinish(context.Background(), r1.Run, r1.Run, r1.Run)
		Expect(err).To(BeNil())
		Expect(r1.counter).To(BeNumerically(">=", 1))
	})
	It("TestCancelOnFirstFinishRunFail", func() {
		r1 := new(testRunnable)
		r1.result = errors.New("fail")
		err := run.CancelOnFirstFinish(context.Background(), r1.Run)
		Expect(err).NotTo(BeNil())
		Expect(r1.counter).To(Equal(1))
	})
	It("TestCancelOnFirstErrorRunNothing", func() {
		err := run.CancelOnFirstError(context.Background())
		Expect(err).To(BeNil())
	})
	It("TestCancelOnFirstErrorReturnOnContextCancel", func() {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			<-time.NewTicker(100 * time.Millisecond).C
			cancel()
		}()
		err := run.CancelOnFirstError(
			ctx,
			func(ctx context.Context) error {
				<-time.NewTicker(time.Minute).C
				return nil
			},
		)
		Expect(err).To(BeNil())
	})
	It("TestAllRunNothing", func() {
		err := run.All(context.Background())
		Expect(err).To(BeNil())
	})
	It("TestAllRunOne", func() {
		r1 := new(testRunnable)
		err := run.All(context.Background(), r1.Run)
		Expect(err).To(BeNil())
		Expect(r1.counter).To(Equal(1))
	})
	It("TestAllWithError", func() {
		r1 := new(testRunnable)
		r1.result = errors.New("fail")
		r2 := new(testRunnable)
		err := run.All(context.Background(), r1.Run, r2.Run)
		Expect(err).NotTo(BeNil())
		Expect(r1.counter).To(Equal(1))
		Expect(r2.counter).To(Equal(1))
	})
	It("TestAllRunThree", func() {
		r1 := new(testRunnable)
		err := run.All(context.Background(), r1.Run, r1.Run, r1.Run)
		Expect(err).To(BeNil())
		Expect(r1.counter).To(BeNumerically(">=", 1))
	})
	It("TestSequential", func() {
		r1 := new(testRunnable)
		r2 := new(testRunnable)
		r2.result = errors.New("fail")
		r3 := new(testRunnable)
		err := run.Sequential(context.Background(), r1.Run, r2.Run, r3.Run)
		Expect(err).NotTo(BeNil())
		Expect(r1.counter).To(Equal(1))
		Expect(r2.counter).To(Equal(1))
		Expect(r3.counter).To(Equal(0))
	})
	It("TestSequentialCancelsOnContextCancel", func() {
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
	It("TestSequentialDoesNotCallFunctionIfContextIsCanceled", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		r1 := new(testRunnable)
		err := run.Sequential(ctx, r1.Run)
		Expect(err).To(BeNil())
		Expect(r1.counter).To(Equal(0))
	})
})
