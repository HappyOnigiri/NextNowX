import { ClipboardCopy } from "lucide-react";
import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { getTaskPrompt } from "../api";
import { TaskPromptKind } from "../gen/prx/v1/prx_pb";
import { IconButton } from "./IconButton";
import { PromptDialogHead } from "./PromptDialogHead";
import { PromptPreview } from "./PromptPreview";
import { TabList, TabPanel } from "./TabList";
import { useCloseOnEscape } from "./useCloseOnEscape";
import { useDialogFocusTrap } from "./useDialogFocusTrap";
import { usePromptPreview } from "./usePromptPreview";

type TaskPromptTab = "design" | "implementation";

const taskPromptTabs: readonly TaskPromptTab[] = ["design", "implementation"];

const noticeKeys = {
  design: "inspector.promptWithPlan",
  implementation: "inspector.promptWithoutPlan",
} as const;

// TaskPromptDialog は渡す種類をユーザーに選ばせる。既定はサーバーの導出と
// 同じなので、これまでどおりのプロンプトはタブを触らずにコピーできる。
// docs/design/agent-prompts.md を参照。
export function TaskPromptDialog({
  taskId,
  hasImplementationPlan,
  onClose,
}: {
  taskId: string;
  hasImplementationPlan: boolean;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const [tab, setTab] = useState<TaskPromptTab>(
    hasImplementationPlan ? "implementation" : "design",
  );
  // コピーの結果は選んだ種類のもの。タブを移れば前の結果は用済みになる。
  const [copied, setCopied] = useState<{ tab: TaskPromptTab; ok: boolean }>();
  const closeRef = useRef<HTMLButtonElement>(null);
  const { dialogRef, onKeyDown } = useDialogFocusTrap<HTMLElement>(closeRef);

  useCloseOnEscape(onClose);

  // 本文はタブを切り替えるたびにサーバーへ求める。テンプレートの解決はサーバー
  // にあり、ブラウザの snapshot はすでに実態とずれている可能性があるため。
  const preview = usePromptPreview(
    `${taskId}\u0000${tab}`,
    () =>
      getTaskPrompt(
        taskId,
        tab === "design"
          ? TaskPromptKind.DESIGN
          : TaskPromptKind.IMPLEMENTATION,
      ).then((response) => response.prompt),
    t("inspector.promptFailed"),
  );

  async function copyPrompt() {
    if (preview.case !== "ready") return;
    try {
      await navigator.clipboard.writeText(preview.body);
    } catch {
      setCopied({ tab, ok: false });
      return;
    }
    setCopied({ tab, ok: true });
  }

  // 選んだ種類がタスクの状態に合わないときだけ注意を出す。ボタンは両方を
  // 提示するので、合わない組み合わせを選べること自体は誤りではない。
  const mismatch =
    (tab === "implementation") === hasImplementationPlan
      ? undefined
      : noticeKeys[tab];
  const outcome = copied?.tab === tab ? copied : undefined;

  return (
    <div className="scrim" role="presentation" onKeyDown={onKeyDown}>
      <section
        ref={dialogRef}
        className="dialog task-prompt-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="task-prompt-title"
      >
        <PromptDialogHead
          className="task-prompt-head"
          closeRef={closeRef}
          lead={t("inspector.promptDialogDescription", { task: taskId })}
          onClose={onClose}
          title={t("inspector.promptDialogTitle")}
          titleId="task-prompt-title"
        />
        <TabList
          tabs={taskPromptTabs.map((id) => ({
            id,
            label: t(`inspector.promptTab.${id}`),
          }))}
          active={tab}
          onSelect={setTab}
          idPrefix="task-prompt"
          className="settings-tabs"
          tabClassName="settings-tab"
          label={t("inspector.promptTabsLabel")}
        />
        <TabPanel
          active
          className="task-prompt-panel"
          idPrefix="task-prompt"
          tab={tab}
        >
          {mismatch && (
            <p className="task-prompt-notice" role="status">
              {t(mismatch)}
            </p>
          )}
          <PromptPreview
            label={t("inspector.promptPreview")}
            body={preview.case === "ready" ? preview.body : ""}
            busy={preview.case === "loading"}
            error={preview.case === "failed" ? preview.message : undefined}
          />
        </TabPanel>
        <footer>
          <p className="task-prompt-status" aria-live="polite">
            {outcome?.ok === true && t(`inspector.${tab}PromptCopied`)}
            {outcome?.ok === false && t("inspector.promptFailed")}
          </p>
          <IconButton
            icon={ClipboardCopy}
            label={t("inspector.copyPromptAction")}
            variant="primary"
            disabled={preview.case !== "ready"}
            onClick={() => void copyPrompt()}
          />
        </footer>
      </section>
    </div>
  );
}
