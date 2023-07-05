// Copyright (c) 2021 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package run_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/run"
)

var _ = Describe("MultiTrigger", func() {
	var multiTrigger run.MultiTrigger
	var t1, t2, t3 run.Fire
	BeforeEach(func() {
		multiTrigger = run.NewMultiTrigger()
		t1 = multiTrigger.Add()
		t2 = multiTrigger.Add()
		t3 = multiTrigger.Add()
	})
	It("nothing done", func() {
		select {
		case <-multiTrigger.Done():
			Fail("should not be done")
		default:
			Succeed()
		}
	})
	It("one done", func() {
		t1.Fire()
		select {
		case <-multiTrigger.Done():
			Fail("should not be done")
		default:
			Succeed()
		}
	})
	It("two done", func() {
		t1.Fire()
		t2.Fire()
		select {
		case <-multiTrigger.Done():
			Fail("should not be done")
		default:
			Succeed()
		}
	})
	It("all done", func() {
		t1.Fire()
		t2.Fire()
		t3.Fire()
		select {
		case <-multiTrigger.Done():
			Succeed()
		default:
			Fail("should be done")
		}
	})
})
