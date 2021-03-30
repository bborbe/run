// Copyright (c) 2020 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package run_test

import (
	"context"
	"errors"
	"time"

	"github.com/bborbe/run"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Retry", func() {
	var err error
	var callCounter int
	var innerResult error
	var innerFn func(ctx context.Context) error
	var ctx context.Context
	BeforeEach(func() {
		ctx = context.Background()
		innerResult = nil
		callCounter = 0
		innerFn = func(ctx context.Context) error {
			callCounter++
			return innerResult
		}
	})
	Context("normal context", func() {
		BeforeEach(func() {
			fn := run.Retry(innerFn, 0, 0)
			err = fn(ctx)
		})
		It("returns no error", func() {
			Expect(err).To(BeNil())
		})
		It("calls inner func", func() {
			Expect(callCounter).To(Equal(1))
		})
	})
	Context("limit 1 and no error", func() {
		BeforeEach(func() {
			fn := run.Retry(innerFn, 1, 0)
			err = fn(ctx)
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
			innerResult = errors.New("banana")
			fn := run.Retry(innerFn, 1, 0)
			err = fn(ctx)
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
			innerResult = errors.New("banana")
			fn := run.Retry(innerFn, 1, 100*time.Millisecond)
			err = fn(ctx)
		})
		It("returns no error", func() {
			Expect(err).NotTo(BeNil())
		})
		It("calls inner func", func() {
			Expect(callCounter).To(Equal(2))
		})
	})
	Context("cancel while waiting for retry", func() {
		BeforeEach(func() {
			ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
			defer cancel()
			innerResult = errors.New("banana")
			fn := run.Retry(innerFn, 1, time.Hour)
			err = fn(ctx)
		})
		It("returns deadline error", func() {
			Expect(err).To(Equal(context.DeadlineExceeded))
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
			fn := run.Retry(innerFn, 0, 0)
			err = fn(ctx)
		})
		It("does not call inner if ctx is canncel func", func() {
			Expect(callCounter).To(Equal(0))
		})
		It("returns no error", func() {
			Expect(err).To(Equal(context.Canceled))
		})
	})
})
