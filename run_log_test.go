// Copyright (c) 2021 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package run_test

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/run"
)

var _ = Describe("LogErrors", func() {
	var (
		ctx context.Context
	)
	BeforeEach(func() {
		ctx = context.Background()
	})

	It("returns no error if wrapped func returns nil", func() {
		fn := run.LogErrors(func(ctx context.Context) error {
			return nil
		})
		err := fn(ctx)
		Expect(err).To(BeNil())
	})

	It("returns error if wrapped func returns error", func() {
		expectedErr := errors.New("banana")
		fn := run.LogErrors(func(ctx context.Context) error {
			return expectedErr
		})
		err := fn(ctx)
		Expect(err).To(Equal(expectedErr))
	})
})
