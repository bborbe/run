package run

import "context"

// Func interface for all run utils.
type Func func(context.Context) error
