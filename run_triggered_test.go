// Copyright (c) 2021 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package run_test

import (
	"context"
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/run"
	"github.com/bborbe/run/mocks"
)

var _ = Describe("Triggered", func() {
	var ch chan struct{}
	var fn run.Func
	var runnable *mocks.Runnable
	var err error
	var ctx context.Context
	var cancel context.CancelFunc
	BeforeEach(func() {
		ctx = context.Background()
		ch = make(chan struct{}, 8)
		runnable = &mocks.Runnable{}
		fn = run.Triggered(runnable.Run, ch)
	})
	AfterEach(func() {
		close(ch)
	})
	Context("success", func() {
		BeforeEach(func() {
			ch <- struct{}{}
			err = fn(ctx)
		})
		It("returns no error", func() {
			Expect(err).To(BeNil())
		})
		It("calls fn", func() {
			Expect(runnable.RunCallCount()).To(Equal(1))
		})
	})
	Context("fails", func() {
		BeforeEach(func() {
			runnable.RunReturns(errors.New("banana"))
			ch <- struct{}{}
			err = fn(ctx)
		})
		It("returns error", func() {
			Expect(err).NotTo(BeNil())
		})
		It("calls fn", func() {
			Expect(runnable.RunCallCount()).To(Equal(1))
		})
	})
	Context("context canceled", func() {
		BeforeEach(func() {
			ctx, cancel = context.WithCancel(ctx)
			cancel()
			err = fn(ctx)
		})
		It("returns cancel err", func() {
			Expect(err).To(Equal(context.Canceled))
		})
		It("does not call fn", func() {
			Expect(runnable.RunCallCount()).To(Equal(0))
		})
	})
	Context("wait for trigger", func() {
		BeforeEach(func() {
			ctx, cancel = context.WithTimeout(ctx, 50*time.Millisecond)
			err = fn(ctx)
		})
		It("returns error", func() {
			Expect(err).To(Equal(context.DeadlineExceeded))
		})
		It("does not call fn", func() {
			Expect(runnable.RunCallCount()).To(Equal(0))
		})
	})
})
