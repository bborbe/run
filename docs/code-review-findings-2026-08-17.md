# Code Review — `github.com/bborbe/run`

**Date:** 2026-08-17
**Branch:** `master` @ `30fbd3a` (chore: bump go to 1.26.5)
**Reviewer:** DSH agent (qwen3.8:27b-mlx)
**Scope:** full source review of the package; baseline verification with `go vet`, `go test -race`, coverage.

## Baseline (healthy)

- `go vet ./...` — clean
- `go test -mod=mod -race -p=1 ./...` — **pass** (~10s)
- Source-line coverage — **98.8%** (excluding auto-generated `mocks/`, which is 0% by nature)
- `golangci-lint` config is strict: `govet`, `errcheck`, `staticcheck`, `unused`, ` revive`, `gosec`, `gocyclo`, `depguard`, `dupl`.
- The most subtle concurrency — `concurrentRunner` (`run_concurrent-runner.go`) — is written **correctly**. In particular the ordering that calls `wg.Done()` *before* the blocking `<-limit` release (and closes `limit`/`errs` only after `wg.Wait()`) avoids a send-on-closed-channel / deadlock. Good work there.

**Verdict:** no blocking issues, no data-race bugs. Findings below are correctness bugs, robustness gaps, and style nits, ordered by severity.

---

## Bugs worth fixing

### 1. `ParallelSkipper` deadlocks permanently if the wrapped fn panics  — `run_parallel-skipper.go:31`

The reset of the running flag happens *after* the wrapped action returns normally:

```go
func (d *parallelSkipper) SkipParallel(action Func) Func {
	return func(ctx context.Context) error {
		d.mux.Lock()
		if d.running {
			glog.V(2).Infof("skip => already running")
			d.mux.Unlock()
			return nil
		}
		glog.V(2).Infof("run started => locked")
		d.running = true
		d.mux.Unlock()
		err := action(ctx)                 // <-- if this PANICS, the lines below never run
		d.mux.Lock()
		glog.V(2).Infof("run finished => unlocked")
		d.running = false                  // <-- never reached on panic
		d.mux.Unlock()
		return err
	}
}
```

If `action` panics, the goroutine unwinds and `d.running` stays `true` **forever**. Every subsequent call then hits the `if d.running` branch and is skipped (`"skip => already running"`) — the skipper is permanently dead. This defeats the exact guarantee the type exists to provide ("only one instance runs at a time").

**Fix:** reset in a `defer` so the lock is released on any exit path:

```go
d.running = true
d.mux.Unlock()
defer func() {
	d.mux.Lock()
	d.running = false
	d.mux.Unlock()
}()
return action(ctx)
```

**Test gap:** the existing test `"skips sampe func parallel"` (`run_parallel-skipper_test.go`) never wraps a panicking fn, so this path is uncovered despite 98.8% coverage. Add a panic case.

---

### 2. `NewBackgroundRunner` logs "run started" *after* the run finishes  — `run_background-runner.go:27`

```go
return FuncRunnerFunc(func(runFunc Func) error {
	go func() {
		action := parallelSkipper.SkipParallel(func(ctx context.Context) error {
			if err := runFunc(ctx); err != nil {
				return err
			}
			return nil
		})
		glog.V(3).Infof("run started")                 // <-- runs AFTER action(ctx) below
		if err := action(ctx); err != nil {
			glog.Warningf("run failed: %v", err)
		}
		glog.V(3).Infof("run completed")
	}()
	return nil
})
```

`glog.V(3).Infof("run started")` is placed *after* `action(ctx)` (which runs the wrapped function). The log order is backwards: the "started" line only appears after the work has completed. Move it above `action(ctx)`.

---

### 3. `Retry` / `RetryWaiter` backoff is linear (not exponential), and the error reports the wrong value  — `run_retry.go:38`

```go
delay := backoff.Delay + backoff.Delay*time.Duration(
	backoff.Factor*float64(counter-1),
)
if err := waiter.Wait(ctx, delay); err != nil {
	return errors.Wrapf(ctx, err, "wait %v failed", backoff.Delay)  // reports backoff.Delay, not delay
}
```

Two problems:

- **Linear, not exponential.** The formula is `Delay + Delay·Factor·(counter-1)`, i.e. the delay grows by a constant `Delay·Factor` each attempt. Despite the field being named `Factor`, readers will reasonably expect an *exponential* multiplier (`Delay · Factor^n`). Either rename/re-document the field or change the math to match the name.
- **Wrong value in the error.** The wait-failure wraps `backoff.Delay` ("wait %v failed"), not the computed `delay`, so the reported duration is the *initial* delay, not the one that actually failed.
- **Ordering of checks.** `IsRetryAble` is evaluated *after* the `counter == backoff.Retries` check, so a non-retryable error on the final attempt surfaces as `"reached try counter(%d)"` instead of `"error is not retryable"`. Swap the two so the retryable classification takes precedence.

---

## Robustness / design smells

### 4. `NewConcurrentRunner` doesn't validate `maxConcurrent`  — `run_concurrent-runner.go:24`

```go
func NewConcurrentRunner(maxConcurrent int) ConcurrentRunner {
	return &concurrentRunner{
		maxConcurrent: maxConcurrent,
		fns:           make(chan Func, maxConcurrent),
		closed:        make(chan struct{}),
	}
}
```

