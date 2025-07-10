// Copyright (c) 2019 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package run_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/run"
)

type CustomError struct {
	Code    int
	Message string
}

func (e CustomError) Error() string {
	return fmt.Sprintf("custom error %d: %s", e.Code, e.Message)
}

var _ = Describe("Complex Error Propagation", func() {
	var ctx context.Context
	var cancel context.CancelFunc

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
	})

	AfterEach(func() {
		cancel()
	})

	Context("Nested error scenarios", func() {
		It("handles multiple nested errors properly", func() {
			nestedError := errors.New("deep nested error")
			middleError := fmt.Errorf("middle error: %w", nestedError)
			topError := fmt.Errorf("top error: %w", middleError)

			fn := run.Func(func(ctx context.Context) error {
				return topError
			})

			err := fn.Run(ctx)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("top error"))
			Expect(err.Error()).To(ContainSubstring("middle error"))
			Expect(err.Error()).To(ContainSubstring("deep nested error"))
		})

		It("handles error chains in All execution", func() {
			error1 := errors.New("error 1")
			error2 := errors.New("error 2")
			error3 := errors.New("error 3")

			err := run.All(ctx,
				func(ctx context.Context) error {
					return error1
				},
				func(ctx context.Context) error {
					return error2
				},
				func(ctx context.Context) error {
					return error3
				},
			)

			Expect(err).To(HaveOccurred())
			errStr := err.Error()
			Expect(errStr).To(ContainSubstring("error 1"))
			Expect(errStr).To(ContainSubstring("error 2"))
			Expect(errStr).To(ContainSubstring("error 3"))
		})

		It("handles mixed success and errors with error aggregation", func() {
			const numFuncs = 20
			var successCount int64
			var errorCount int64

			funcs := make([]run.Func, numFuncs)
			for i := 0; i < numFuncs; i++ {
				index := i
				funcs[i] = func(ctx context.Context) error {
					if index%3 == 0 {
						atomic.AddInt64(&errorCount, 1)
						return fmt.Errorf("error from function %d", index)
					} else {
						atomic.AddInt64(&successCount, 1)
						return nil
					}
				}
			}

			err := run.All(ctx, funcs...)
			Expect(err).To(HaveOccurred())

			// Count errors in the aggregated error
			errStr := err.Error()
			errorLines := strings.Split(errStr, "\n")
			actualErrorCount := int64(0)
			for _, line := range errorLines {
				if strings.Contains(line, "error from function") {
					actualErrorCount++
				}
			}

			Expect(actualErrorCount).To(Equal(atomic.LoadInt64(&errorCount)))
			Expect(atomic.LoadInt64(&successCount)).To(BeNumerically(">", 0))
		})
	})

	Context("Error propagation with context cancellation", func() {
		It("properly handles errors when context is cancelled", func() {
			var functionsStarted int64
			var functionsCompleted int64
			var errorsReturned int64

			funcs := make([]run.Func, 10)
			for i := 0; i < 10; i++ {
				index := i
				funcs[i] = func(ctx context.Context) error {
					atomic.AddInt64(&functionsStarted, 1)
					defer atomic.AddInt64(&functionsCompleted, 1)

					select {
					case <-ctx.Done():
						atomic.AddInt64(&errorsReturned, 1)
						return ctx.Err()
					case <-time.After(100 * time.Millisecond):
						if index%2 == 0 {
							atomic.AddInt64(&errorsReturned, 1)
							return fmt.Errorf("error from function %d", index)
						}
						return nil
					}
				}
			}

			// Cancel context after short delay
			go func() {
				time.Sleep(20 * time.Millisecond)
				cancel()
			}()

			err := run.All(ctx, funcs...)
			Expect(err).To(HaveOccurred())

			// Should have some functions started
			Expect(atomic.LoadInt64(&functionsStarted)).To(BeNumerically(">", 0))
			// Should have some errors returned
			Expect(atomic.LoadInt64(&errorsReturned)).To(BeNumerically(">", 0))
		})

		It("handles context timeout with mixed errors", func() {
			timeoutCtx, timeoutCancel := context.WithTimeout(ctx, 50*time.Millisecond)
			defer timeoutCancel()

			var timeoutErrors int64
			var otherErrors int64

			err := run.All(timeoutCtx,
				func(ctx context.Context) error {
					<-ctx.Done()
					if ctx.Err() == context.DeadlineExceeded {
						atomic.AddInt64(&timeoutErrors, 1)
					}
					return ctx.Err()
				},
				func(ctx context.Context) error {
					atomic.AddInt64(&otherErrors, 1)
					return errors.New("custom error")
				},
				func(ctx context.Context) error {
					time.Sleep(100 * time.Millisecond) // Should timeout
					return nil
				},
			)

			Expect(err).To(HaveOccurred())
			Expect(atomic.LoadInt64(&timeoutErrors)).To(BeNumerically(">=", 1))
			Expect(atomic.LoadInt64(&otherErrors)).To(Equal(int64(1)))
		})
	})

	Context("Error propagation in different execution strategies", func() {
		XIt("handles errors in CancelOnFirstError strategy", func() {
			var executedFuncs int64

			err := run.CancelOnFirstError(ctx,
				func(ctx context.Context) error {
					atomic.AddInt64(&executedFuncs, 1)
					time.Sleep(10 * time.Millisecond)
					return errors.New("first error")
				},
				func(ctx context.Context) error {
					atomic.AddInt64(&executedFuncs, 1)
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(100 * time.Millisecond):
						return nil
					}
				},
				func(ctx context.Context) error {
					atomic.AddInt64(&executedFuncs, 1)
					return errors.New("should not execute")
				},
			)

			Expect(err).To(HaveOccurred())
			// CancelOnFirstError returns the first error encountered
			Expect(err.Error()).To(Or(ContainSubstring("first error"), ContainSubstring("should not execute")))

			// Should have started at least the first function
			Expect(atomic.LoadInt64(&executedFuncs)).To(BeNumerically(">=", 1))
		})

		It("handles errors in CancelOnFirstErrorWait strategy", func() {
			var executedFuncs int64
			var completedFuncs int64

			err := run.CancelOnFirstErrorWait(ctx,
				func(ctx context.Context) error {
					atomic.AddInt64(&executedFuncs, 1)
					defer atomic.AddInt64(&completedFuncs, 1)
					return errors.New("error 1")
				},
				func(ctx context.Context) error {
					atomic.AddInt64(&executedFuncs, 1)
					defer atomic.AddInt64(&completedFuncs, 1)
					time.Sleep(20 * time.Millisecond)
					return errors.New("error 2")
				},
				func(ctx context.Context) error {
					atomic.AddInt64(&executedFuncs, 1)
					defer atomic.AddInt64(&completedFuncs, 1)
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(100 * time.Millisecond):
						return nil
					}
				},
			)

			Expect(err).To(HaveOccurred())
			errStr := err.Error()
			Expect(errStr).To(ContainSubstring("error 1"))
			// May or may not contain error 2 depending on timing

			Expect(atomic.LoadInt64(&executedFuncs)).To(BeNumerically(">=", 1))
		})

		It("handles errors in CancelOnFirstFinishWait strategy", func() {
			var executedFuncs int64

			err := run.CancelOnFirstFinishWait(ctx,
				func(ctx context.Context) error {
					atomic.AddInt64(&executedFuncs, 1)
					return errors.New("quick error")
				},
				func(ctx context.Context) error {
					atomic.AddInt64(&executedFuncs, 1)
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(100 * time.Millisecond):
						return nil
					}
				},
				func(ctx context.Context) error {
					atomic.AddInt64(&executedFuncs, 1)
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(100 * time.Millisecond):
						return errors.New("slow error")
					}
				},
			)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("quick error"))

			Expect(atomic.LoadInt64(&executedFuncs)).To(BeNumerically(">=", 1))
		})

		It("handles sequential error propagation", func() {
			var executedFuncs int64
			var funcOrder []int
			var mutex sync.Mutex

			err := run.Sequential(ctx,
				func(ctx context.Context) error {
					mutex.Lock()
					funcOrder = append(funcOrder, 1)
					mutex.Unlock()
					atomic.AddInt64(&executedFuncs, 1)
					return nil
				},
				func(ctx context.Context) error {
					mutex.Lock()
					funcOrder = append(funcOrder, 2)
					mutex.Unlock()
					atomic.AddInt64(&executedFuncs, 1)
					return errors.New("sequential error")
				},
				func(ctx context.Context) error {
					mutex.Lock()
					funcOrder = append(funcOrder, 3)
					mutex.Unlock()
					atomic.AddInt64(&executedFuncs, 1)
					return nil
				},
			)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("sequential error"))

			// Should have executed first two functions in order
			Expect(atomic.LoadInt64(&executedFuncs)).To(Equal(int64(2)))
			mutex.Lock()
			Expect(funcOrder).To(Equal([]int{1, 2}))
			mutex.Unlock()
		})
	})

	Context("Error propagation with ConcurrentRunner", func() {
		XIt("handles errors in concurrent runner execution", func() {
			runner := run.NewConcurrentRunner(3)

			var executedFuncs int64
			var erroredFuncs int64

			// Add functions that will error
			runner.Add(ctx, func(ctx context.Context) error {
				atomic.AddInt64(&executedFuncs, 1)
				atomic.AddInt64(&erroredFuncs, 1)
				return errors.New("concurrent error 1")
			})

			runner.Add(ctx, func(ctx context.Context) error {
				atomic.AddInt64(&executedFuncs, 1)
				return nil
			})

			runner.Add(ctx, func(ctx context.Context) error {
				atomic.AddInt64(&executedFuncs, 1)
				atomic.AddInt64(&erroredFuncs, 1)
				return errors.New("concurrent error 2")
			})

			runner.Close()
			err := runner.Run(ctx)

			Expect(err).To(HaveOccurred())
			// Should return first error encountered
			Expect(err.Error()).To(ContainSubstring("concurrent error"))

			// Should have attempted to execute some functions
			Expect(atomic.LoadInt64(&executedFuncs)).To(BeNumerically(">=", 1))
			Expect(atomic.LoadInt64(&erroredFuncs)).To(BeNumerically(">=", 1))
		})

		XIt("handles context cancellation in concurrent runner", func() {
			localCtx, localCancel := context.WithCancel(ctx)
			runner := run.NewConcurrentRunner(2)

			var startedFuncs int64
			var cancelledFuncs int64

			for i := 0; i < 5; i++ {
				runner.Add(localCtx, func(ctx context.Context) error {
					atomic.AddInt64(&startedFuncs, 1)
					select {
					case <-ctx.Done():
						atomic.AddInt64(&cancelledFuncs, 1)
						return ctx.Err()
					case <-time.After(100 * time.Millisecond):
						return nil
					}
				})
			}

			// Cancel context after short delay
			go func() {
				time.Sleep(10 * time.Millisecond)
				localCancel()
			}()

			runner.Close()
			err := runner.Run(localCtx)

			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, context.Canceled)).To(BeTrue())

			Expect(atomic.LoadInt64(&startedFuncs)).To(BeNumerically(">=", 1))
			Expect(atomic.LoadInt64(&cancelledFuncs)).To(BeNumerically(">=", 1))
		})
	})

	Context("Complex error wrapping scenarios", func() {
		It("handles deeply nested error contexts", func() {
			var createNestedError func(depth int) error
			createNestedError = func(depth int) error {
				if depth == 0 {
					return errors.New("base error")
				}
				return fmt.Errorf("level %d: %w", depth, createNestedError(depth-1))
			}

			err := run.All(ctx,
				func(ctx context.Context) error {
					return createNestedError(5)
				},
				func(ctx context.Context) error {
					return createNestedError(3)
				},
			)

			Expect(err).To(HaveOccurred())
			errStr := err.Error()
			Expect(errStr).To(ContainSubstring("level 5"))
			Expect(errStr).To(ContainSubstring("level 3"))
			Expect(errStr).To(ContainSubstring("base error"))
		})

		It("handles error type preservation", func() {
			customErr := CustomError{Code: 404, Message: "not found"}

			err := run.All(ctx,
				func(ctx context.Context) error {
					return customErr
				},
				func(ctx context.Context) error {
					return fmt.Errorf("wrapped: %w", customErr)
				},
			)

			Expect(err).To(HaveOccurred())
			errStr := err.Error()
			Expect(errStr).To(ContainSubstring("custom error 404"))
			Expect(errStr).To(ContainSubstring("not found"))
		})
	})
})
