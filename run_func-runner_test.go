// Copyright (c) 2025 Benjamin Borbe All rights reserved.
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

var _ = Describe("FuncRunner", func() {
	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("FuncRunnerFunc", func() {
		It("should implement FuncRunner interface", func() {
			var runner run.FuncRunner = run.FuncRunnerFunc(func(runFunc run.Func) error {
				return runFunc(ctx)
			})
			Expect(runner).NotTo(BeNil())
		})

		It("should execute the provided function", func() {
			called := false
			runner := run.FuncRunnerFunc(func(runFunc run.Func) error {
				return runFunc(ctx)
			})

			err := runner.Run(func(ctx context.Context) error {
				called = true
				return nil
			})

			Expect(err).To(BeNil())
			Expect(called).To(BeTrue())
		})

		It("should propagate errors from the executed function", func() {
			expectedErr := errors.New("test error")
			runner := run.FuncRunnerFunc(func(runFunc run.Func) error {
				return runFunc(ctx)
			})

			err := runner.Run(func(ctx context.Context) error {
				return expectedErr
			})

			Expect(err).To(Equal(expectedErr))
		})

		It("should allow wrapping and transforming the function", func() {
			wrappedCalled := false
			runner := run.FuncRunnerFunc(func(runFunc run.Func) error {
				wrappedCalled = true
				// Transform the error
				err := runFunc(ctx)
				if err != nil {
					return errors.New("wrapped: " + err.Error())
				}
				return nil
			})

			originalErr := errors.New("original error")
			err := runner.Run(func(ctx context.Context) error {
				return originalErr
			})

			Expect(wrappedCalled).To(BeTrue())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("wrapped: original error"))
		})

		It("should respect context cancellation", func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			runner := run.FuncRunnerFunc(func(runFunc run.Func) error {
				return runFunc(ctx)
			})

			err := runner.Run(func(ctx context.Context) error {
				return ctx.Err()
			})

			Expect(err).To(Equal(context.Canceled))
		})

		It("should allow multiple executions", func() {
			execCount := 0
			runner := run.FuncRunnerFunc(func(runFunc run.Func) error {
				execCount++
				return runFunc(ctx)
			})

			for i := 0; i < 3; i++ {
				err := runner.Run(func(ctx context.Context) error {
					return nil
				})
				Expect(err).To(BeNil())
			}

			Expect(execCount).To(Equal(3))
		})

		It("should allow decorator pattern for adding behavior", func() {
			beforeCalled := false
			afterCalled := false

			runner := run.FuncRunnerFunc(func(runFunc run.Func) error {
				beforeCalled = true
				err := runFunc(ctx)
				afterCalled = true
				return err
			})

			functionCalled := false
			err := runner.Run(func(ctx context.Context) error {
				functionCalled = true
				Expect(beforeCalled).To(BeTrue(), "before should be called first")
				Expect(afterCalled).To(BeFalse(), "after should not be called yet")
				return nil
			})

			Expect(err).To(BeNil())
			Expect(beforeCalled).To(BeTrue())
			Expect(functionCalled).To(BeTrue())
			Expect(afterCalled).To(BeTrue())
		})

		It("should handle nil function gracefully when wrapper checks", func() {
			runner := run.FuncRunnerFunc(func(runFunc run.Func) error {
				if runFunc == nil {
					return errors.New("nil function")
				}
				return runFunc(ctx)
			})

			err := runner.Run(nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("nil function"))
		})
	})
})
