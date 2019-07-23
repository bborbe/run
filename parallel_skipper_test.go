// Copyright (c) 2019 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package run_test

import (
	"context"
	"time"

	"github.com/bborbe/run"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ParallelSkipper", func() {
	It("runs", func() {
		p := run.NewParallelSkipper()
		counter := 0
		p.SkipParallel(func(ctx context.Context) error {
			counter++
			return nil
		})(context.Background())
		Expect(counter).To(Equal(1))
	})
	It("skip if already running", func() {
		p := run.NewParallelSkipper()
		p.SkipParallel(func(ctx context.Context) error {
			time.Sleep(time.Second)
			return nil
		})(context.Background())
		counter := 0
		p.SkipParallel(func(ctx context.Context) error {
			counter++
			return nil
		})(context.Background())
		Expect(counter).To(Equal(0))
	})
})
