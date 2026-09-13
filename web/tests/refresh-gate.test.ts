import { describe, expect, it, vi } from "vitest";
import { holdRefresh, runWhenIdle } from "../src/refresh-gate";

describe("refresh gate", () => {
  it("runs immediately while nothing holds it", () => {
    const run = vi.fn();
    runWhenIdle(run);
    expect(run).toHaveBeenCalledOnce();
  });

  // 依存をつなぐドラッグの最中に描き直すと、掴んでいた接続がその場で消える。
  it("defers a refresh until the last hold is released", () => {
    const run = vi.fn();
    const release = holdRefresh();
    runWhenIdle(run);
    expect(run).not.toHaveBeenCalled();
    release();
    expect(run).toHaveBeenCalledOnce();
  });

  it("waits for every hold and releases are idempotent", () => {
    const run = vi.fn();
    const first = holdRefresh();
    const second = holdRefresh();
    runWhenIdle(run);
    first();
    first();
    expect(run).not.toHaveBeenCalled();
    second();
    expect(run).toHaveBeenCalledOnce();
  });

  // 取り直しは冪等なので、保留中に溜まった分は最後の 1 回に畳む。
  it("collapses repeated requests into the latest one", () => {
    const stale = vi.fn();
    const latest = vi.fn();
    const release = holdRefresh();
    runWhenIdle(stale);
    runWhenIdle(latest);
    release();
    expect(stale).not.toHaveBeenCalled();
    expect(latest).toHaveBeenCalledOnce();
  });

  it("forgets a deferred refresh once it has run", () => {
    const run = vi.fn();
    const first = holdRefresh();
    runWhenIdle(run);
    first();
    const second = holdRefresh();
    second();
    expect(run).toHaveBeenCalledOnce();
  });
});
