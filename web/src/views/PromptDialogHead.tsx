import { X } from "lucide-react";
import type { Ref } from "react";
import { useTranslation } from "react-i18next";
import { IconButton } from "./IconButton";

// PromptDialogHead は 2 つのプロンプトダイアログで同じ見出しと閉じる操作を出す。
export function PromptDialogHead({
  className,
  closeRef,
  lead,
  onClose,
  title,
  titleId,
}: {
  className: string;
  closeRef: Ref<HTMLButtonElement>;
  lead: string;
  onClose: () => void;
  title: string;
  titleId: string;
}) {
  const { t } = useTranslation();
  return (
    <header className={className}>
      <div>
        <h2 id={titleId}>{title}</h2>
        <p className="dialog-lead">{lead}</p>
      </div>
      <IconButton
        ref={closeRef}
        icon={X}
        label={t("common.close")}
        variant="secondary"
        iconOnly
        onClick={onClose}
      />
    </header>
  );
}
