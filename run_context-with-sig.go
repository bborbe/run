// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package run

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/golang/glog"
)

func ContextWithSig(ctx context.Context) context.Context {
	ctxWithCancel, cancel := context.WithCancel(ctx)
	go func() {
		defer cancel()

		signalCh := make(chan os.Signal, 1)
		defer close(signalCh)

		signal.Notify(signalCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

		select {
		case signal, ok := <-signalCh:
			if !ok {
				glog.V(2).Infof("signal channel closed => cancel context ")
				return
			}
			glog.V(2).Infof("got signal %s => cancel context ", signal)
		case <-ctx.Done():
		}
	}()

	return ctxWithCancel
}
