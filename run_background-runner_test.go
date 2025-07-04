// Copyright (c) 2023 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package run_test

import (
	"context"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/run"
)

var _ = Describe("BackgroundRunner", func() {
	var (
		ctx    context.Context
		cancel context.CancelFunc
	)
	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
	})
	AfterEach(func() {
		cancel()
	})

	It("should run the function in the background", func() {
		br := run.NewBackgroundRunner(ctx)
		var wg sync.WaitGroup
		wg.Add(1)
		called := false
		err := br.Run(func(ctx context.Context) error {
			defer wg.Done()
			called = true
			return nil
		})
		Expect(err).To(BeNil())
		// Wait for the background goroutine to finish
		wg.Wait()
		Expect(called).To(BeTrue())
	})

	It("should propagate error from the function (via logs, but Run always returns nil)", func() {
		br := run.NewBackgroundRunner(ctx)
		var wg sync.WaitGroup
		wg.Add(1)
		called := false
		err := br.Run(func(ctx context.Context) error {
			defer wg.Done()
			called = true
			return context.Canceled
		})
		Expect(err).To(BeNil())
		wg.Wait()
		Expect(called).To(BeTrue())
	})

	It("should not block on Run", func() {
		br := run.NewBackgroundRunner(ctx)
		start := time.Now()
		err := br.Run(func(ctx context.Context) error {
			time.Sleep(100 * time.Millisecond)
			return nil
		})
		Expect(err).To(BeNil())
		// Should return quickly (well before the function completes)
		Expect(time.Since(start)).To(BeNumerically("<", 50*time.Millisecond))
	})
})
