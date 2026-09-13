import { useState } from "react";
import { mutations } from "../api";
import type { Document } from "../gen/prx/v1/prx_pb";
import { useDomainMutation } from "../hooks";
import { DocumentDialog } from "./DocumentDialog";
import type { TaskNodeDocument } from "./TaskNode";

interface EditTarget {
  document: Document;
  content: string;
  trigger: HTMLElement | null;
}

// 資料の編集は追加と同じダイアログで行う。本文は一覧に載っていないので、
// 開く前に GetDocument で取りに行く。親は応答の document が持っている。
export function useDocumentEditing() {
  const getDocument = useDomainMutation(mutations.getDocument);
  const [target, setTarget] = useState<EditTarget | null>(null);

  const dialog = target ? (
    <DocumentDialog
      mode="edit"
      document={target.document}
      content={target.content}
      trigger={target.trigger}
      onClose={() => {
        setTarget(null);
      }}
    />
  ) : null;

  return {
    request: (document: TaskNodeDocument, trigger: HTMLElement | null) => {
      getDocument.mutate(document.id, {
        onSuccess: (response) => {
          if (!response.document) return;
          setTarget({
            document: response.document,
            content: response.content,
            trigger,
          });
        },
      });
    },
    dialog,
    error: getDocument.error,
  };
}
