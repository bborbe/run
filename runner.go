package runner

import (
	"context"

	"github.com/golang/glog"
)

type run func(context.Context) error

func RunAndWaitOnFirst(runners ...run) error {
	if len(runners) == 0 {
		glog.V(2).Infof("nothing to run")
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errors := make(chan error)
	for _, runner := range runners {
		run := runner
		go func() {
			errors <- run(ctx)
		}()
	}

	return <-errors
}
