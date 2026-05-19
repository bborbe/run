---
status: committing
summary: Fixed close-on-registered panic in ContextWithSig by replacing defer close(signalCh) with defer signal.Stop(signalCh), removed dead if !ok branch, and added Ginkgo tests for signal delivery paths achieving 100% coverage
container: run-exec-005-test-context-with-sig-signal-delivery
dark-factory-version: v0.163.4-1-gce8e514
created: "2026-05-19T18:30:00Z"
queued: "2026-05-19T16:53:32Z"
started: "2026-05-19T20:19:45Z"
completed: "2026-05-19T19:21:22Z"
lastFailReason: 'execute prompt: docker run failed: wait command: exit status 137'
---

# Cover the signal-delivery path in ContextWithSig (and fix close-on-registered panic)

<summary>
- `ContextWithSig` cancels its returned context on SIGINT/SIGTERM, but the existing tests only cover parent-context cancellation.
- The signal-received branch is currently untested, leaving the function at 66.7% coverage.
- A previous attempt to add signal-delivery tests surfaced a real bug: the goroutine does `defer close(signalCh)` while `signalCh` is still registered with `signal.Notify`. After the first signal is delivered and the goroutine exits, the channel is closed but `signal.Notify` may still attempt a non-blocking send on a subsequent signal — panic on send-to-closed.
- Fix the production code with the canonical Go idiom: `signal.Stop(signalCh)` to unregister before the channel is GC'd. Drop the `defer close(signalCh)` and the now-unreachable `if !ok` branch on the receive.
- Add Ginkgo tests that deliver real SIGINT and SIGTERM via `syscall.Kill(os.Getpid(), ...)` and assert the returned context is cancelled.
- After the change, `ContextWithSig` is at 100% coverage and no longer panic-prone under repeated signal delivery.
</summary>

<objective>
Replace `defer close(signalCh)` in `run_context-with-sig.go` with `defer signal.Stop(signalCh)` (positioned after `signal.Notify`), remove the dead `if !ok` branch, and add Ginkgo tests in `run_context-with-sig_test.go` that send real OS signals to verify the returned context is cancelled. End state: 100% coverage of `ContextWithSig` and no latent panic on repeated signal delivery.
</objective>

<context>
Read `CLAUDE.md` and `docs/dod.md` for project conventions.

Read `run_context-with-sig.go`. Current shape (verbatim):

```go
func ContextWithSig(ctx context.Context) context.Context {
	ctxWithCancel, cancel := context.WithCancel(ctx) // #nosec G118
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
```

The bug: `defer close(signalCh)` runs while `signal.Notify` still has `signalCh` in its internal registration list. If any further signal arrives after the goroutine has exited the `select`, the runtime does a non-blocking send to the now-closed channel and panics. This is observable in tests that repeatedly call `ContextWithSig` and deliver signals.

The canonical fix is `signal.Stop(signalCh)`. From Go's `os/signal` package docs: *"Stop causes package signal to stop relaying incoming signals to c. It undoes the effect of all prior calls to Notify using c."* After `signal.Stop`, no further sends happen and the channel can be GC'd without an explicit close.

Read `run_context-with-sig_test.go` for the existing Ginkgo/Gomega test style. The outer `BeforeEach` at lines 22-25 creates a shared `sigCtx` — do NOT reuse it for signal-delivery tests, because `syscall.Kill` delivers to every registered handler in the process. Each new `It` must build its own isolated `parent context + cancel + sigCtx` triple and tear it down before returning.

Key behavior:
- Once any `signal.Notify` has registered SIGINT/SIGTERM, the Go runtime no longer applies the default "terminate process" action for those signals. So `syscall.Kill(os.Getpid(), syscall.SIGTERM)` is safe inside a test process.
- `signal.Stop(signalCh)` MUST be deferred AFTER `signal.Notify(signalCh, ...)` so it actually has something to undo. Defers run LIFO, so listing `defer cancel()` first and `defer signal.Stop(signalCh)` second means `signal.Stop` runs first on exit, then `cancel` — exactly what we want.
- On Unix `os.Interrupt == syscall.SIGINT`, so the source's three-signal `signal.Notify` covers two distinct values. Testing SIGINT and SIGTERM covers both branches of real behavior.
</context>

