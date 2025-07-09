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

	Context("SIGINT signal handling", func() {
		It("cancels context when SIGINT is received", func() {
			Skip("Signal tests are problematic in test environment")
		})
	})

	Context("SIGTERM signal handling", func() {
		It("cancels context when SIGTERM is received", func() {
			Skip("Signal tests are problematic in test environment")
		})
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
	})

	Context("Multiple signal context creation", func() {
		It("handles multiple signal contexts independently", func() {
			Skip("Signal tests are problematic in test environment")
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
	})

	Context("Resource cleanup", func() {
		It("properly cleans up signal handlers", func() {
			Skip("Signal tests are problematic in test environment")
		})
	})

	Context("Concurrent signal handling", func() {
		It("handles concurrent signal context creation and cancellation", func() {
			Skip("Signal tests are problematic in test environment")
		})
	})

	Context("Signal handling race conditions", func() {
		It("handles race between signal and parent context cancellation", func() {
			Skip("Signal tests are problematic in test environment")
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
