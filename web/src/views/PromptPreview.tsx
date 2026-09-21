// PromptPreview はコピー対象の本文をそのまま見せる。サーバーが描いたテキストを
// 読み取り専用で置き、失敗はサーバーの文言のまま伝える。
// docs/design/webui.md を参照。
export function PromptPreview({
  label,
  body,
  busy,
  error,
  placeholder,
}: {
  label: string;
  body: string;
  busy: boolean;
  error: string | undefined;
  // placeholder は本文をまだ求められない状態、たとえばタスクが未選択のとき。
  placeholder?: string | undefined;
}) {
  if (error)
    return (
      <div className="prompt-preview">
        <p className="form-error" role="alert">
          {error}
        </p>
      </div>
    );
  if (placeholder !== undefined && body === "" && !busy)
    return (
      <div className="prompt-preview">
        <p className="prompt-preview-placeholder">{placeholder}</p>
      </div>
    );
  return (
    <div className="prompt-preview">
      {/* 処理中はラベルの語を変えず aria-busy だけで伝える。読み手が同じ
          領域を目で追い直さずに済む。 */}
      <textarea
        aria-busy={busy || undefined}
        aria-label={label}
        className="prompt-preview-body"
        readOnly
        rows={12}
        value={body}
      />
    </div>
  );
}
