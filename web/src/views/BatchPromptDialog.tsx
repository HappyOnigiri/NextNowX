import { ClipboardCopy, Square, SquareCheckBig } from "lucide-react";
import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { getBatchPrompt } from "../api";
import { TaskPromptKind, type Task } from "../gen/nnx/v1/nnx_pb";
import { isSelectable, type BatchPromptKind } from "./batchPromptTasks";
import { IconButton } from "./IconButton";
import { PromptDialogHead } from "./PromptDialogHead";
import { PromptPreview } from "./PromptPreview";
import { TabList, TabPanel } from "./TabList";
import { useBatchSelection, type BatchSelection } from "./useBatchSelection";
import { useCloseOnEscape } from "./useCloseOnEscape";
import { useDialogFocusTrap } from "./useDialogFocusTrap";
import { usePromptPreview } from "./usePromptPreview";

const batchTabs: readonly BatchPromptKind[] = ["design", "implementation"];

// 選択のたびにサーバーへ求めず、連続したクリックが落ち着いてから 1 回描く。
const previewDelayMilliseconds = 250;

// BatchPromptDialog は複数の task を 1 つのプロンプトにまとめて渡す。タブが
// 設計と実装のどちらを回すかを決め、それが候補のベース集合とテンプレートの
// 両方を選ぶ。docs/design/agent-prompts.md を参照。
export function BatchPromptDialog({
  featureId,
  tasks,
  onClose,
}: {
  featureId: string;
  tasks: Task[];
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const selection = useBatchSelection(tasks);
  const { kind, targets } = selection;
  // コピーの結果はそのとき選んでいた task のもの。選択が変われば用済みになる。
  const [copied, setCopied] = useState<{ key: string; count?: number }>();
  const closeRef = useRef<HTMLButtonElement>(null);
  const { dialogRef, onKeyDown } = useDialogFocusTrap<HTMLElement>(closeRef);

  useCloseOnEscape(onClose);

  const key =
    targets.length === 0 ? "" : `${kind}\u0000${targets.join("\u0000")}`;
  const preview = usePromptPreview(
    key,
    () =>
      getBatchPrompt(
        featureId,
        targets,
        kind === "design"
          ? TaskPromptKind.DESIGN
          : TaskPromptKind.IMPLEMENTATION,
      ).then((response) => response.prompt),
    t("batchPrompt.failed"),
    previewDelayMilliseconds,
  );

  async function copyPrompt() {
    if (preview.case !== "ready") return;
    try {
      await navigator.clipboard.writeText(preview.body);
    } catch {
      setCopied({ key });
      return;
    }
    setCopied({ key, count: targets.length });
  }

  const outcome = copied?.key === key ? copied : undefined;

  return (
    <div className="scrim" role="presentation" onKeyDown={onKeyDown}>
      <section
        ref={dialogRef}
        className="dialog batch-prompt-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="batch-prompt-title"
      >
        <PromptDialogHead
          className="batch-prompt-head"
          closeRef={closeRef}
          lead={t(`batchPrompt.${kind}Description`)}
          onClose={onClose}
          title={t("batchPrompt.title")}
          titleId="batch-prompt-title"
        />
        <TabList
          tabs={batchTabs.map((id) => ({
            id,
            label: t(`batchPrompt.tab.${id}`),
          }))}
          active={kind}
          onSelect={selection.changeKind}
          idPrefix="batch-prompt"
          className="settings-tabs"
          tabClassName="settings-tab"
          label={t("batchPrompt.tabsLabel")}
        />
        <TabPanel
          active
          className="batch-prompt-panel"
          idPrefix="batch-prompt"
          tab={kind}
        >
          <BatchPromptTaskList selection={selection} />
          <PromptPreview
            label={t("batchPrompt.preview")}
            body={preview.case === "ready" ? preview.body : ""}
            busy={preview.case === "loading"}
            error={preview.case === "failed" ? preview.message : undefined}
            placeholder={t("batchPrompt.previewEmpty")}
          />
        </TabPanel>
        <footer>
          <p className="batch-prompt-status" aria-live="polite">
            {outcome?.count !== undefined &&
              t("batchPrompt.copied", { selected: outcome.count })}
            {outcome?.count === undefined &&
              outcome !== undefined &&
              t("batchPrompt.failed")}
          </p>
          <IconButton
            icon={ClipboardCopy}
            label={t("batchPrompt.copy")}
            variant="primary"
            disabled={preview.case !== "ready"}
            onClick={() => void copyPrompt()}
          />
        </footer>
      </section>
    </div>
  );
}

function BatchPromptTaskList({ selection }: { selection: BatchSelection }) {
  const { t } = useTranslation();
  const { candidates, kind, selected } = selection;
  // 候補がないときもトグルは出す。表示を広げれば候補が現れることがあり、
  // リストごと畳むとその手立てまで隠れてしまう。
  return (
    <div className="batch-prompt-body">
      <div className="batch-prompt-toolbar">
        <IconButton
          icon={selection.allSelected ? Square : SquareCheckBig}
          label={
            selection.allSelected
              ? t("batchPrompt.clearAll")
              : t("batchPrompt.selectAll")
          }
          variant="quiet"
          size="compact"
          disabled={candidates.length === 0}
          onClick={selection.toggleAll}
        />
        {/* 表示範囲の切り替えは行の操作ではなくチェックボックスにする。行が持つ
            選択に加わるのではなく、リスト全体の表示を切り替えるため。 */}
        {kind === "design" ? (
          <BatchPromptToggle
            checked={selection.includeDesigned}
            label={t("batchPrompt.includeDesigned")}
            onChange={selection.changeIncludeDesigned}
          />
        ) : (
          <BatchPromptToggle
            checked={selection.includeUndesigned}
            label={t("batchPrompt.includeUndesigned")}
            onChange={selection.changeIncludeUndesigned}
          />
        )}
        <BatchPromptToggle
          checked={selection.includeBlocked}
          label={t("batchPrompt.includeBlocked")}
          onChange={selection.changeIncludeBlocked}
        />
        <span className="batch-prompt-count">
          {t("batchPrompt.selectedCount", {
            selected: selected.size,
            total: candidates.length,
          })}
        </span>
      </div>
      {/* 実装計画がないタスクを候補に加えている間だけ、実装プロンプトが
          まだない計画を読ませることを伝える。選べること自体は誤りではない
          ので操作は止めない。 */}
      {kind === "implementation" && selection.includeUndesigned && (
        <p className="batch-prompt-notice" role="status">
          {t("batchPrompt.undesignedNotice")}
        </p>
      )}
      {candidates.length === 0 ? (
        <p className="batch-prompt-empty">{t(`batchPrompt.${kind}Empty`)}</p>
      ) : (
        <ul className="batch-prompt-list">
          {candidates.map((candidate) => (
            <li key={candidate.task.id}>
              {/* 行そのものが操作要素で、見た目ではアクセント枠と塗りが示す選択
                  を aria-pressed が伝える。docs/design/webui.md を参照。 */}
              <button
                type="button"
                className="batch-prompt-task"
                aria-pressed={selected.has(candidate.task.id)}
                // ブロッカーを一緒に渡さない task は着手の土台がないので、
                // それらが選ばれるまで選択できない。
                disabled={!isSelectable(candidate, selected)}
                onClick={() => {
                  selection.toggle(candidate.task.id);
                }}
              >
                <span className="batch-prompt-task-title">
                  {candidate.task.title}
                </span>
                {candidate.pendingBlockerIds.length > 0 && (
                  <span className="batch-prompt-task-after">
                    {t("batchPrompt.afterTasks", {
                      tasks: candidate.pendingBlockerIds.join(", "),
                    })}
                  </span>
                )}
                <span className="batch-prompt-task-id">
                  {candidate.task.id}
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

function BatchPromptToggle({
  checked,
  label,
  onChange,
}: {
  checked: boolean;
  label: string;
  onChange: (checked: boolean) => void;
}) {
  return (
    <label className="batch-prompt-include-blocked">
      <input
        type="checkbox"
        checked={checked}
        onChange={(event) => {
          onChange(event.target.checked);
        }}
      />
      {label}
    </label>
  );
}
