// Copyright (c) 2020 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package run

import (
	"context"
	"time"

	"github.com/bborbe/errors"
)

var DefaultWaiter = NewWaiter()

// Backoff settings for retry
type Backoff struct {
	// Initial delay to wait on retry
	Delay time.Duration `json:"delay"`
	// Factor initial delay is multipled on retries
	Factor float64 `json:"factor"`
	// Retries how often to retry
	Retries int `json:"retries"`
	// IsRetryAble allow the check if error is retryable
	IsRetryAble func(error) bool `json:"-"`
}

// Retry on error n times and wait between the given delay.
func Retry(backoff Backoff, fn Func) Func {
	return func(ctx context.Context) error {
		var counter int
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				if err := fn(ctx); err != nil {
					if counter == backoff.Retries || backoff.IsRetryAble != nil && backoff.IsRetryAble(err) == false {
						return err
					}
					counter++

					if backoff.Delay > 0 {
						select {
						case <-ctx.Done():
							return ctx.Err()
						case <-time.NewTimer(backoff.Delay).C:
						}
						if err := DefaultWaiter.Wait(ctx, backoff.Delay); err != nil {
							return errors.Wrapf(ctx, err, "wait %v failed", backoff.Delay)
						}
					}
					continue
				}
				return nil
			}
		}
	}
}
