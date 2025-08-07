// Copyright (c) 2021 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package run_test

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bborbe/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/run"
)

var _ = Describe("ConcurrentRunner", func() {
	var ctx context.Context
	var max int
	var concurrentRunner run.ConcurrentRunner
	var counter uint64
	var mux sync.Mutex
	var err error
	var wg sync.WaitGroup
	BeforeEach(func() {
		ctx = context.Background()
		atomic.StoreUint64(&counter, 0)
		max = 8
		concurrentRunner = run.NewConcurrentRunner(max)
	})
	JustBeforeEach(func() {
		ctx, cancel := context.WithCancel(ctx)
		errs := make(chan error, runtime.NumCPU())
		defer close(errs)
		wg.Add(1)
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
			case <-time.NewTimer(time.Second).C:
				cancel()
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- concurrentRunner.Run(ctx)
		}()
		err = <-errs
		wg.Wait()
	})
	Context("returns on context cancel", func() {
		BeforeEach(func() {
			_, cancel := context.WithCancel(ctx)
			wg.Add(1)
			go func() {
				defer wg.Done()
				cancel()
			}()
		})
		It("returns context canceled error", func() {
			Expect(errors.Cause(err)).To(Equal(context.Canceled))
		})
	})
	Context("executes funcs", func() {
		BeforeEach(func() {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < 10; i++ {
					concurrentRunner.Add(ctx, func(ctx context.Context) error {
						mux.Lock()
						atomic.AddUint64(&counter, 1)
						mux.Unlock()
						return nil
					})
				}
				concurrentRunner.Close()
			}()
		})
		It("returns no error", func() {
			Expect(err).To(BeNil())
		})
		It("executes 10 methods", func() {
			mux.Lock()
			Expect(atomic.LoadUint64(&counter)).To(Equal(uint64(10)))
			mux.Unlock()
		})
	})
	Context("run only max at a time", func() {
		var concurrentChecker *int32
		var maxConcurrent int32
		BeforeEach(func() {
			var currentConcurrent int32
			concurrentChecker = &currentConcurrent
			maxConcurrent = 0
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 0; i < 20; i++ {
					concurrentRunner.Add(ctx, func(ctx context.Context) error {
						current := atomic.AddInt32(concurrentChecker, 1)
						for {
							old := atomic.LoadInt32(&maxConcurrent)
							if current <= old ||
								atomic.CompareAndSwapInt32(&maxConcurrent, old, current) {
								break
							}
						}
						// Simulate work
						time.Sleep(10 * time.Millisecond)
						atomic.AddInt32(concurrentChecker, -1)
						atomic.AddUint64(&counter, 1)
						return nil
					})
				}
				concurrentRunner.Close()
			}()
		})
		It("never exceeds max concurrent limit", func() {
			Expect(atomic.LoadInt32(&maxConcurrent)).To(BeNumerically("<=", int32(max)))
		})
		It("executes all methods", func() {
			Expect(atomic.LoadUint64(&counter)).To(Equal(uint64(20)))
		})
	})
	Context("Close() edge cases", func() {
		It("returns error when closed multiple times", func() {
			runner := run.NewConcurrentRunner(5)
			err1 := runner.Close()
			Expect(err1).To(BeNil())

			err2 := runner.Close()
			Expect(err2).To(HaveOccurred())
			Expect(err2.Error()).To(ContainSubstring("already closed"))
		})
	})
	Context("Add() edge cases", func() {
		It("discards functions added after close", func() {
			var localCounter uint64
			runner := run.NewConcurrentRunner(5)
			runner.Close()

			// This should not panic and should discard the function
			runner.Add(ctx, func(ctx context.Context) error {
				atomic.AddUint64(&localCounter, 1)
				return nil
			})

			// Run should complete without executing the discarded function
			err := runner.Run(ctx)
			Expect(err).To(BeNil())
			Expect(atomic.LoadUint64(&localCounter)).To(Equal(uint64(0)))
		})
	})
})
