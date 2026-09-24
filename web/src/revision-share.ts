import {
  startRevisionStream,
  type RevisionStreamHandlers,
} from "./revision-stream";

// サーバーは TLS なしの HTTP/1.1 なので、ブラウザは同一オリジンへの同時接続を
// 6 本前後に制限する。タブごとに 1 本張ると、数枚開いただけで取得が接続待ちになる。
// そこでプロファイル全体で 1 本に絞り、受け取ったリビジョンをタブ間で配る。
const revisionLockName = "nnx-revision-stream";
const revisionChannelName = "nnx-revision-stream";

type ShareMessage =
  | { kind: "connect"; revision: string }
  | { kind: "revision"; revision: string }
  | { kind: "disconnect" }
  // ask と state は、開いたばかりのタブが現在の接続状態を知るための往復である。
  // 宛先を id で絞らないと、他のフォロワーまで接続直後として取り直してしまう。
  | { kind: "ask"; id: string }
  | { kind: "state"; id: string; revision: string | null };

interface ShareEnvironment {
  locks: LockManager;
  createChannel: (name: string) => BroadcastChannel;
}

// 型の上ではどちらも常にあることになっているが、Web Locks は安全なコンテキストに
// 限られ、古いブラウザには無い。実行時に確かめて、無ければタブごとに張る。
function sharedEnvironment(): ShareEnvironment | undefined {
  const locks = (navigator as { locks?: LockManager }).locks;
  const channel = (globalThis as { BroadcastChannel?: typeof BroadcastChannel })
    .BroadcastChannel;
  if (!locks || !channel) return undefined;
  return { locks, createChannel: (name) => new channel(name) };
}

// startSharedRevisionStream は購読を開く。ストリームを実際に張るのはロックを取れた
// 1 タブだけで、他のタブはその通知を受け取る。返り値を呼ぶと購読を終える。
// ロックか BroadcastChannel が無い環境では、そのタブが自分で張る。
export function startSharedRevisionStream(
  handlers: RevisionStreamHandlers,
  environment = sharedEnvironment(),
): () => void {
  if (!environment) return startRevisionStream(handlers);
  const { locks, createChannel } = environment;
  const channel = createChannel(revisionChannelName);
  const id = Math.random().toString(36).slice(2);
  const session = {
    stopped: false,
    leading: false,
    // 最後に配ったリビジョン。リーダーが遅れて参加したタブへ返すために持つ。
    revision: null as bigint | null,
    stop: undefined as (() => void) | undefined,
  };

  channel.onmessage = (event: MessageEvent<ShareMessage>) => {
    const message = event.data;
    if (session.leading) {
      if (message.kind !== "ask") return;
      channel.postMessage({
        kind: "state",
        id: message.id,
        revision: session.revision?.toString() ?? null,
      } satisfies ShareMessage);
      return;
    }
    follow(message);
  };

  function follow(message: ShareMessage) {
    switch (message.kind) {
      case "connect":
        handlers.onConnect(BigInt(message.revision));
        return;
      case "revision":
        handlers.onRevision(BigInt(message.revision));
        return;
      case "disconnect":
        handlers.onDisconnect();
        return;
      case "ask":
        return;
      case "state":
        if (message.id !== id) return;
        // リーダーが繋がっていれば接続直後と同じ扱いにする。無条件の取り直しは
        // 開いたばかりのタブでは空のキャッシュを捨てるだけで、余計な取得を生まない。
        if (message.revision === null) handlers.onDisconnect();
        else handlers.onConnect(BigInt(message.revision));
        return;
    }
  }

  function lead(release: () => void) {
    if (session.stopped) {
      release();
      return;
    }
    session.leading = true;
    const share = (message: ShareMessage) => {
      channel.postMessage(message);
    };
    const stop = startRevisionStream({
      onConnect: (revision) => {
        session.revision = revision;
        share({ kind: "connect", revision: revision.toString() });
        handlers.onConnect(revision);
      },
      onRevision: (revision) => {
        session.revision = revision;
        share({ kind: "revision", revision: revision.toString() });
        handlers.onRevision(revision);
      },
      onDisconnect: () => {
        session.revision = null;
        share({ kind: "disconnect" });
        handlers.onDisconnect();
      },
    });
    session.stop = () => {
      stop();
      release();
    };
  }

  // ロックは待ち行列になる。リーダーのタブが閉じれば次のタブが引き継ぎ、
  // そこで張り直した接続の 1 通目が切断中の変更を回収する。
  locks
    .request(
      revisionLockName,
      () =>
        new Promise<void>((resolve) => {
          lead(resolve);
        }),
    )
    .catch(() => {
      if (session.stopped || session.leading) return;
      session.stop = startRevisionStream(handlers);
    });
  channel.postMessage({ kind: "ask", id } satisfies ShareMessage);

  return () => {
    session.stopped = true;
    session.stop?.();
    session.stop = undefined;
    channel.onmessage = null;
    channel.close();
  };
}
