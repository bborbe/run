---
status: completed
execution_id: run-parallel-skipper-exec-007-parallel-skipper-stale-lock
dark-factory-version: v0.198.17
created: "2026-11-26T11:49:45Z"
queued: "2026-11-26T11:49:45Z"
started: "2026-11-26T11:50:11Z"
completed: "2026-11-26T11:54:09Z"
note: 'Completed with a recorded deviation: requirement 1 (add github.com/bborbe/time) is impossible — time -> bborbe/collection -> bborbe/run is an import cycle. Implemented an equivalent local CurrentDateTimeGetter clock seam plus stdlib time.Duration. make precommit exits 0.'
---
# Release stale ParallelSkipper lock after a configurable max running duration

<summary>
- A wrapped action that hangs and ignores context cancellation no longer blocks every later invocation forever.
- The parallel skipper can be given a maximum running duration; once the running action has held the lock longer than that, the next invocation force-releases the stale lock and runs.
- With no option given, behaviour is identical to today (the stale-release mechanism is disabled), so all existing callers keep working unchanged.
- Skipped invocations are now logged at default log level with how long the current run has been going, so a stuck action is visible in production logs.
- Force-releasing a stale lock is logged at default log level as well, with the same elapsed duration.
- The constructor stays source-compatible for every existing caller (no-argument calls still compile), so this ships as a minor version bump.
- The clock is injectable, so tests simulate hours of elapsed time without sleeping.
- A reproducer test proves both halves: the first invocation stays blocked and is skipped before the threshold, and the next invocation runs after the threshold.
- If the original hung action finally returns after the lock was taken over, it no longer frees the newer run's lock, so a third invocation cannot overlap with it.
</summary>

<objective>
Fix the stale-lock bug in `parallelSkipper.SkipParallel`: add functional options `WithMaxRunningDuration` and `WithCurrentDateTime` to `NewParallelSkipper`, track when the current run started, and force-release a lock held longer than `MaxRunningDuration` (0 = disabled, the default). Skip decisions and force-releases are logged at default level. End state: a hung wrapped action can no longer silently suppress all future runs once `MaxRunningDuration` is set, skips and take-overs are visible in default-level logs, and `run.NewParallelSkipper()` with no arguments behaves exactly as before.
</objective>

<context>
Read `docs/dod.md` for the definition of done. The repo has no `CLAUDE.md`; follow the conventions of the files you edit.

Read these coding-plugin docs before writing code:
- `/home/node/.claude/plugins/marketplaces/coding/docs/go-functional-options-pattern.md` (singular `XxxOption` func type, plural `XxxOptions` struct, `WithXxx` constructors, defaults set before applying options)
- `/home/node/.claude/plugins/marketplaces/coding/docs/go-time-injection.md` (use `github.com/bborbe/time`, never call `time.Now()` in production code, tests use `SetNow`)

Read `run_parallel-skipper.go`. Current code (verbatim, after the license header):

```go
package run

import (
	"context"
	"sync"

	"github.com/golang/glog"
)

// ParallelSkipper prevents parallel execution of wrapped functions.
// It ensures that only one instance of a wrapped function can run at a time, skipping subsequent calls.
type ParallelSkipper interface {
	SkipParallel(action Func) Func
}

// NewParallelSkipper creates a new ParallelSkipper that can wrap functions to prevent parallel execution.
// Each ParallelSkipper instance maintains its own execution state.
func NewParallelSkipper() ParallelSkipper {
	return &parallelSkipper{}
}

type parallelSkipper struct {
	running bool
	mux     sync.Mutex
}

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
		err := action(ctx)
		d.mux.Lock()
		glog.V(2).Infof("run finished => unlocked")
		d.running = false
		d.mux.Unlock()
		return err
	}
}
```

The bug: `running` is set to true before `action(ctx)` and only cleared after it returns. An action that hangs and ignores context cancellation leaves `running == true` forever, so every later invocation returns nil without running. The skip is logged at `glog.V(2)`, invisible at default log level. In production a fetcher silently produced no data for 4 days.

Read `run_parallel-skipper_test.go`. It is package `run_test`, Ginkgo/Gomega, with `Describe("ParallelSkipper", ...)`, a `BeforeEach` that sets `ctx = context.Background()`, and two specs `It("runs", ...)` and `It("skips sampe func parallel", ...)`. The existing imports are `context`, `sync`, `time`, `. "github.com/onsi/ginkgo/v2"`, `. "github.com/onsi/gomega"`, `github.com/bborbe/run`.

