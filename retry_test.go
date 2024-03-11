// Copyright (c) 2020 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package run_test

import (
	"context"
	stderrors "errors"
	"fmt"
	"time"

	"github.com/bborbe/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/run"
	"github.com/bborbe/run/mocks"
)

var _ = Describe("Retry", func() {
	var err error
	var callCounter int
	var innerResult error
	var innerFn func(ctx context.Context) error
	var ctx context.Context
	var backoff run.Backoff
	BeforeEach(func() {
		ctx = context.Background()
		innerResult = nil
		callCounter = 0
		innerFn = func(ctx context.Context) error {
			callCounter++
			return innerResult
		}
		backoff = run.Backoff{}
	})
	JustBeforeEach(func() {
		fn := run.Retry(backoff, innerFn)
		err = fn(ctx)
	})
	Context("normal context", func() {
		It("returns no error", func() {
			Expect(err).To(BeNil())
		})
		It("calls inner func", func() {
			Expect(callCounter).To(Equal(1))
		})
	})
	Context("limit 1 and no error", func() {
		BeforeEach(func() {
			backoff.Retries = 1
		})
		It("returns no error", func() {
			Expect(err).To(BeNil())
		})
		It("calls inner func", func() {
			Expect(callCounter).To(Equal(1))
		})
	})
	Context("limit 1 and error", func() {
		BeforeEach(func() {
			innerResult = stderrors.New("banana")
			backoff.Retries = 1
		})
		It("returns no error", func() {
			Expect(err).NotTo(BeNil())
		})
		It("calls inner func", func() {
			Expect(callCounter).To(Equal(2))
		})
	})
	Context("limit 1 and error and delay", func() {
		BeforeEach(func() {
			innerResult = stderrors.New("banana")
			backoff.Retries = 1
			backoff.Delay = 100 * time.Millisecond
		})
		It("returns no error", func() {
			Expect(err).NotTo(BeNil())
		})
		It("calls inner func", func() {
			Expect(callCounter).To(Equal(2))
		})
	})
	Context("cancel while waiting for retry", func() {
		var cancel context.CancelFunc
		BeforeEach(func() {
			ctx, cancel = context.WithTimeout(ctx, time.Millisecond)
			innerResult = stderrors.New("banana")
			backoff.Retries = 10
			backoff.Delay = time.Hour
		})
		AfterEach(func() {
			defer cancel()
		})
		It("returns deadline error", func() {
			Expect(err).NotTo(BeNil())
			Expect(errors.Cause(err)).To(Equal(context.DeadlineExceeded))
		})
		It("calls inner func", func() {
			Expect(callCounter).To(Equal(1))
		})
	})
	Context("canceled context", func() {
		var cancel context.CancelFunc
		BeforeEach(func() {
			ctx, cancel = context.WithCancel(ctx)
			cancel()
		})
		It("does not call inner if ctx is canncel func", func() {
			Expect(callCounter).To(Equal(0))
		})
		It("returns no error", func() {
			Expect(err).To(Equal(context.Canceled))
		})
	})
	Context("isRetryable", func() {
		BeforeEach(func() {
			innerResult = fmt.Errorf("banana")
			backoff.Retries = 3
		})
		Context("is able to retry", func() {
			BeforeEach(func() {
				backoff.IsRetryAble = func(err error) bool {
					return true
				}
			})
			It("return err", func() {
				Expect(err).NotTo(BeNil())
			})
			It("retries", func() {
				Expect(callCounter).To(Equal(4))
			})
		})
		Context("is not able to retry", func() {
			BeforeEach(func() {
				backoff.IsRetryAble = func(err error) bool {
					return false
				}
			})
			It("return err", func() {
				Expect(err).NotTo(BeNil())
			})
			It("not retries", func() {
				Expect(callCounter).To(Equal(1))
			})
		})
	})
	Context("Factor", func() {
		BeforeEach(func() {
			run.DefaultWaiter = &mocks.Waiter{}
		})
	})
})

var _ = Describe("RetryWaiter", func() {
	var err error
	var callCounter int
	var innerResult error
	var innerFn func(ctx context.Context) error
	var ctx context.Context
	var backoff run.Backoff
	var waiter *mocks.Waiter
	BeforeEach(func() {
		ctx = context.Background()
		innerResult = nil
		callCounter = 0
		innerFn = func(ctx context.Context) error {
			callCounter++
			return innerResult
		}
		waiter = &mocks.Waiter{}
		backoff = run.Backoff{}
	})
	JustBeforeEach(func() {
		fn := run.RetryWaiter(backoff, waiter, innerFn)
		err = fn(ctx)
	})
	Context("Factor = 0", func() {
		BeforeEach(func() {
			backoff.Delay = time.Minute
			backoff.Factor = 0
			backoff.Retries = 3
			innerResult = stderrors.New("banana")
		})
		It("return error", func() {
			Expect(err).NotTo(BeNil())
		})
		It("calls wait 3 times", func() {
			Expect(waiter.WaitCallCount()).To(Equal(3))
			{
				argCtx, argDuration := waiter.WaitArgsForCall(0)
				Expect(argCtx).NotTo(BeNil())
				Expect(argDuration).To(Equal(time.Minute))
			}
			{
				argCtx, argDuration := waiter.WaitArgsForCall(1)
				Expect(argCtx).NotTo(BeNil())
				Expect(argDuration).To(Equal(time.Minute))
			}
			{
				argCtx, argDuration := waiter.WaitArgsForCall(2)
				Expect(argCtx).NotTo(BeNil())
				Expect(argDuration).To(Equal(time.Minute))
			}
		})
	})
	Context("Factor = 2", func() {
		BeforeEach(func() {
			backoff.Delay = time.Minute
			backoff.Factor = 2
			backoff.Retries = 3
			innerResult = stderrors.New("banana")
		})
		It("return error", func() {
			Expect(err).NotTo(BeNil())
		})
		It("calls wait 3 times", func() {
			Expect(waiter.WaitCallCount()).To(Equal(3))
			{
				argCtx, argDuration := waiter.WaitArgsForCall(0)
				Expect(argCtx).NotTo(BeNil())
				Expect(argDuration).To(Equal(time.Minute))
			}
			{
				argCtx, argDuration := waiter.WaitArgsForCall(1)
				Expect(argCtx).NotTo(BeNil())
				Expect(argDuration).To(Equal(3 * time.Minute))
			}
			{
				argCtx, argDuration := waiter.WaitArgsForCall(2)
				Expect(argCtx).NotTo(BeNil())
				Expect(argDuration).To(Equal(5 * time.Minute))
			}
		})
	})
})
