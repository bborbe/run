// Copyright (c) 2020 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package run

import (
	"context"
	"sync"
	stdtime "time"

	"github.com/golang/glog"
)

// CurrentDateTimeGetter supplies the current time. Tests inject a fake clock.
type CurrentDateTimeGetter interface {
	Now() stdtime.Time
}

// systemClockNow is a variable so the real clock is only referenced through an
// injectable seam. Production uses the system clock by default.
var systemClockNow = stdtime.Now

type systemCurrentDateTime struct{}

func (systemCurrentDateTime) Now() stdtime.Time {
	return systemClockNow()
}

// ParallelSkipper prevents parallel execution of wrapped functions.
// It ensures that only one instance of a wrapped function can run at a time, skipping subsequent calls.
type ParallelSkipper interface {
	SkipParallel(action Func) Func
}

// ParallelSkipperOptions holds optional configuration for a ParallelSkipper.
type ParallelSkipperOptions struct {
	// MaxRunningDuration is the longest a wrapped action may hold the lock before a later
	// invocation force-releases it. 0 disables force-release (an action may hold the lock forever).
	MaxRunningDuration stdtime.Duration
	// CurrentDateTimeGetter supplies the current time. Tests inject a fake clock.
	CurrentDateTimeGetter CurrentDateTimeGetter
}

// ParallelSkipperOption modifies ParallelSkipperOptions.
type ParallelSkipperOption func(*ParallelSkipperOptions)

// WithMaxRunningDuration sets how long a wrapped action may hold the lock before a later
// invocation force-releases it and runs. 0 disables force-release.
func WithMaxRunningDuration(d stdtime.Duration) ParallelSkipperOption {
	return func(o *ParallelSkipperOptions) {
		o.MaxRunningDuration = d
	}
}

// WithCurrentDateTime sets the clock used to measure how long the lock has been held.
func WithCurrentDateTime(getter CurrentDateTimeGetter) ParallelSkipperOption {
	return func(o *ParallelSkipperOptions) {
		o.CurrentDateTimeGetter = getter
	}
}

// NewParallelSkipper creates a new ParallelSkipper that can wrap functions to prevent parallel execution.
// Each ParallelSkipper instance maintains its own execution state.
// Without options the lock is never force-released (MaxRunningDuration 0 = disabled).
func NewParallelSkipper(options ...ParallelSkipperOption) ParallelSkipper {
	parallelSkipperOptions := ParallelSkipperOptions{
		MaxRunningDuration: 0,
		// This constructor is called with zero arguments from other modules, so requiring a
		// clock would be a breaking API change. The clock is therefore defaulted here and
		// replaceable via WithCurrentDateTime.
		CurrentDateTimeGetter: systemCurrentDateTime{},
	}
	for _, option := range options {
		option(&parallelSkipperOptions)
	}
	return &parallelSkipper{
		options: parallelSkipperOptions,
	}
}

type parallelSkipper struct {
	options    ParallelSkipperOptions
	running    bool
	startedAt  stdtime.Time // only meaningful while running is true
	generation uint64       // incremented on every lock acquisition; identifies the current holder
	mux        sync.Mutex
}

func (d *parallelSkipper) SkipParallel(action Func) Func {
	return func(ctx context.Context) error {
		d.mux.Lock()
		now := d.options.CurrentDateTimeGetter.Now()
		if d.running {
			elapsed := now.Sub(d.startedAt)
			if d.options.MaxRunningDuration > 0 &&
				elapsed > d.options.MaxRunningDuration {
				glog.Infof("skip => already running for %s => force release", elapsed)
			} else {
				glog.Infof("skip => already running for %s", elapsed)
				d.mux.Unlock()
				return nil
			}
		}
		glog.V(2).Infof("run started => locked")
		d.generation++
		myGeneration := d.generation
		d.running = true
		d.startedAt = now
		d.mux.Unlock()
		err := action(ctx)
		d.mux.Lock()
		if d.generation == myGeneration {
			glog.V(2).Infof("run finished => unlocked")
			d.running = false
		} else {
			glog.V(2).Infof("run finished => lock already released")
		}
		d.mux.Unlock()
		return err
	}
}
