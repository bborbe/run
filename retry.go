// Copyright (c) 2020 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package run

import (
	"context"
	"time"
)

// Backoff settings for retry
type Backoff struct {
	Delay       time.Duration    `json:"delay"`
	Retries     int              `json:"retries"`
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
					}
					continue
				}
				return nil
			}
		}
	}
}
