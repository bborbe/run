// Copyright (c) 2019 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package run

import (
	"context"
	"sync"
	"github.com/golang/glog"
)


// CancelOnFirstFinish executes all given functions. After the first function finishes, any remaining functions will be canceled.
func CancelOnFirstFinish(ctx context.Context, funcs ...Func) error {
	if len(funcs) == 0 {
		glog.V(2).Infof("nothing to run")
		return nil
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	result := make(chan error)
	defer close(result)
	var wg sync.WaitGroup
	for _, runner := range funcs {
		wg.Add(1)
		go func(run Func) {
			defer wg.Done()
			err := run(ctx)
			select {
			case result <- err:
			default:
			}
		}(runner)
	}
	var err error
	select {
	case err = <-result:
		cancel()
	case <-ctx.Done():
	}
	wg.Wait()
	return err
}

// CancelOnFirstError executes all given functions. When a function encounters an error all remaining functions will be canceled.
func CancelOnFirstError(ctx context.Context, funcs ...Func) error {
	if len(funcs) == 0 {
		glog.V(2).Infof("nothing to run")
		return nil
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	result := make(chan error)
	defer close(result)
	var wg sync.WaitGroup
	for _, runner := range funcs {
		wg.Add(1)
		go func(run Func) {
			defer wg.Done()
			if err := run(ctx); err != nil {
				select {
				case result <- err:
				default:
				}
			}
		}(runner)
	}
	var err error
	select {
	case err = <-result:
		cancel()
	case <-ctx.Done():
	}
	wg.Wait()
	return err
}

// All executes all given functions. Errors are wrapped into one aggregate error.
func All(ctx context.Context, funcs ...Func) error {
	if len(funcs) == 0 {
		glog.V(2).Infof("nothing to run")
		return nil
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	errors := make(chan error, len(funcs))
	var wg sync.WaitGroup
	for _, runner := range funcs {
		wg.Add(1)
		go func(run Func) {
			defer wg.Done()
			if err := run(ctx); err != nil {
				errors <- err
			}
		}(runner)
	}
	go func() {
		wg.Wait()
		close(errors)
	}()
	return NewErrorListByChan(errors)
}

// Sequential run every given function.
func Sequential(ctx context.Context, funcs ...Func) (err error) {
	if len(funcs) == 0 {
		glog.V(2).Infof("nothing to run")
		return nil
	}
	for _, fn := range funcs {
		select {
		case <-ctx.Done():
			glog.V(1).Infof("context canceled return")
			return nil
		default:
			if err = fn(ctx); err != nil {
				return
			}
		}
	}
	return
}
