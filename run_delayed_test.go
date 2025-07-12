// Copyright (c) 2021 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package run_test

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/bborbe/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/run"
)

var _ = Describe("Delayed", func() {
	var ctx context.Context
	var executed bool
	var startTime time.Time

	BeforeEach(func() {
		ctx = context.Background()
		executed = false
		startTime = time.Now()
	})

	Context("with valid duration", func() {
		It("delays execution by specified duration", func() {
			delay := 50 * time.Millisecond
			fn := run.Delayed(func(ctx context.Context) error {
				executed = true
				return nil
			}, delay)

			err := fn(ctx)
			duration := time.Since(startTime)

			Expect(err).To(BeNil())
			Expect(executed).To(BeTrue())
			Expect(duration).To(BeNumerically(">=", delay))
			Expect(duration).To(BeNumerically("<", delay+20*time.Millisecond))
		})

		It("passes through function errors", func() {
			delay := 10 * time.Millisecond
			expectedErr := errors.New(ctx, "test error")
			fn := run.Delayed(func(ctx context.Context) error {
				return expectedErr
			}, delay)

			err := fn(ctx)

			Expect(err).To(Equal(expectedErr))
		})

		It("respects context cancellation", func() {
			delay := 100 * time.Millisecond
			cancelCtx, cancel := context.WithCancel(ctx)

			fn := run.Delayed(func(ctx context.Context) error {
				executed = true
				return nil
			}, delay)

			go func() {
				time.Sleep(20 * time.Millisecond)
				cancel()
			}()

			err := fn(cancelCtx)
			duration := time.Since(startTime)

			Expect(err).To(Equal(context.Canceled))
			Expect(executed).To(BeFalse())
			Expect(duration).To(BeNumerically("<", delay))
			Expect(duration).To(BeNumerically(">=", 20*time.Millisecond))
		})

		It("respects context timeout", func() {
			delay := 100 * time.Millisecond
			timeout := 30 * time.Millisecond
			timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			fn := run.Delayed(func(ctx context.Context) error {
				executed = true
				return nil
			}, delay)

			err := fn(timeoutCtx)
			duration := time.Since(startTime)

			Expect(err).To(Equal(context.DeadlineExceeded))
			Expect(executed).To(BeFalse())
			Expect(duration).To(BeNumerically(">=", timeout))
			Expect(duration).To(BeNumerically("<", delay))
		})
	})

	Context("with zero duration", func() {
		It("executes immediately", func() {
			var counter uint64
			fn := run.Delayed(func(ctx context.Context) error {
				atomic.AddUint64(&counter, 1)
				return nil
			}, 0)

			err := fn(ctx)
			duration := time.Since(startTime)

			Expect(err).To(BeNil())
			Expect(atomic.LoadUint64(&counter)).To(Equal(uint64(1)))
			Expect(duration).To(BeNumerically("<", 10*time.Millisecond))
		})
	})

	Context("concurrent execution", func() {
		It("handles multiple delayed functions correctly", func() {
			var counter uint64
			numFunctions := 5
			delay := 20 * time.Millisecond

			functions := make([]run.Func, numFunctions)
			for i := 0; i < numFunctions; i++ {
				functions[i] = run.Delayed(func(ctx context.Context) error {
					atomic.AddUint64(&counter, 1)
					return nil
				}, delay)
			}

			err := run.All(ctx, functions...)
			duration := time.Since(startTime)

			Expect(err).To(BeNil())
			Expect(atomic.LoadUint64(&counter)).To(Equal(uint64(numFunctions)))
			Expect(duration).To(BeNumerically(">=", delay))
		})
	})
})
