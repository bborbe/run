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
	//Context("run only max at a time", func() {
	//	BeforeEach(func() {
	//		for i := 0; i < 20; i++ {
	//			go func() {
	//				concurrentRunner.Add(ctx, func(ctx context.Context) error {
	//					mux.Lock()
	//					atomic.AddUint64(&counter, 1)
	//					mux.Unlock()
	//					select {
	//					case <-ctx.Done():
	//					}
	//					return nil
	//				})
	//			}()
	//		}
	//		time.Sleep(100 * time.Millisecond)
	//		concurrentRunner.Close()
	//	})
	//	It("executes 8 methods", func() {
	//		mux.Lock()
	//		Expect(atomic.LoadUint64(&counter)).To(Equal(uint64(8)))
	//		mux.Unlock()
	//	})
	//})
})
