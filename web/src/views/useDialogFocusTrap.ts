import { useEffect, useRef, type KeyboardEvent, type RefObject } from "react";

// 焦点を持てる要素。roving tabindex のタブは -1 を持つので、属性ではなく
// 実際の tabIndex で外す。そうしないと Tab が非選択のタブへ入り込む。
const focusableSelector = [
  "button:not(:disabled)",
  "input:not(:disabled)",
  "select:not(:disabled)",
  "textarea:not(:disabled)",
  "a[href]",
  "[tabindex]",
].join(", ");

// useDialogFocusTrap はモーダルの中に Tab を閉じ込め、閉じたら開く前の位置へ
// 焦点を戻す。docs/design/webui.md を参照。
export function useDialogFocusTrap<T extends HTMLElement>(
  initialFocus?: RefObject<HTMLElement | null>,
): {
  dialogRef: RefObject<T | null>;
  onKeyDown: (event: KeyboardEvent) => void;
} {
  const dialogRef = useRef<T>(null);

  useEffect(() => {
    const origin = document.activeElement;
    initialFocus?.current?.focus();
    return () => {
      // 閉じる操作の後始末が終わってから戻す。先に戻すと、消える途中の
      // モーダルが焦点を奪い返すことがある。
      window.setTimeout(() => {
        if (origin instanceof HTMLElement && origin.isConnected) origin.focus();
      });
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps -- 初回の焦点移動と復帰だけを行う。
  }, []);

  function onKeyDown(event: KeyboardEvent) {
    if (event.key !== "Tab") return;
    const focusable = Array.from(
      dialogRef.current?.querySelectorAll<HTMLElement>(focusableSelector) ?? [],
    ).filter((element) => element.tabIndex >= 0);
    const first = focusable[0];
    const last = focusable.at(-1);
    if (!first || !last) {
      event.preventDefault();
      return;
    }
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  }

  return { dialogRef, onKeyDown };
}
