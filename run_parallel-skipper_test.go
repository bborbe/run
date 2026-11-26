// Copyright (c) 2019 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package run_test

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/run"
)

type fakeCurrentDateTime struct {
	mux sync.Mutex
	now time.Time
}

func (f *fakeCurrentDateTime) Now() time.Time {
	f.mux.Lock()
	defer f.mux.Unlock()
	return f.now
}

func (f *fakeCurrentDateTime) SetNow(now time.Time) {
	f.mux.Lock()
	defer f.mux.Unlock()
	f.now = now
}

var _ = Describe("ParallelSkipper", func() {
	var ctx context.Context
	BeforeEach(func() {
		ctx = context.Background()
	})
	It("runs", func() {
		p := run.NewParallelSkipper()
		counter := 0
		fn := p.SkipParallel(func(ctx context.Context) error {
			counter++
			return nil
		})
		err := fn(ctx)
		Expect(err).To(BeNil())
		Expect(counter).To(Equal(1))
	})
	It("skips sampe func parallel", func() {
		p := run.NewParallelSkipper()
		counter := 0
		parallel := 4
		var wg sync.WaitGroup
		wg.Add(parallel)
		fn := p.SkipParallel(func(ctx context.Context) error {
			time.Sleep(100 * time.Millisecond)
			counter++
			return nil
		})
		for i := 0; i < parallel; i++ {
			go func() {
				defer wg.Done()
				err := fn(ctx)
				Expect(err).To(BeNil())
			}()
		}
		wg.Wait()
		Expect(counter).To(Equal(1))
	})
	It("releases a stale lock after MaxRunningDuration and runs the next invocation", func() {
		currentDateTime := &fakeCurrentDateTime{}
		start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		currentDateTime.SetNow(start)
		p := run.NewParallelSkipper(
			run.WithCurrentDateTime(currentDateTime),
			run.WithMaxRunningDuration(1*time.Hour),
		)
		var counter atomic.Int32
		release := make(chan struct{})
		started := make(chan struct{}, 2)
		fn := p.SkipParallel(func(ctx context.Context) error {
			counter.Add(1)
			started <- struct{}{}
			<-release
			return nil
		})
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer GinkgoRecover()
			defer wg.Done()
			Expect(fn(ctx)).To(BeNil())
		}()
		Eventually(started).Should(Receive())

		currentDateTime.SetNow(start.Add(30 * time.Minute))
		Expect(fn(ctx)).To(BeNil())
		Expect(counter.Load()).To(Equal(int32(1)))

		currentDateTime.SetNow(start.Add(2 * time.Hour))
		go func() {
			defer GinkgoRecover()
			defer wg.Done()
			Expect(fn(ctx)).To(BeNil())
		}()
		Eventually(counter.Load).Should(Equal(int32(2)))

		close(release)
		wg.Wait()
	})
	It("never releases the lock when MaxRunningDuration is not set", func() {
		p := run.NewParallelSkipper()
		var counter atomic.Int32
		release := make(chan struct{})
		started := make(chan struct{}, 2)
		fn := p.SkipParallel(func(ctx context.Context) error {
			counter.Add(1)
			started <- struct{}{}
			<-release
			return nil
		})
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer GinkgoRecover()
			defer wg.Done()
			Expect(fn(ctx)).To(BeNil())
		}()
		Eventually(started).Should(Receive())

		Expect(fn(ctx)).To(BeNil())
		Expect(counter.Load()).To(Equal(int32(1)))

		close(release)
		wg.Wait()
	})
	It("keeps the force-released lock held when the hung invocation returns", func() {
		currentDateTime := &fakeCurrentDateTime{}
		start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		currentDateTime.SetNow(start)
		p := run.NewParallelSkipper(
			run.WithCurrentDateTime(currentDateTime),
			run.WithMaxRunningDuration(1*time.Hour),
		)
		var counter atomic.Int32
		releaseA := make(chan struct{})
		releaseB := make(chan struct{})
		startedA := make(chan struct{}, 1)
		startedB := make(chan struct{}, 1)
		doneA := make(chan struct{})
		doneB := make(chan struct{})
		fnA := p.SkipParallel(func(ctx context.Context) error {
			counter.Add(1)
			startedA <- struct{}{}
			<-releaseA
			return nil
		})
		fnB := p.SkipParallel(func(ctx context.Context) error {
			counter.Add(1)
			startedB <- struct{}{}
			<-releaseB
			return nil
		})
		fnC := p.SkipParallel(func(ctx context.Context) error {
			counter.Add(1)
			return nil
		})

		go func() {
			defer GinkgoRecover()
			defer close(doneA)
			Expect(fnA(ctx)).To(BeNil())
		}()
		Eventually(startedA).Should(Receive())

		currentDateTime.SetNow(start.Add(2 * time.Hour))
		go func() {
			defer GinkgoRecover()
			defer close(doneB)
			Expect(fnB(ctx)).To(BeNil())
		}()
		Eventually(startedB).Should(Receive())
		Expect(counter.Load()).To(Equal(int32(2)))

		close(releaseA)
		Eventually(doneA).Should(BeClosed())

		Expect(fnC(ctx)).To(BeNil())
		Expect(counter.Load()).To(Equal(int32(2)))

		close(releaseB)
		Eventually(doneB).Should(BeClosed())

		Expect(fnC(ctx)).To(BeNil())
		Expect(counter.Load()).To(Equal(int32(3)))
	})
})
