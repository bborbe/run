// Copyright (c) 2019 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package run_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestRun(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Run Suite")
}