<requirements>
1. **Edit `run_context-with-sig.go`.** Replace the goroutine body so it reads:

   ```go
   go func() {
       defer cancel()

       signalCh := make(chan os.Signal, 1)
       signal.Notify(signalCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
       defer signal.Stop(signalCh)

       select {
       case sig := <-signalCh:
           glog.V(2).Infof("got signal %s => cancel context ", sig)
       case <-ctx.Done():
       }
   }()
   ```

   Specifically:
   - Remove `defer close(signalCh)`.
   - Add `defer signal.Stop(signalCh)` immediately AFTER the `signal.Notify` call.
   - Replace `case signal, ok := <-signalCh:` + the `if !ok` block with `case sig := <-signalCh:` (no `ok` variable needed because the channel is no longer closed).
   - Preserve the `glog.V(2).Infof("got signal %s => cancel context ", sig)` line (rename the variable from `signal` to `sig` to avoid shadowing the `signal` package).
   - Leave the function's doc comment, signature, and `#nosec G118` annotation unchanged.
   - Leave imports unchanged (`signal.Stop` is in the already-imported `os/signal` package).

2. **Edit `run_context-with-sig_test.go`.** Add a new `Context("Signal delivery", func() { ... })` block inside the existing `Describe("ContextWithSig", ...)`. Inside it, add two `It` blocks:
   - `"cancels signal context when SIGTERM is received"` — send the signal via `syscall.Kill(os.Getpid(), syscall.SIGTERM)` from a goroutine with `time.Sleep(50 * time.Millisecond)` (matching the timing style at lines 33-37 of the existing file), then `select` on `<-sigCtx.Done()` with `time.After(200 * time.Millisecond)`, and assert `sigCtx.Err()` is `context.Canceled`.
   - `"cancels signal context when SIGINT is received"` — same shape, with `syscall.SIGINT`.

3. **Each new `It` must build its own isolated context triple** and tear it down before returning:
   - Locally declare `parentCtx, parentCancel := context.WithCancel(context.Background())`.
   - Locally declare `sigCtx := run.ContextWithSig(parentCtx)`.
   - At the end of the test, call `parentCancel()` and then `Eventually(sigCtx.Done()).Should(BeClosed())` so the spawned goroutine and its `signal.Stop` defer run before the next test starts. Do NOT use the outer `BeforeEach`'s shared `sigCtx`.

4. **Imports to add** to the test file's import block (positioned alphabetically among stdlib imports):
   - `"os"`
   - `"syscall"`

5. **Capture and assert the `syscall.Kill` return**: `err := syscall.Kill(os.Getpid(), syscall.SIGTERM)` followed by `Expect(err).NotTo(HaveOccurred())`.

6. **Do NOT add new test files.** Extend the existing `run_context-with-sig_test.go`.

7. **Run `make precommit`.** Must pass end-to-end. If it surfaces an unrelated pre-existing lint or formatting issue, fix it minimally; do NOT use it as cover to refactor unrelated code.

8. **Coverage gate.** After `make precommit` passes, run:

   ```
   go test -coverprofile=/tmp/cover.out ./... && go tool cover -func=/tmp/cover.out | awk '/ContextWithSig/ && $3 != "100.0%" {print; exit 1}'
   ```

   The command must exit 0. Any `ContextWithSig` line whose third column is not `100.0%` causes a non-zero exit.
</requirements>

<constraints>
- Don't commit — dark-factory handles git
- Production change is scoped to `run_context-with-sig.go` ONLY. No other production files.
- Behavior contract for `ContextWithSig` callers is unchanged: it still returns a context that cancels on parent cancellation or on SIGINT/SIGTERM. The internal "channel was closed externally" branch was dead code anyway (no code ever closed `signalCh` from outside the goroutine), so removing it is not a contract change.
- Existing tests must still pass
- Follow Ginkgo/Gomega conventions already in the file (`Describe`/`Context`/`It`, `Expect(...).To(...)`)
- Use `syscall.Kill(os.Getpid(), syscall.SIG...)` for signal delivery — do not spawn subprocesses
- Do not introduce flaky timing: keep the post-signal `time.After` timeout at 200ms or more (matches existing tests at lines 44, 149, 222)
- Add a `## Unreleased` CHANGELOG.md entry describing both the bug fix and the new coverage (per `docs/dod.md`)
</constraints>

<verification>
Run `make precommit` — must pass.

Then assert coverage as a hard gate:
```
go test -coverprofile=/tmp/cover.out ./... && go tool cover -func=/tmp/cover.out | awk '/ContextWithSig/ && $3 != "100.0%" {print; exit 1}'
```
The command must exit 0. Any `ContextWithSig` line whose third column is not `100.0%` causes a non-zero exit.

Also verify the bug fix landed by grep:
```
grep -c 'close(signalCh)' run_context-with-sig.go   # must print 0
grep -c 'signal.Stop(signalCh)' run_context-with-sig.go  # must print 1
```
</verification>