Read `run_suite_test.go`. The Ginkgo bootstrap is `func TestSuite(t *testing.T)` calling `RunSpecs(t, "Test Suite")`. Run a single spec with `go test ./... -run TestSuite -ginkgo.focus '<regex>' -ginkgo.v`.

Verified `github.com/bborbe/time` v1.27.8 API (import as `libtime "github.com/bborbe/time"`):
- `type CurrentDateTimeGetter interface { Now() DateTime }`
- `type CurrentDateTime interface { CurrentDateTimeGetter; CurrentDateTimeSetter }` where `CurrentDateTimeSetter` has `SetNow(now DateTime)`
- `func NewCurrentDateTime() CurrentDateTime` (returns real time until `SetNow` is called; safe for concurrent use)
- `type DateTime stdtime.Time` and `type Duration stdtime.Duration`; `Duration.String()` renders day-aware forms such as `5d12h`.

Callers of `run.NewParallelSkipper()` outside this module (`bborbe/http`, `bborbe/cron`) and `run.NewBackgroundRunner` inside this module call it with zero arguments. Variadic options keep them compiling.
</context>

<requirements>
1. **Add the dependency.** From the repo root run `go get github.com/bborbe/time@v1.27.8` then `go mod tidy`. `github.com/bborbe/time` must appear as a direct requirement at `v1.27.8` in `go.mod`.

2. **Edit `run_parallel-skipper.go` imports** to:

   ```go
   import (
   	"context"
   	"sync"
   	stdtime "time"

   	libtime "github.com/bborbe/time"
   	"github.com/golang/glog"
   )
   ```

3. **Add the option types** (exported, with doc comments), exactly:

   ```go
   // ParallelSkipperOptions holds optional configuration for a ParallelSkipper.
   type ParallelSkipperOptions struct {
   	// MaxRunningDuration is the longest a wrapped action may hold the lock before a later
   	// invocation force-releases it. 0 disables force-release (an action may hold the lock forever).
   	MaxRunningDuration libtime.Duration
   	// CurrentDateTimeGetter supplies the current time. Tests inject a fake clock.
   	CurrentDateTimeGetter libtime.CurrentDateTimeGetter
   }

   // ParallelSkipperOption modifies ParallelSkipperOptions.
   type ParallelSkipperOption func(*ParallelSkipperOptions)

   // WithMaxRunningDuration sets how long a wrapped action may hold the lock before a later
   // invocation force-releases it and runs. 0 disables force-release.
   func WithMaxRunningDuration(d libtime.Duration) ParallelSkipperOption {
   	return func(o *ParallelSkipperOptions) {
   		o.MaxRunningDuration = d
   	}
   }

   // WithCurrentDateTime sets the clock used to measure how long the lock has been held.
   func WithCurrentDateTime(getter libtime.CurrentDateTimeGetter) ParallelSkipperOption {
   	return func(o *ParallelSkipperOptions) {
   		o.CurrentDateTimeGetter = getter
   	}
   }
   ```

4. **Replace the constructor and struct** with:

   ```go
   // NewParallelSkipper creates a new ParallelSkipper that can wrap functions to prevent parallel execution.
   // Each ParallelSkipper instance maintains its own execution state.
   // Without options the lock is never force-released (MaxRunningDuration 0 = disabled).
   func NewParallelSkipper(options ...ParallelSkipperOption) ParallelSkipper {
   	parallelSkipperOptions := ParallelSkipperOptions{
   		MaxRunningDuration: 0,
   		// Deliberate deviation from go-time-injection.md (factories should receive a getter):
   		// this constructor is called with zero arguments from other modules, so requiring a
   		// getter would be a breaking API change. The getter is therefore defaulted here and
   		// replaceable via WithCurrentDateTime.
   		CurrentDateTimeGetter: libtime.NewCurrentDateTime(),
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
   	startedAt  libtime.DateTime // only meaningful while running is true
   	generation uint64           // incremented on every lock acquisition; identifies the current holder
   	mux        sync.Mutex
   }
   ```

