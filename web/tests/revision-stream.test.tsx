import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import {
  afterEach,
  beforeEach,
  describe,
  expect,
  it,
  vi,
  type MockInstance,
} from "vitest";
import { useRevisionStream } from "../src/hooks";
import { localWriteRevisionWindowMs } from "../src/revision-status";
import {
  backoffFor,
  initialBackoffMs,
  maxBackoffMs,
  receiveTimeoutMs,
} from "../src/revision-stream";

const streamMocks = vi.hoisted(() => ({ watchRevision: vi.fn() }));

vi.mock("../src/api", () => ({
  getSnapshot: vi.fn(),
  getConfig: vi.fn(),
  getSyncStatus: vi.fn(),
  syncIfDue: vi.fn(),
  getDebugReport: vi.fn(),
  getPromptTemplates: vi.fn(),
  watchRevision: streamMocks.watchRevision,
}));

// controllableStream はサーバーからの送信を試験から 1 通ずつ流す。接続が開いた
// ことも待てるようにする。
function controllableStream() {
  const pending: { revision: bigint }[] = [];
  const state = { ended: false };
  let wake: (() => void) | undefined;
  let opened: () => void = () => undefined;
  const connected = new Promise<void>((resolve) => {
    opened = resolve;
  });
  return {
    connected,
    send(revision: number) {
      pending.push({ revision: BigInt(revision) });
      wake?.();
    },
    end() {
      state.ended = true;
      wake?.();
    },
    // abort は本物の購読と同じく例外で抜ける。無視すると受信タイムアウトの
    // 試験が接続の張り直しまで進まない。
    iterable: async function* (signal?: AbortSignal) {
      opened();
      signal?.addEventListener("abort", () => wake?.());
      while (!state.ended) {
        if (signal?.aborted) throw new Error("aborted");
        while (pending.length > 0) {
          const message = pending.shift();
          if (message) yield message;
        }
        await new Promise<void>((resolve) => {
          wake = resolve;
        });
      }
    },
  };
}

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    );
  };
}

describe("backoffFor", () => {
  it("doubles the delay up to the cap and keeps the jitter inside half of it", () => {
    expect(backoffFor(0, () => 1)).toBe(initialBackoffMs);
    expect(backoffFor(0, () => 0)).toBe(initialBackoffMs / 2);
    expect(backoffFor(3, () => 1)).toBe(initialBackoffMs * 8);
    expect(backoffFor(20, () => 1)).toBe(maxBackoffMs);
    expect(backoffFor(20, () => 0)).toBe(maxBackoffMs / 2);
  });
});