- `maxConcurrent == 0` produces an *unbuffered* `fns`/`limit` channel — a behavioral surprise, not a panic, but not what a caller expects.
- A **negative** value panics at `make(chan Func, maxConcurrent)`.

Suggest validating (`if maxConcurrent < 1 { return error / panic with a clear message }`) or documenting a `> 0` contract on the constructor.

---

### 5. `ContextWithSig` registers the same signal twice  — `run_context-with-sig.go:25`

```go
signal.Notify(signalCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
```

`os.Interrupt` *is* `syscall.SIGINT`, so the list contains a duplicate. Harmless at runtime, but sloppy. Note that the recent revert (commit `2ebad97`, "revert v1.9.26 production signal-list expansion and misleading tests") removed `SIGUSR1`/`SIGUSR2` from this list but left this duplicate behind.

---

### 6. `MultiTrigger.Done()` is not idempotent and leaks goroutines  — `run_trigger-multi.go:44`

```go
func (m *multiTrigger) Done() <-chan struct{} {
	m.mux.Lock()
	defer m.mux.Unlock()

	result := NewTrigger()                 // fresh trigger on EVERY call
	var wg sync.WaitGroup
	counter := 0
	for _, trigger := range m.triggers {
		v := trigger
		select {
		case <-v.Done():
			counter++
		default:
			wg.Add(1)
			go func(done Done) {           // new waiter goroutine on EVERY call
				<-done.Done()
				wg.Done()
			}(v)
		}
	}
	if counter == len(m.triggers) {
		result.Fire()
	} else {
		go func() {                        // another fresh goroutine on EVERY call
			wg.Wait()
			result.Fire()
		}()
	}
	return result.Done()
}
```

Each `Done()` call allocates a fresh `result` trigger, a fresh set of waiter goroutines, and returns a *different* channel each time. Repeated calls spawn overlapping, uncoordinated goroutine sets that are never joined. Concretely, the tests `"nothing done"`, `"one done"`, `"two done"` (`run_trigger-multi_test.go`) call `Done()` and never fire the sub-triggers, so the `wg.Wait()` goroutine blocks **forever** — a goroutine leak (harmless only because the test process exits). This is surprising for a method that implements the `Done() <-chan` interface: callers reasonably expect idempotency / a stable channel.

---

### 7. `CancelOnFirstFinishWait` cancels on first *error*, not first *finish*  — `run_runner.go:26`

Doc comment: *"cancels the remaining functions when the first one completes."* Implementation:

```go
func CancelOnFirstFinishWait(ctx context.Context, funcs ...Func) error {
	...
	var errs []error
	for err := range Run(ctx, funcs...) {
		cancel()                       // cancelled only when an ERROR surfaces
		if err != nil {
			errs = append(errs, err)
		}
	}
	return NewErrorList(errs...)
}
```

It calls `cancel()` only when an *error* appears in the channel. A first-to-finish **success** (nil) does not trigger cancellation — which diverges from its sibling `CancelOnFirstFinish` (`run_runner.go:15`), which returns on the *first channel value regardless of success/failure*. The name/doc and the behavior don't match; clarify intent (cancel-on-finish vs cancel-on-error) and align the doc.

---

## Minor / style

- **`CatchPanic` loses the error chain** — `run_panic.go:21`: `fmt.Errorf("catch panic: %v", r)` uses `%v`; if `r` is an `error`, `%w` would preserve wrapping/unwrapping.
- **Trailing blank line** — `run_log.go:31` (extra blank line after the closing brace).
- **`coverage.out` committed and stale** — dated 2025-10-16, also listed in `.gitignore`. Double-check whether it should still be tracked (a stale generated artifact in the tree is a minor hazard).
- **Test-name typo** — `run_parallel-skipper_test.go`: `"skips sampe func parallel"` → "same".
- **`Run` buffer size** — `run_runner.go:116` uses `runtime.NumCPU()` as the result-channel buffer. Arbitrary but correct (producers just block on send when it's full). No action needed; noting for completeness.
- **`Sequential` cancellation granularity** — `run_runner.go:92` checks `ctx.Done()` only *between* functions, not during them. Expected for sequential execution, but worth a one-line doc note.

---

## Suggested priority

| # | Item | Severity | Effort |
|---|------|----------|--------|
| 1 | `ParallelSkipper` panic → permanent lockout | **bug** | small (defer reset) |
| 2 | `NewBackgroundRunner` log-order | **bug** (misleading) | tiny |
| 3 | `Retry` linear-vs-exponential + wrong error value + check order | **bug/design** | small |
| 4 | `NewConcurrentRunner` no `maxConcurrent` validation | robustness | small |
| 5 | `ContextWithSig` duplicate signal | hygiene | tiny |
| 6 | `MultiTrigger.Done()` non-idempotent + leak | design | medium |
| 7 | `CancelOnFirstFinishWait` semantics vs name | design | small |
| — | `CatchPanic` `%w`, blank line, typo, stale `coverage.out` | nit | tiny |

Items 1–3 are the clear "fix it" set; 4–7 are judgment calls; the nits are cheap.
