// 再描画を伴う取り直しは、進行中のポインタジェスチャを壊す。依存をつなぐドラッグの
// 最中にグラフが描き直されると、掴んでいた接続がその場で消える。
// 取り直し自体は捨てず、ジェスチャが終わってから 1 回だけ走らせる。

let holds = 0;
let deferred: (() => void) | undefined;

// holdRefresh は取り直しを保留させる。返した関数を呼ぶと保留を解く。解除は冪等。
export function holdRefresh(): () => void {
  holds += 1;
  let released = false;
  return () => {
    if (released) return;
    released = true;
    holds -= 1;
    if (holds > 0) return;
    const run = deferred;
    deferred = undefined;
    run?.();
  };
}

// runWhenIdle は保留がなければ即座に、あれば解除後に 1 回だけ実行する。保留中に
// 何度呼ばれても最後の 1 回に畳まれる。取り直しは冪等な操作だからである。
export function runWhenIdle(run: () => void): void {
  if (holds === 0) {
    run();
    return;
  }
  deferred = run;
}