5. **Replace `SkipParallel`** with the following. The log format strings and the `stdtime.Time(...).Sub(...)` elapsed computation are fixed:

   ```go
   func (d *parallelSkipper) SkipParallel(action Func) Func {
   	return func(ctx context.Context) error {
   		d.mux.Lock()
   		now := d.options.CurrentDateTimeGetter.Now()
   		if d.running {
   			elapsed := stdtime.Time(now).Sub(stdtime.Time(d.startedAt))
   			if d.options.MaxRunningDuration > 0 &&
   				elapsed > stdtime.Duration(d.options.MaxRunningDuration) {
   				glog.Infof("skip => already running for %s => force release", libtime.Duration(elapsed))
   			} else {
   				glog.Infof("skip => already running for %s", libtime.Duration(elapsed))
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
   ```

   Rules for this file:
   - The `generation` guard is required, not optional. When a stale lock is force-released, the original hung action is still executing. Without the guard its eventual `d.running = false` would clear the lock now held by the newer run, and a third invocation would then run concurrently with it. Each acquisition takes the next generation number; a returning action clears the lock only if it still owns the current generation.
   - Do not call `time.Now()`, `stdtime.Now()`, `libtime.Now()`, or `time.Since()` anywhere in it, and do not write the literal text `time.Now()` in a comment.
   - Both skip messages contain the literal substring `skip => already running for ` followed by the `libtime.Duration` rendering; the stale path appends ` => force release`. The text `already running for` appears in exactly two places in the file.
   - Keep the `glog.V(2).Infof("run started => locked")` and `glog.V(2).Infof("run finished => unlocked")` lines unchanged.
   - Do not change `SkipParallel`'s signature or the `ParallelSkipper` interface.

6. **Extend `run_parallel-skipper_test.go`** (no new test file). Add imports `"sync/atomic"` and `libtime "github.com/bborbe/time"`; keep the existing ones. Add the following specs inside the existing `Describe("ParallelSkipper", ...)`, in the same Ginkgo/Gomega style:

   a. `It("releases a stale lock after MaxRunningDuration and runs the next invocation", ...)`:
      - `currentDateTime := libtime.NewCurrentDateTime()`; `start := libtime.DateTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))`; `currentDateTime.SetNow(start)`.
      - `p := run.NewParallelSkipper(run.WithCurrentDateTime(currentDateTime), run.WithMaxRunningDuration(libtime.Duration(1*time.Hour)))`.
      - `var counter atomic.Int32`; `release := make(chan struct{})`; `started := make(chan struct{}, 2)`.
      - `fn := p.SkipParallel(func(ctx context.Context) error { counter.Add(1); started <- struct{}{}; <-release; return nil })`.
      - A `sync.WaitGroup` tracks the goroutines; each goroutine does `defer GinkgoRecover()` and `defer wg.Done()` and asserts `Expect(fn(ctx)).To(BeNil())`.
      - Start the first invocation in a goroutine, then block on `<-started` (use `Eventually(started).Should(Receive())`) so the lock is definitely held.
      - `currentDateTime.SetNow(start.Add(libtime.Duration(30 * time.Minute)))`; call `fn(ctx)` synchronously: assert it returns nil and `counter.Load()` is still `1`. This asserts the first invocation is still blocked and the call before the threshold is skipped.
      - `currentDateTime.SetNow(start.Add(libtime.Duration(2 * time.Hour)))`; call `fn(ctx)` in a second goroutine: `Eventually(counter.Load).Should(Equal(int32(2)))`. This asserts the stale lock was released and the second invocation ran.
      - `close(release)`; `wg.Wait()`.
      - Both halves (skipped before the threshold with counter 1, ran after the threshold with counter 2) must be asserted in the same spec.

   b. `It("never releases the lock when MaxRunningDuration is not set", ...)`: `p := run.NewParallelSkipper()` with no options. Use the same blocking-action shape as spec (a) with a `release` channel and a `started` channel; start the first invocation in a goroutine and wait for `started`; call `fn(ctx)` synchronously a second time and assert it returns nil and the counter stays `1`; then `close(release)` and `wg.Wait()`. This asserts the default (disabled) behaviour is unchanged.

   c. `It("keeps the force-released lock held when the hung invocation returns", ...)`: covers the generation guard. Same clock setup and a `MaxRunningDuration` of `1*time.Hour`. Declare `var counter atomic.Int32` and separate channels `releaseA`, `releaseB`, `startedA`, `startedB`, plus `fnC := p.SkipParallel(func(ctx context.Context) error { counter.Add(1); return nil })`. Wrap the blocking actions as `fnA` (signals `startedA`, waits `releaseA`) and `fnB` (signals `startedB`, waits `releaseB`) on the SAME skipper `p`.
      - Start `fnA` in a goroutine and wait for `startedA`.
      - Advance the clock past the threshold with `currentDateTime.SetNow(start.Add(libtime.Duration(2 * time.Hour)))` (stdlib `time.Duration` does not satisfy `libtime.HasDuration`, so it must be wrapped in `libtime.Duration`); start `fnB` in a goroutine and wait for `startedB`; assert `counter.Load()` is `2`.
      - `close(releaseA)` and wait for `fnA`'s goroutine to finish (a `doneA` channel closed via `defer`).
      - Call `fnC(ctx)` synchronously: assert it returns nil and `counter.Load()` is still `2` — the returning hung invocation must NOT have cleared the lock now held by `fnB`.
      - `close(releaseB)`; wait for `fnB`'s goroutine; call `fnC(ctx)` again: assert it returns nil and `counter.Load()` is `3` — the lock is released only once its true owner returns.

   Do not use `time.Sleep` in the new specs. The existing specs `runs` and `skips sampe func parallel` stay unchanged.

