// Copyright (c) 2021 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package run_test

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/bborbe/run"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Trigger", func() {
	var trigger run.Trigger
	BeforeEach(func() {
		trigger = run.NewTrigger()
	})
	It("returns something if fire was called before", func() {
		trigger.Fire()
		select {
		case <-trigger.Done():
		default:
			Fail("should never happen")
		}
	})
	It("triggers only once", func() {
		trigger.Fire()
		trigger.Fire()
		counter := 0
		for {
			_, ok := <-trigger.Done()
			counter++
			if !ok {
				break
			}
		}
		Expect(counter).To(Equal(1))
	})
	It("returns something if fire was called before", func() {
		go func() {
			<-time.NewTimer(100 * time.Millisecond).C
			trigger.Fire()
		}()
		select {
		case <-trigger.Done():
		case <-time.NewTimer(200 * time.Millisecond).C:
			Fail("should never happen")
		}
	})
	It("returns nothing if fire never was called", func() {
		select {
		case <-trigger.Done():
			Fail("should never happen")
		case <-time.NewTimer(100 * time.Millisecond).C:
		}
	})
	It("trigger multi done", func() {
		go func() {
			<-time.NewTimer(100 * time.Millisecond).C
			trigger.Fire()
		}()
		var counter int64
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				select {
				case <-trigger.Done():
					atomic.AddInt64(&counter, 1)
				case <-time.NewTimer(200 * time.Millisecond).C:
				}
			}()
		}
		wg.Wait()
		Expect(counter).To(Equal(int64(10)))
	})
})
