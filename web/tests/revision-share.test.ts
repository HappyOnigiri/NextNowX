import { beforeEach, describe, expect, it, vi } from "vitest";
import { startSharedRevisionStream } from "../src/revision-share";
import type { RevisionStreamHandlers } from "../src/revision-stream";

const streamMocks = vi.hoisted(() => ({ startRevisionStream: vi.fn() }));

vi.mock("../src/revision-stream", () => ({
  startRevisionStream: streamMocks.startRevisionStream,
}));

interface FakeChannel {
  name: string;
  onmessage: ((event: MessageEvent) => void) | null;
  closed: boolean;
  postMessage: (data: unknown) => void;
  close: () => void;
}

// fakeBus は同じ名前の BroadcastChannel をつないだブラウザプロファイルを模す。
// 本物と同じく、送り手自身には配らない。
function fakeBus() {
  const channels: FakeChannel[] = [];
  const create = (name: string): BroadcastChannel => {
    const channel: FakeChannel = {
      name,
      onmessage: null,
      closed: false,
      postMessage: (data) => {
        for (const other of channels) {
          if (other === channel || other.name !== name) continue;
          other.onmessage?.({ data } as MessageEvent);
        }
      },
      close: () => {
        channel.closed = true;
      },
    };
    channels.push(channel);
    return channel as unknown as BroadcastChannel;
  };
  return { create };
}

// fakeLocks は最初の要求だけに許可を出し、あとは待ち行列に積む。解放で次へ渡す。
function fakeLocks() {
  const waiting: (() => void)[] = [];
  let held = false;
  const grant = (run: () => Promise<void>) => {
    held = true;
    void run().then(() => {
      held = false;
      waiting.shift()?.();
    });
  };
  const request = (_name: string, callback: () => Promise<void>) => {
    if (held)
      waiting.push(() => {
        grant(callback);
      });
    else grant(callback);
    return Promise.resolve();
  };
  return { manager: { request } as unknown as LockManager };
}

function handlers(): RevisionStreamHandlers & {
  calls: { connect: bigint[]; revision: bigint[]; disconnect: number };
} {
  const calls = {
    connect: [] as bigint[],
    revision: [] as bigint[],
    disconnect: 0,
  };
  return {
    calls,
    onConnect: (revision) => calls.connect.push(revision),
    onRevision: (revision) => calls.revision.push(revision),
    onDisconnect: () => {
      calls.disconnect += 1;
    },
  };
}

describe("startSharedRevisionStream", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("opens the stream only in the tab that takes the lock", () => {
    const bus = fakeBus();
    const locks = fakeLocks();
    streamMocks.startRevisionStream.mockReturnValue(() => undefined);
    const leader = handlers();
    const follower = handlers();
    const environment = { locks: locks.manager, createChannel: bus.create };

    startSharedRevisionStream(leader, environment);
    startSharedRevisionStream(follower, environment);

    expect(streamMocks.startRevisionStream).toHaveBeenCalledTimes(1);
  });

  it("hands the leader's revisions to the other tabs", () => {
    const bus = fakeBus();
    const locks = fakeLocks();
    let leaderHandlers: RevisionStreamHandlers | undefined;
    streamMocks.startRevisionStream.mockImplementation(
      (given: RevisionStreamHandlers) => {
        leaderHandlers = given;
        return () => undefined;
      },
    );
    const leader = handlers();
    const follower = handlers();
    const environment = { locks: locks.manager, createChannel: bus.create };

    startSharedRevisionStream(leader, environment);
    startSharedRevisionStream(follower, environment);
    // 参加した時点でリーダーがまだ繋がっていないので、切断として 1 回数えられている。
    const joined = follower.calls.disconnect;
    leaderHandlers?.onConnect(1n);
    leaderHandlers?.onRevision(2n);
    leaderHandlers?.onDisconnect();

    expect(leader.calls.connect).toEqual([1n]);
    expect(follower.calls.connect).toEqual([1n]);
    expect(follower.calls.revision).toEqual([2n]);
    expect(follower.calls.disconnect).toBe(joined + 1);
  });

  // 途中で開いたタブは、次の変更を待たずに今の接続状態を受け取る。
  it("answers a tab that joins after the leader connected", () => {
    const bus = fakeBus();
    const locks = fakeLocks();
    let leaderHandlers: RevisionStreamHandlers | undefined;
    streamMocks.startRevisionStream.mockImplementation(
      (given: RevisionStreamHandlers) => {
        leaderHandlers = given;
        return () => undefined;
      },
    );
    const leader = handlers();
    const environment = { locks: locks.manager, createChannel: bus.create };
    startSharedRevisionStream(leader, environment);
    leaderHandlers?.onConnect(5n);

    const latecomer = handlers();
    startSharedRevisionStream(latecomer, environment);

    expect(latecomer.calls.connect).toEqual([5n]);
  });

  it("takes over the stream when the leading tab stops", async () => {
    const bus = fakeBus();
    const locks = fakeLocks();
    streamMocks.startRevisionStream.mockReturnValue(() => undefined);
    const environment = { locks: locks.manager, createChannel: bus.create };
    const stopLeader = startSharedRevisionStream(handlers(), environment);
    startSharedRevisionStream(handlers(), environment);
    expect(streamMocks.startRevisionStream).toHaveBeenCalledTimes(1);

    stopLeader();
    await Promise.resolve();
    await Promise.resolve();

    expect(streamMocks.startRevisionStream).toHaveBeenCalledTimes(2);
  });

  // 既定の引数はブラウザの実装を見る。試験ではそこに差し替えたものが渡る。
  it("takes the lock and the channel from the browser when none are given", () => {
    const locks = fakeLocks();
    class StubChannel {
      onmessage: ((event: MessageEvent) => void) | null = null;
      postMessage() {
        return undefined;
      }
      close() {
        return undefined;
      }
    }
    vi.stubGlobal("BroadcastChannel", StubChannel);
    Object.defineProperty(navigator, "locks", {
      value: locks.manager,
      configurable: true,
    });
    streamMocks.startRevisionStream.mockReturnValue(() => undefined);

    const stop = startSharedRevisionStream(handlers());

    expect(streamMocks.startRevisionStream).toHaveBeenCalledTimes(1);
    stop();
    vi.unstubAllGlobals();
  });

  // ロックを取れない環境でも更新は止めない。そのタブが自分で張る。
  it("falls back to its own stream when the lock is refused", async () => {
    const bus = fakeBus();
    const request = () => Promise.reject(new Error("locks are unavailable"));
    const environment = {
      locks: { request } as unknown as LockManager,
      createChannel: bus.create,
    };
    streamMocks.startRevisionStream.mockReturnValue(() => undefined);

    startSharedRevisionStream(handlers(), environment);
    await Promise.resolve();

    expect(streamMocks.startRevisionStream).toHaveBeenCalledTimes(1);
  });

  it("opens its own stream where locks or channels are unavailable", () => {
    streamMocks.startRevisionStream.mockReturnValue(() => undefined);
    const own = handlers();

    startSharedRevisionStream(own, undefined);

    expect(streamMocks.startRevisionStream).toHaveBeenCalledTimes(1);
  });
});
