// Copyright (c) 2020 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package run_test

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/bborbe/run"
)

var _ = Describe("Metrics", func() {
	var err error
	var callCounter int
	var innerResult error
	var innerFn func(ctx context.Context) error
	var ctx context.Context
	var registerer prometheus.Registerer
	var fn run.Func
	BeforeEach(func() {
		ctx = context.Background()
		innerResult = nil
		callCounter = 0
		innerFn = func(ctx context.Context) error {
			callCounter++
			return innerResult
		}
		registerer = prometheus.NewRegistry()
		fn = run.NewMetrics(registerer, "ns", "sub", innerFn)
	})
	Context("no error", func() {
		BeforeEach(func() {
			err = fn(ctx)
		})
		It("calls inner func", func() {
			Expect(callCounter).To(Equal(1))
		})
		It("returns no error", func() {
			Expect(err).To(BeNil())
		})
	})
	Context("error", func() {
		BeforeEach(func() {
			innerResult = errors.New("banana")
			err = fn(ctx)
		})
		It("calls inner func", func() {
			Expect(callCounter).To(Equal(1))
		})
		It("returns error", func() {
			Expect(err).NotTo(BeNil())
		})
	})
})
