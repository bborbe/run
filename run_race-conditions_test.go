// Copyright (c) 2019 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package run_test

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/run"
)

var _ = Describe("Race Conditions and Resource Cleanup", func() {
	var ctx context.Context
	var cancel context.CancelFunc

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
	})

	AfterEach(func() {
		cancel()
	})

	Context("Race condition tests", func() {
		It("handles concurrent function execution without race conditions", func() {
			const numGoroutines = 100
			var counter int64
			var wg sync.WaitGroup

			funcs := make([]run.Func, numGoroutines)
			for i := 0; i < numGoroutines; i++ {
				funcs[i] = func(ctx context.Context) error {
					atomic.AddInt64(&counter, 1)
					return nil
				}
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				err := run.All(ctx, funcs...)
				Expect(err).To(BeNil())
			}()

			wg.Wait()
			Expect(atomic.LoadInt64(&counter)).To(Equal(int64(numGoroutines)))
		})

		It("handles race between context cancellation and function execution", func() {
			const numGoroutines = 50
			var startedFuncs int64
			var cancelledFuncs int64

			// Create functions that detect cancellation
			funcs := make([]run.Func, numGoroutines)
			for i := 0; i < numGoroutines; i++ {
				funcs[i] = func(ctx context.Context) error {
					atomic.AddInt64(&startedFuncs, 1)

					// Simulate some work
					select {
					case <-ctx.Done():
						atomic.AddInt64(&cancelledFuncs, 1)
						return ctx.Err()
					case <-time.After(10 * time.Millisecond):
						return nil
					}
				}
			}

			// Cancel context after a short delay
			go func() {
				time.Sleep(5 * time.Millisecond)
				cancel()
			}()

			err := run.All(ctx, funcs...)
			Expect(err).To(HaveOccurred())

			// Some functions should have started
			Expect(atomic.LoadInt64(&startedFuncs)).To(BeNumerically(">", 0))
			// Some functions should have been cancelled
			Expect(atomic.LoadInt64(&cancelledFuncs)).To(BeNumerically(">", 0))
		})

		It("handles concurrent ConcurrentRunner operations", func() {
			const numRunners = 10
			const numFuncsPerRunner = 20
			var totalExecuted int64
			var wg sync.WaitGroup

			// Create multiple concurrent runners
			runners := make([]run.ConcurrentRunner, numRunners)
			for i := 0; i < numRunners; i++ {
				runners[i] = run.NewConcurrentRunner(5)
			}

			// Add functions to runners concurrently
			for i := 0; i < numRunners; i++ {
				wg.Add(1)
				go func(runner run.ConcurrentRunner) {
					defer wg.Done()
					for j := 0; j < numFuncsPerRunner; j++ {
						runner.Add(ctx, func(ctx context.Context) error {
							atomic.AddInt64(&totalExecuted, 1)
							return nil
						})
					}
					runner.Close()
				}(runners[i])
			}

			// Run all runners concurrently
			for i := 0; i < numRunners; i++ {
				wg.Add(1)
				go func(runner run.ConcurrentRunner) {
					defer wg.Done()
					err := runner.Run(ctx)
					Expect(err).To(BeNil())
				}(runners[i])
			}

			wg.Wait()
			Expect(
				atomic.LoadInt64(&totalExecuted),
			).To(Equal(int64(numRunners * numFuncsPerRunner)))
		})

		It("handles race between multiple signal context creations", func() {
			const numContexts = 50
			var wg sync.WaitGroup
			contexts := make([]context.Context, numContexts)

			// Create multiple signal contexts concurrently
			for i := 0; i < numContexts; i++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()
					parentCtx, parentCancel := context.WithCancel(context.Background())
					defer parentCancel()

					contexts[idx] = run.ContextWithSig(parentCtx)

					// Simulate some work
					time.Sleep(time.Duration(idx) * time.Millisecond)
					parentCancel()
				}(i)
			}

			wg.Wait()

			// All contexts should be valid
			for i := 0; i < numContexts; i++ {
				Expect(contexts[i]).NotTo(BeNil())
			}
		})
	})

	Context("Resource cleanup tests", func() {
		It("properly cleans up resources in ConcurrentRunner", func() {
			runner := run.NewConcurrentRunner(5)

			// Add some functions
			runner.Add(ctx, func(ctx context.Context) error {
				time.Sleep(10 * time.Millisecond)
				return nil
			})

			// Close the runner
			err := runner.Close()
			Expect(err).To(BeNil())

			// Closing again should return an error
			err = runner.Close()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("already closed"))
		})

		It("handles resource cleanup with context cancellation", func() {
			const numFuncs = 20
			var resourcesOpened int64
			var resourcesClosed int64
			var wg sync.WaitGroup

			funcs := make([]run.Func, numFuncs)
			for i := 0; i < numFuncs; i++ {
				funcs[i] = func(ctx context.Context) error {
					atomic.AddInt64(&resourcesOpened, 1)
					defer atomic.AddInt64(&resourcesClosed, 1)

					// Simulate resource usage
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(100 * time.Millisecond):
						return nil
					}
				}
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				// Cancel context after short delay
				time.Sleep(10 * time.Millisecond)
				cancel()
			}()

			err := run.All(ctx, funcs...)
			Expect(err).To(HaveOccurred())

			// Wait for all goroutines to complete
			wg.Wait()
			runtime.GC()
			time.Sleep(50 * time.Millisecond)

			// All opened resources should be closed
			opened := atomic.LoadInt64(&resourcesOpened)
			closed := atomic.LoadInt64(&resourcesClosed)
			Expect(opened).To(Equal(closed))
		})

		It("handles memory cleanup with large number of functions", func() {
			const numFuncs = 1000
			var memStats1, memStats2 runtime.MemStats

			// Get initial memory stats
			runtime.GC()
			runtime.ReadMemStats(&memStats1)

			// Create and execute many functions
			funcs := make([]run.Func, numFuncs)
			for i := 0; i < numFuncs; i++ {
				funcs[i] = func(ctx context.Context) error {
					// Allocate some memory
					data := make([]byte, 1024)
					_ = data
					return nil
				}
			}

			err := run.All(ctx, funcs...)
			Expect(err).To(BeNil())

			// Force garbage collection
			runtime.GC()
			runtime.ReadMemStats(&memStats2)

			// Memory usage should not grow excessively
			// Allow some reasonable growth but not proportional to numFuncs
			memoryGrowth := memStats2.Alloc - memStats1.Alloc
			Expect(
				memoryGrowth,
			).To(BeNumerically("<", numFuncs*1024))
			// Should be less than total allocated
		})

		It("handles goroutine cleanup properly", func() {
			initialGoroutines := runtime.NumGoroutine()

			// Create many concurrent operations
			const numOperations = 100
			var wg sync.WaitGroup

			for i := 0; i < numOperations; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()

					localCtx, localCancel := context.WithCancel(ctx)
					defer localCancel()

					// Create concurrent runner
					runner := run.NewConcurrentRunner(3)

					// Add some functions
					runner.Add(localCtx, func(ctx context.Context) error {
						return nil
					})

					_ = runner.Close()
					_ = runner.Run(localCtx)
				}()
			}

			wg.Wait()

			// Wait for potential goroutine cleanup
			time.Sleep(100 * time.Millisecond)
			runtime.GC()

			finalGoroutines := runtime.NumGoroutine()

			// Should not have significant goroutine leak
			// Allow some variance but not proportional to numOperations
			goroutineGrowth := finalGoroutines - initialGoroutines
			Expect(goroutineGrowth).To(BeNumerically("<", numOperations/2))
		})

	})

	Context("Complex concurrent scenarios", func() {
		It("handles mixed success and failure with resource cleanup", func() {
			const numFuncs = 50
			var successCount int64
			var errorCount int64
			var cleanupCount int64

			funcs := make([]run.Func, numFuncs)
			for i := 0; i < numFuncs; i++ {
				index := i // Capture loop variable
				funcs[i] = func(ctx context.Context) error {
					defer atomic.AddInt64(&cleanupCount, 1)

					// Some functions succeed, some fail
					if index%3 == 0 {
						atomic.AddInt64(&errorCount, 1)
						return errors.New("test error")
					} else {
						atomic.AddInt64(&successCount, 1)
						return nil
					}
				}
			}

			err := run.All(ctx, funcs...)
			Expect(err).To(HaveOccurred()) // Should have errors

			// All functions should have been called
			Expect(atomic.LoadInt64(&successCount)).To(BeNumerically(">", 0))
			Expect(atomic.LoadInt64(&errorCount)).To(BeNumerically(">", 0))
			Expect(atomic.LoadInt64(&cleanupCount)).To(Equal(int64(numFuncs)))
		})

		It("handles concurrent access to shared resources", func() {
			const numReaders = 25
			const numWriters = 25
			var rwMutex sync.RWMutex
			var sharedValue int64
			var readCount int64
			var writeCount int64

			// Create reader functions
			readerFuncs := make([]run.Func, numReaders)
			for i := 0; i < numReaders; i++ {
				readerFuncs[i] = func(ctx context.Context) error {
					rwMutex.RLock()
					defer rwMutex.RUnlock()

					_ = atomic.LoadInt64(&sharedValue)
					atomic.AddInt64(&readCount, 1)
					return nil
				}
			}

			// Create writer functions
			writerFuncs := make([]run.Func, numWriters)
			for i := 0; i < numWriters; i++ {
				writerFuncs[i] = func(ctx context.Context) error {
					rwMutex.Lock()
					defer rwMutex.Unlock()

					atomic.AddInt64(&sharedValue, 1)
					atomic.AddInt64(&writeCount, 1)
					return nil
				}
			}

			// Combine all functions
			allFuncs := append(readerFuncs, writerFuncs...)

			err := run.All(ctx, allFuncs...)
			Expect(err).To(BeNil())

			Expect(atomic.LoadInt64(&readCount)).To(Equal(int64(numReaders)))
			Expect(atomic.LoadInt64(&writeCount)).To(Equal(int64(numWriters)))
			Expect(atomic.LoadInt64(&sharedValue)).To(Equal(int64(numWriters)))
		})
	})
})