7. **Add a CHANGELOG entry.** `CHANGELOG.md` currently starts the version list at `## v1.12.3` with no `## Unreleased` section. Insert a new `## Unreleased` section immediately above `## v1.12.3` (after the semver preamble), with one bullet in the existing style:

   ```
   ## Unreleased

   - feat: add `WithMaxRunningDuration` and `WithCurrentDateTime` options to `NewParallelSkipper` so a stale lock held by a hung action is force-released, and log skips at default level (fixes silent permanent skipping)
   ```

   Leave a blank line between this section and `## v1.12.3`.

7b. **Update `README.md`.** In the `### Prevent Parallel Execution` section (the `skipper := run.NewParallelSkipper()` example, around lines 104-113), keep the existing example and add a second one showing the new option:

   ```go
   // Force-release a lock held longer than an hour by a hung action
   skipper := run.NewParallelSkipper(
       run.WithMaxRunningDuration(libtime.Duration(time.Hour)),
   )
   ```

   Add one sentence directly below it: passing no options leaves the default `MaxRunningDuration` at `0`, which never force-releases a lock and preserves the previous behaviour. Mention that skipped invocations are logged at default log level with the elapsed running time. Change no other README section.

8. **Run `make precommit`.** It must pass end-to-end. If it surfaces an unrelated pre-existing lint or formatting issue, fix it minimally; do not use it as cover to refactor unrelated code.
</requirements>

<constraints>
- Production code changes are scoped to `run_parallel-skipper.go` ONLY, plus `go.mod` and `go.sum` for the `github.com/bborbe/time` dependency. `run_parallel-skipper_test.go` is extended for tests, and `README.md` plus `CHANGELOG.md` are updated as documentation. No other file changes.
- Do not change `SkipParallel`'s signature or the `ParallelSkipper` interface.
- `NewParallelSkipper` must stay callable with zero arguments (variadic `...ParallelSkipperOption`); existing callers in this module and in `bborbe/http` and `bborbe/cron` must compile unchanged. This is a MINOR change, never MAJOR.
- Defaults: `MaxRunningDuration` is `0` (disabled, preserves today's behaviour for all existing callers) and `CurrentDateTimeGetter` is `libtime.NewCurrentDateTime()`. This deliberately overrides the `go-time-injection.md` rule that a factory should receive a getter instead of creating one, because the zero-argument constructor is called from three other modules and requiring a getter would be a breaking change. The default site carries a code comment stating exactly this trade-off.
- Do not remove the `run started => locked` / `run finished => unlocked` `glog.V(2)` logs.
- The `generation` counter is mandatory: a force-released run and the original hung run may both be executing, and only the current generation may clear the lock on return. Do not replace it with a boolean, a pointer comparison, or an unconditional `d.running = false`.
- Do not call `time.Now()` or `time.Since()` in non-test code.
- Do not add configuration knobs, metrics, or log lines beyond those specified.
- Do not add a new test file; extend `run_parallel-skipper_test.go`.
- Existing tests must still pass.
- Follow Ginkgo/Gomega conventions already in the test file.
- Don't commit — dark-factory handles git.
- No AI attribution anywhere (code, comments, CHANGELOG).
</constraints>

<verification>
Run `make precommit` — must pass.

Then run these checks from the repo root:

```
grep -c 'glog.V(2).Infof("skip' run_parallel-skipper.go   # must print 0
grep -c 'already running for' run_parallel-skipper.go      # must print 2
grep -c 'time.Now()' run_parallel-skipper.go               # must print 0
grep -c 'd.generation == myGeneration' run_parallel-skipper.go   # must print 1
go test ./... -run TestSuite -ginkgo.focus 'releases a stale lock' -ginkgo.v 2>&1 | grep -i 'releases a stale lock'   # must show the spec running
go test ./... -run TestSuite -ginkgo.focus 'force-released lock held' -ginkgo.v 2>&1 | grep -i 'force-released lock held'   # must show the spec running
go test ./... -run TestSuite -ginkgo.focus 'ParallelSkipper' 2>&1 | tail -5   # must report ok, all ParallelSkipper specs pass
```
</verification>
