---
status: failed
container: run-exec-005-test-context-with-sig-signal-delivery
dark-factory-version: v0.163.2-1-g54e4b3a
created: "2026-05-19T18:30:00Z"
queued: "2026-05-19T16:53:32Z"
started: "2026-05-19T18:07:55Z"
completed: "2026-05-19T18:19:11Z"
lastFailReason: 'execute prompt: docker run failed: wait command: exit status 137'
---

# Cover the signal-delivery path in ContextWithSig

<summary>
- `ContextWithSig` cancels its returned context on SIGINT/SIGTERM, but the existing tests only cover parent-context cancellation.
- The signal-received branch is currently untested, leaving the function at 66.7% coverage.
- Add a Ginkgo test that delivers an actual signal to the process and asserts the returned context is cancelled.
- Use both SIGINT and SIGTERM to cover both registered signals.
- After the change, `ContextWithSig` reaches 100% coverage.
- No production code changes — test-only addition.
</summary>

<objective>
Add Ginkgo tests in `run_context-with-sig_test.go` that send real OS signals (SIGINT and SIGTERM) via `syscall.Kill(os.Getpid(), ...)` and verify the context returned by `run.ContextWithSig` is cancelled. This closes the only meaningful coverage gap in the package's production code.
</objective>

<context>
Read `CLAUDE.md` and `docs/dod.md` for project conventions.

Read `run_context-with-sig.go` — `ContextWithSig` registers a channel via `signal.Notify(signalCh, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)` inside a goroutine. When a signal arrives, the goroutine cancels the returned context and exits. Currently no test triggers this branch.

Read `run_context-with-sig_test.go` for the existing Ginkgo/Gomega test style in this file. New tests must follow the same `Describe`/`Context`/`It` shape. The outer `BeforeEach` at lines 22-25 already creates a shared `sigCtx` — do NOT reuse it for signal-delivery tests, because `syscall.Kill` delivers to every registered handler in the process. Each new `It` must build its own isolated `parent context + cancel + sigCtx` triple and tear it down before returning (see requirement 3).

Key behavior:
- Once `signal.Notify` has registered SIGINT/SIGTERM with any channel, the Go runtime no longer applies the default "terminate process" action for those signals — they are delivered only to registered channels. So `syscall.Kill(os.Getpid(), syscall.SIGTERM)` from within a test is safe and will not kill the test binary.
- The signal-handling goroutine is per-call to `ContextWithSig`. Each invocation registers an independent channel via `signal.Notify`, so signals fan out to every still-alive registration.
- On Unix `os.Interrupt == syscall.SIGINT`, so the source's three-signal `signal.Notify` call (`os.Interrupt`, `syscall.SIGINT`, `syscall.SIGTERM`) effectively covers two distinct values. Testing SIGINT and SIGTERM covers both branches of real behavior.
</context>

<requirements>
1. Add a new `Context("Signal delivery", func() { ... })` block inside the existing `Describe("ContextWithSig", ...)` in `run_context-with-sig_test.go`.

2. Inside that `Context`, add two `It` blocks:
   - `"cancels signal context when SIGTERM is received"` — call `syscall.Kill(os.Getpid(), syscall.SIGTERM)` after a short delay (use a goroutine with `time.Sleep(50 * time.Millisecond)` matching the existing test style at lines 33-37), then `select` on `<-sigCtx.Done()` with a `time.After(200 * time.Millisecond)` timeout, and assert `sigCtx.Err()` equals `context.Canceled`.
   - `"cancels signal context when SIGINT is received"` — same shape, with `syscall.SIGINT`.

3. Each `It` must create its own `parent context + cancel + sigCtx` triple (do NOT rely on the outer `BeforeEach`'s `sigCtx` — the outer one is shared across `It`s in the same `BeforeEach` scope, and we want a fresh signal handler per test to avoid cross-test signal delivery). After asserting the cancellation, the test must explicitly cancel its parent context AND wait for the signal-handler goroutine to exit before returning — use `cancel()` followed by an `Eventually(sigCtx.Done()).Should(BeClosed())` (or equivalent) so no live `signal.Notify` registration leaks into the next test.

4. Imports to add to the test file's import block:
   - `"os"`
   - `"syscall"`

5. Do NOT modify `run_context-with-sig.go`. Test-only change.

6. Do NOT add new test files. Extend the existing `run_context-with-sig_test.go`.

7. After the change, run `make precommit` and verify it passes end-to-end.

8. Verify coverage of `ContextWithSig` reaches 100% via:
   ```
   go test -coverprofile=/tmp/cover.out ./... && go tool cover -func=/tmp/cover.out | grep ContextWithSig
   ```
   The reported coverage for `ContextWithSig` must be `100.0%`.
</requirements>

<constraints>
- Don't commit — dark-factory handles git
- Test-only change; do not touch production code in `run_context-with-sig.go` or any other `.go` file outside `_test.go`
- Existing tests must still pass
- Follow Ginkgo/Gomega conventions already in the file (`Describe`/`Context`/`It`, `Expect(...).To(...)`)
- Use `syscall.Kill(os.Getpid(), syscall.SIG...)` for signal delivery — do not spawn subprocesses
- Do not introduce flaky timing: keep the post-signal `time.After` timeout at 200ms or more (matches existing tests at lines 44, 149, 222)
- Error handling: `syscall.Kill` returns an error — capture and assert it via `Expect(err).NotTo(HaveOccurred())`
</constraints>

<verification>
Run `make precommit` — must pass.

Then assert coverage as a hard gate:
```
go test -coverprofile=/tmp/cover.out ./... && go tool cover -func=/tmp/cover.out | awk '/ContextWithSig/ && $3 != "100.0%" {print; exit 1}'
```
The command must exit 0. Any `ContextWithSig` line whose third column is not `100.0%` causes a non-zero exit.
</verification>
