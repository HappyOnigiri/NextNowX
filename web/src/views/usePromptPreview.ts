import { useEffect, useRef, useState } from "react";

export type PromptPreviewState =
  | { case: "empty" }
  | { case: "loading" }
  | { case: "ready"; body: string }
  | { case: "failed"; message: string };

// usePromptPreview は key が変わるたびに本文を取り直す。結果は key と一緒に
// 覚え、key が進んでいる間は loading として読む。効果の中で同期的に状態を
// 書くと連鎖レンダリングになるためである。空の key は要求そのものがない状態。
export function usePromptPreview(
  key: string,
  render: () => Promise<string>,
  fallbackMessage: string,
  // delayMilliseconds は連続した操作が落ち着くのを待つ。0 なら即座に求める。
  delayMilliseconds = 0,
): PromptPreviewState {
  const [result, setResult] = useState<{
    key: string;
    state: PromptPreviewState;
  }>();
  const latest = useRef({ render, fallbackMessage });
  useEffect(() => {
    latest.current = { render, fallbackMessage };
  });

  useEffect(() => {
    if (key === "") return undefined;
    let cancelled = false;
    const request = () => {
      latest.current
        .render()
        .then((body) => {
          if (!cancelled) setResult({ key, state: { case: "ready", body } });
        })
        .catch((cause: unknown) => {
          if (cancelled) return;
          // サーバーのメッセージは原因となったタスクやテンプレートを名指しする
          // ので、そのまま見せる。
          setResult({
            key,
            state: {
              case: "failed",
              message:
                cause instanceof Error
                  ? cause.message
                  : latest.current.fallbackMessage,
            },
          });
        });
    };
    const timer = window.setTimeout(request, delayMilliseconds);
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [key, delayMilliseconds]);

  if (key === "") return { case: "empty" };
  return result?.key === key ? result.state : { case: "loading" };
}
