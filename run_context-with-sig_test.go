// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package run_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/run"
)

var _ = Describe("ContextWithSig", func() {
	var ctx context.Context
	var cancel context.CancelFunc
	var sigCtx context.Context

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		sigCtx = run.ContextWithSig(ctx)
	})

	AfterEach(func() {
		cancel()
	})

	Context("Parent context cancellation", func() {
		It("cancels signal context when parent context is cancelled", func() {
			// Cancel parent context
			go func() {
				time.Sleep(50 * time.Millisecond)
				cancel()
			}()

			// Wait for signal context to be cancelled
			select {
			case <-sigCtx.Done():
				// Context should be cancelled due to parent cancellation
				Expect(sigCtx.Err()).To(Equal(context.Canceled))
			case <-time.After(200 * time.Millisecond):
				Fail("Context was not cancelled within timeout")
			}
		})

		It("cancels signal context immediately when parent is already cancelled", func() {
			parentCtx, parentCancel := context.WithCancel(context.Background())
			parentCancel() // Cancel before creating signal context

			sigCtx := run.ContextWithSig(parentCtx)

			// Signal context should be cancelled immediately
			select {
			case <-sigCtx.Done():
				Expect(sigCtx.Err()).To(Equal(context.Canceled))
			case <-time.After(100 * time.Millisecond):
				Fail("Context should have been cancelled immediately")
			}
		})
	})

	Context("Multiple signal context creation", func() {
		It("handles multiple signal contexts independently", func() {
			ctx1, cancel1 := context.WithCancel(context.Background())
			ctx2, cancel2 := context.WithCancel(context.Background())
			defer cancel2()

			sigCtx1 := run.ContextWithSig(ctx1)
			sigCtx2 := run.ContextWithSig(ctx2)

			// Cancel first context
			cancel1()

			// First signal context should be cancelled
			select {
			case <-sigCtx1.Done():
				Expect(sigCtx1.Err()).To(Equal(context.Canceled))
			case <-time.After(100 * time.Millisecond):
				Fail("First context was not cancelled")
			}

			// Second signal context should still be active
			select {
			case <-sigCtx2.Done():
				Fail("Second context should not be cancelled")
			case <-time.After(50 * time.Millisecond):
				// Expected behavior - context is still active
			}
		})
	})

	Context("Context values preservation", func() {
		It("preserves context values from parent context", func() {
			type testKey string
			key := testKey("test")
			value := "test-value"

			// Create parent context with value
			parentCtx := context.WithValue(context.Background(), key, value)
			sigCtx := run.ContextWithSig(parentCtx)

			// Value should be preserved
			Expect(sigCtx.Value(key)).To(Equal(value))
		})

		It("preserves deadline from parent context", func() {
			deadline := time.Now().Add(100 * time.Millisecond)
			parentCtx, parentCancel := context.WithDeadline(context.Background(), deadline)
			defer parentCancel()

			sigCtx := run.ContextWithSig(parentCtx)

			// Deadline should be preserved
			actualDeadline, hasDeadline := sigCtx.Deadline()
			Expect(hasDeadline).To(BeTrue())
			Expect(actualDeadline).To(BeTemporally("~", deadline, time.Millisecond))
		})
	})

	Context("Concurrent signal handling", func() {
		It("handles concurrent signal context creation and cancellation", func() {
			const numContexts = 10
			contexts := make([]context.Context, numContexts)
			cancels := make([]context.CancelFunc, numContexts)
			sigContexts := make([]context.Context, numContexts)

			// Create multiple contexts concurrently
			for i := 0; i < numContexts; i++ {
				contexts[i], cancels[i] = context.WithCancel(context.Background())
				sigContexts[i] = run.ContextWithSig(contexts[i])
			}

			// Cancel all contexts concurrently
			for i := 0; i < numContexts; i++ {
				go func(idx int) {
					time.Sleep(time.Duration(idx) * time.Millisecond)
					cancels[idx]()
				}(i)
			}

			// Wait for all signal contexts to be cancelled
			for i := 0; i < numContexts; i++ {
				select {
				case <-sigContexts[i].Done():
					Expect(sigContexts[i].Err()).To(Equal(context.Canceled))
				case <-time.After(200 * time.Millisecond):
					Fail("Context was not cancelled within timeout")
				}
			}
		})
	})

	Context("Signal handling behavior", func() {
		It("creates proper signal context structure", func() {
			// Test that the signal context is properly structured
			sigCtx := run.ContextWithSig(context.Background())

			// The signal context should not be cancelled initially
			select {
			case <-sigCtx.Done():
				Fail("Signal context should not be cancelled initially")
			case <-time.After(10 * time.Millisecond):
				// Expected behavior - context is not cancelled
			}

			// Context should have proper type
			Expect(sigCtx).NotTo(BeNil())
			Expect(sigCtx.Err()).To(BeNil())
		})

		It("handles signal context cleanup properly", func() {
			// Create and immediately cancel parent context
			parentCtx, parentCancel := context.WithCancel(context.Background())
			sigCtx := run.ContextWithSig(parentCtx)

			// Cancel parent context
			parentCancel()

			// Signal context should be cancelled and cleaned up
			select {
			case <-sigCtx.Done():
				Expect(sigCtx.Err()).To(Equal(context.Canceled))
			case <-time.After(100 * time.Millisecond):
				Fail("Context should have been cancelled")
			}
		})
	})

	Context("Edge cases", func() {
		It("handles already cancelled parent context", func() {
			parentCtx, parentCancel := context.WithCancel(context.Background())
			parentCancel() // Cancel immediately

			sigCtx := run.ContextWithSig(parentCtx)

			// Signal context should be cancelled immediately
			select {
			case <-sigCtx.Done():
				Expect(sigCtx.Err()).To(Equal(context.Canceled))
			case <-time.After(100 * time.Millisecond):
				Fail("Context should have been cancelled immediately")
			}
		})

		It("handles context with deadline", func() {
			deadlineCtx, deadlineCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer deadlineCancel()

			sigCtx := run.ContextWithSig(deadlineCtx)

			// Context should be cancelled due to deadline
			select {
			case <-sigCtx.Done():
				// Should be cancelled due to deadline
				Expect(sigCtx.Err()).To(Equal(context.DeadlineExceeded))
			case <-time.After(200 * time.Millisecond):
				Fail("Context was not cancelled within timeout")
			}
		})
	})
})
