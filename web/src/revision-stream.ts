import { watchRevision } from "./api";

// heartbeat 3 回分を待って何も届かなければ、接続は生きていても死んでいる。
export const receiveTimeoutMs = 45_000;
export const initialBackoffMs = 1_000;
export const maxBackoffMs = 30_000;

export interface RevisionStreamHandlers {
  // onConnect は 1 通目を受け取るたびに呼ばれる。切断中の変更を必ず拾うため、
  // 呼び出し側はここで無条件に再取得する。
  onConnect: (revision: bigint) => void;
  // onRevision はリビジョンが実際に変わったときだけ呼ばれる。同じ値の再送は
  // heartbeat なので、生存確認のたびに再取得しない。
  onRevision: (revision: bigint) => void;
  // onDisconnect は接続が切れるたびに呼ばれる。表示は呼び出し側が遅らせる。
  onDisconnect: () => void;
}

// backoffFor は再接続までの待ち時間を返す。jitter は、同時に落ちた複数のタブが
// 揃って再接続しないようにする。
export function backoffFor(attempt: number, random = Math.random): number {
  const capped = Math.min(initialBackoffMs * 2 ** attempt, maxBackoffMs);
  return Math.round(capped * (0.5 + random() * 0.5));
}

// startRevisionStream は購読を開き、切れたら backoff をおいて張り直す。返り値を
// 呼ぶと購読を終える。1 タブにつき 1 本だけ開く。
export function startRevisionStream(
  handlers: RevisionStreamHandlers,
): () => void {
  // 停止の合図は後片付けから書き、ループは await をまたいで読む。
  const session = {
    stopped: false,
    attempt: 0,
    timer: undefined as ReturnType<typeof setTimeout> | undefined,
    connection: undefined as AbortController | undefined,
  };

  // 関数越しに読む。await をまたいでも停止フラグが false に絞り込まれないようにする。
  const stopped = () => session.stopped;

  const wait = (delay: number) =>
    new Promise<void>((resolve) => {
      session.timer = setTimeout(resolve, delay);
    });

  async function loop() {
    while (!stopped()) {
      const connection = new AbortController();
      session.connection = connection;
      try {
        await consume(connection, handlers, () => {
          session.attempt = 0;
        });
      } catch {
        // 切断の理由は再接続の判断を変えない。恒久失敗でも
        // refetchOnWindowFocus による従来の再取得が残る。
      }
      connection.abort();
      if (stopped()) return;
      handlers.onDisconnect();
      await wait(backoffFor(session.attempt));
      session.attempt += 1;
    }
  }

  void loop();
  return () => {
    session.stopped = true;
    if (session.timer !== undefined) clearTimeout(session.timer);
    session.connection?.abort();
  };
}

// consume は 1 本の購読を受信タイムアウトつきで読み切る。heartbeat が途絶えたら
// abort して呼び出し側に張り直させる。
async function consume(
  connection: AbortController,
  handlers: RevisionStreamHandlers,
  onProgress: () => void,
): Promise<void> {
  let seen: bigint | undefined;
  let deadline = watchdog(connection);
  try {
    for await (const message of watchRevision(connection.signal)) {
      clearTimeout(deadline);
      onProgress();
      if (seen === undefined) handlers.onConnect(message.revision);
      else if (message.revision !== seen) handlers.onRevision(message.revision);
      seen = message.revision;
      deadline = watchdog(connection);
    }
  } finally {
    clearTimeout(deadline);
  }
}

function watchdog(connection: AbortController) {
  return setTimeout(() => {
    connection.abort();
  }, receiveTimeoutMs);
}
