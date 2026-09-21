import { ClipboardCopy } from "lucide-react";
import { useState } from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import { IconButton } from "./IconButton";
import { TaskPromptDialog } from "./TaskPromptDialog";

// TaskPromptCopyButton はタスクを別のエージェントに渡す入口。どのプロンプトを
// 渡すかはクリックした人が選ぶので、ボタンは種類を決めずダイアログを開く。
// docs/design/agent-prompts.md を参照。
export function TaskPromptCopyButton({
  taskId,
  hasImplementationPlan,
  size = "standard",
}: {
  taskId: string;
  hasImplementationPlan: boolean;
  size?: "standard" | "compact";
}) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);

  return (
    <span className="task-prompt-copy">
      <IconButton
        icon={ClipboardCopy}
        label={t("inspector.copyPrompt")}
        variant="secondary"
        size={size}
        iconOnly
        className="task-prompt-copy-button"
        type="button"
        onClick={() => {
          setOpen(true);
        }}
      />
      {/* ダイアログは body へ出す。グラフのノードは transform を持ち、その中の
          スクリムは fixed でもノードの矩形に閉じ込められてしまうため。 */}
      {open &&
        createPortal(
          <TaskPromptDialog
            taskId={taskId}
            hasImplementationPlan={hasImplementationPlan}
            onClose={() => {
              setOpen(false);
            }}
          />,
          document.body,
        )}
    </span>
  );
}
