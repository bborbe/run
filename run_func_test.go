// Copyright (c) 2019 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package run_test

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/run"
)

var _ = Describe("Func", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("Run", func() {
		Context("successful function", func() {
			It("returns nil when function succeeds", func() {
				fn := run.Func(func(ctx context.Context) error {
					return nil
				})
				err := fn.Run(ctx)
				Expect(err).To(BeNil())
			})
		})

		Context("function with error", func() {
			It("returns error when function fails", func() {
				expectedError := errors.New("test error")
				fn := run.Func(func(ctx context.Context) error {
					return expectedError
				})
				err := fn.Run(ctx)
				Expect(err).To(Equal(expectedError))
			})
		})

		Context("function with context", func() {
			It("passes context to function", func() {
				var receivedCtx context.Context
				fn := run.Func(func(ctx context.Context) error {
					receivedCtx = ctx
					return nil
				})
				err := fn.Run(ctx)
				Expect(err).To(BeNil())
				Expect(receivedCtx).To(Equal(ctx))
			})
		})

		Context("cancelled context", func() {
			It("can handle cancelled context", func() {
				cancelledCtx, cancel := context.WithCancel(ctx)
				cancel()

				fn := run.Func(func(ctx context.Context) error {
					return ctx.Err()
				})
				err := fn.Run(cancelledCtx)
				Expect(err).To(Equal(context.Canceled))
			})
		})
	})

	Describe("Runnable interface", func() {
		It("Func implements Runnable interface", func() {
			fn := run.Func(func(ctx context.Context) error {
				return nil
			})

			// Test that Func can be used as Runnable
			var runnable run.Runnable = fn
			err := runnable.Run(ctx)
			Expect(err).To(BeNil())
		})
	})
})
