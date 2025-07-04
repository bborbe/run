// Copyright (c) 2024 Benjamin Borbe All rights reserved.
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

var _ = Describe("Waiter", func() {
	var (
		ctx context.Context
	)
	BeforeEach(func() {
		ctx = context.Background()
	})

	It("WaiterFunc waits for the specified duration", func() {
		waiter := run.NewWaiter()
		start := time.Now()
		err := waiter.Wait(ctx, 50*time.Millisecond)
		Expect(err).To(BeNil())
		Expect(time.Since(start)).To(BeNumerically(">=", 50*time.Millisecond))
	})

	It("WaiterFunc returns context error if canceled", func() {
		waiter := run.NewWaiter()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := waiter.Wait(ctx, 100*time.Millisecond)
		Expect(err).To(Equal(context.Canceled))
	})
})