describe("useRevisionStream", () => {
  let queryClient: QueryClient;
  let invalidate: MockInstance<QueryClient["invalidateQueries"]>;

  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers({ shouldAdvanceTime: true });
    queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    invalidate = vi
      .spyOn(queryClient, "invalidateQueries")
      .mockResolvedValue(undefined);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  function invalidatedKeys(): (string | undefined)[] {
    return invalidate.mock.calls.map(
      (call) => (call[0]?.queryKey as string[] | undefined)?.[0],
    );
  }

  // 切断中の変更を必ず拾うため、接続が成立するたびに無条件で捨てる。
  it("drops the cached snapshot and sync status as soon as it connects", async () => {
    const stream = controllableStream();
    streamMocks.watchRevision.mockImplementation((signal: AbortSignal) =>
      stream.iterable(signal),
    );
    const { result } = renderHook(() => useRevisionStream(), {
      wrapper: createWrapper(queryClient),
    });
    await stream.connected;
    act(() => {
      stream.send(1);
    });
    await waitFor(() => {
      expect(result.current.connected).toBe(true);
    });
    expect(invalidatedKeys()).toEqual(["snapshot", "github-sync-status"]);
  });

  it("drops them again when the revision changes but not on a heartbeat", async () => {
    const stream = controllableStream();
    streamMocks.watchRevision.mockImplementation((signal: AbortSignal) =>
      stream.iterable(signal),
    );
    renderHook(() => useRevisionStream(), {
      wrapper: createWrapper(queryClient),
    });
    await stream.connected;
    act(() => {
      stream.send(1);
    });
    await waitFor(() => {
      expect(invalidatedKeys()).toHaveLength(2);
    });
    act(() => {
      stream.send(1);
    });
    act(() => {
      stream.send(2);
    });
    await waitFor(() => {
      expect(invalidatedKeys()).toHaveLength(4);
    });
  });

  // サーバーは自分の書き込みも他人のものと同じく検知するので、抑えないと 1 回の
  // 変更で 2 回取り直し、2 回目が操作中の画面に届く。
  it("ignores a revision this tab's own mutation caused", async () => {
    const stream = controllableStream();
    streamMocks.watchRevision.mockImplementation((signal: AbortSignal) =>
      stream.iterable(signal),
    );
    renderHook(() => useRevisionStream(), {
      wrapper: createWrapper(queryClient),
    });
    await stream.connected;
    act(() => {
      stream.send(1);
    });
    await waitFor(() => {
      expect(invalidatedKeys()).toHaveLength(2);
    });
    await act(async () => {
      await queryClient
        .getMutationCache()
        .build(queryClient, { mutationFn: () => Promise.resolve("written") })
        .execute(undefined);
    });
    act(() => {
      stream.send(2);
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(100);
    });
    expect(invalidatedKeys()).toHaveLength(2);

    // 猶予を過ぎた後の変更は、外からの書き込みとして取り直す。
    await act(async () => {
      await vi.advanceTimersByTimeAsync(localWriteRevisionWindowMs);
    });
    act(() => {
      stream.send(3);
    });
    await waitFor(() => {
      expect(invalidatedKeys()).toHaveLength(4);
    });
  });

  // heartbeat が途絶えたら、接続が生きて見えても張り直す。
  it("reconnects after the receive timeout and after a backoff", async () => {
    const first = controllableStream();
    const second = controllableStream();
    streamMocks.watchRevision
      .mockImplementationOnce((signal: AbortSignal) => first.iterable(signal))
      .mockImplementation((signal: AbortSignal) => second.iterable(signal));
    const { result } = renderHook(() => useRevisionStream(), {
      wrapper: createWrapper(queryClient),
    });
    await first.connected;
    act(() => {
      first.send(1);
    });
    await waitFor(() => {
      expect(result.current.connected).toBe(true);
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(receiveTimeoutMs + 1);
    });
    await waitFor(() => {
      expect(result.current.connected).toBe(false);
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(maxBackoffMs);
    });
    await second.connected;
    act(() => {
      second.send(7);
    });
    await waitFor(() => {
      expect(result.current.connected).toBe(true);
    });
    expect(streamMocks.watchRevision.mock.calls.length).toBeGreaterThan(1);
  });

  // 単発の切断では警告を出さない。頻繁に起こり得るので、そのたびに出すと表示が
  // 信用されなくなる。
  it("reports a stale stream only after it stays down for the notice delay", async () => {
    streamMocks.watchRevision.mockImplementation(() => {
      throw new Error("the stream is unavailable");
    });
    const { result } = renderHook(() => useRevisionStream(), {
      wrapper: createWrapper(queryClient),
    });
    await waitFor(() => {
      expect(result.current.connected).toBe(false);
    });
    await act(async () => {
      await vi.advanceTimersByTimeAsync(29_000);
    });
    expect(result.current.stale).toBe(false);
    await act(async () => {
      await vi.advanceTimersByTimeAsync(2_000);
    });
    await waitFor(() => {
      expect(result.current.stale).toBe(true);
    });
  });

  // StrictMode の二重マウントで 2 本張らないよう、後片付けで購読を止める。
  it("closes the subscription when the hook unmounts", async () => {
    const stream = controllableStream();
    const aborted = vi.fn();
    streamMocks.watchRevision.mockImplementation((signal: AbortSignal) => {
      signal.addEventListener("abort", aborted);
      return stream.iterable(signal);
    });
    const { unmount } = renderHook(() => useRevisionStream(), {
      wrapper: createWrapper(queryClient),
    });
    await stream.connected;
    unmount();
    expect(aborted).toHaveBeenCalled();
  });
});
