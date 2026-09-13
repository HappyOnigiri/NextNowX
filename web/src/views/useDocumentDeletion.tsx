import { useState } from "react";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import { useDomainMutation } from "../hooks";
import { ConfirmationDialog } from "./ConfirmationDialog";
import type { TaskNodeDocument } from "./TaskNode";

// 資料の削除は確認を挟む。タスク・feature・project のどのパネルからでも同じ
// 文言と同じ mutation を使うため、状態とダイアログをここへまとめる。
export function useDocumentDeletion() {
  const { t } = useTranslation();
  const deleteDocument = useDomainMutation(mutations.deleteDocument);
  const [target, setTarget] = useState<TaskNodeDocument | null>(null);

  const dialog = target ? (
    <ConfirmationDialog
      title={t("inspector.deleteReferenceTitle", {
        title: target.title || t("inspector.referenceFallback"),
      })}
      description={t("inspector.deleteReferenceDescription")}
      confirmLabel={t("inspector.confirmDeleteReference")}
      danger
      pending={deleteDocument.isPending}
      error={deleteDocument.error}
      onCancel={() => {
        setTarget(null);
      }}
      onConfirm={() => {
        deleteDocument.mutate(target.id, {
          onSuccess: () => {
            setTarget(null);
          },
        });
      }}
    />
  ) : null;

  return {
    request: (document: TaskNodeDocument) => {
      setTarget(document);
    },
    dialog,
  };
}
