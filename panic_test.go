// Copyright (c) 2021 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package run_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/run"
)

var _ = Describe("CatchPanic", func() {
	var fn run.Func
	var ctx context.Context
	var err error
	BeforeEach(func() {
		ctx = context.Background()
		fn = run.CatchPanic(func(ctx context.Context) error {
			panic("banana")
		})
		err = fn(ctx)
	})
	It("returns error", func() {
		Expect(err).NotTo(BeNil())
	})
	It("contains correct error message", func() {
		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(Equal("catch panic: banana"))
	})
})
