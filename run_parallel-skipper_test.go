// Copyright (c) 2019 Benjamin Borbe All rights reserved.
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
})
