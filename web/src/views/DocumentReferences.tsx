import { ChevronDown, Plus } from "lucide-react";
import { useEffect, useId, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import type { DocumentParent } from "../document-parent";
import { DocumentDialog } from "./DocumentDialog";
import { IconButton } from "./IconButton";
import { MutationError } from "./MutationError";
import { DocumentRow } from "./TaskInspectorReferences";
import type { TaskNodeDocument } from "./TaskNode";
import { useDocumentDeletion } from "./useDocumentDeletion";
import { useDocumentEditing } from "./useDocumentEditing";

// パネルと編集の流れはドキュメントの所属先に依存しないので、親はそのまま
// 追加ダイアログへ渡す。
interface DocumentReferencesProps {
  parent: DocumentParent;
  documents: TaskNodeDocument[];
  onPreview: (document: TaskNodeDocument) => void;
  readOnly?: boolean;
}

export function DocumentReferences({
  parent,
  documents,
  onPreview,
  readOnly = false,
}: DocumentReferencesProps) {
  const { t } = useTranslation();
  const deletion = useDocumentDeletion();
  const editing = useDocumentEditing();
  const [open, setOpen] = useState(false);
  const [showAddDialog, setShowAddDialog] = useState(false);
  const [addTrigger, setAddTrigger] = useState<HTMLElement | null>(null);
  const rootRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const triggerId = useId();
  const panelId = useId();

  useEffect(() => {
    if (!open) return;
    const closeOnOutsidePointer = (event: PointerEvent) => {
      if (
        event.target instanceof Node &&
        !rootRef.current?.contains(event.target)
      )
        setOpen(false);
    };
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return;
      setOpen(false);
      triggerRef.current?.focus();
    };
    document.addEventListener("pointerdown", closeOnOutsidePointer);
    document.addEventListener("keydown", closeOnEscape);
    return () => {
      document.removeEventListener("pointerdown", closeOnOutsidePointer);
      document.removeEventListener("keydown", closeOnEscape);
    };
  }, [open]);

  const ordered = [...documents].sort(
    (left, right) =>
      Number(right.isImplementationPlan) - Number(left.isImplementationPlan),
  );

  return (
    <div className="document-references" ref={rootRef}>
      <IconButton
        ref={triggerRef}
        id={triggerId}
        icon={ChevronDown}
        label={t("workspace.references")}
        className="document-references-trigger"
        variant="secondary"
        aria-expanded={open}
        aria-controls={panelId}
        onClick={() => {
          setOpen((current) => !current);
        }}
      />
      {open && (
        <DocumentReferencesPanel
          id={panelId}
          labelledBy={triggerId}
          documents={ordered}
          readOnly={readOnly}
          onPreview={(selected) => {
            setOpen(false);
            onPreview(selected);
          }}
          onDelete={(target) => {
            // 確認ダイアログはパネルの外に出す。パネルを開いたままにすると、
            // 外側の pointerdown で閉じたときにダイアログごと消える。
            setOpen(false);
            deletion.request(target);
          }}
          onEdit={(target) => {
            // 編集ダイアログも同じ理由でパネルの外に出す。鉛筆ボタンは閉じた
            // 時点で外れるので、フォーカスの戻り先はパネルの trigger にする。
            setOpen(false);
            editing.request(target, triggerRef.current);
          }}
          onAdd={() => {
            setOpen(false);
            setAddTrigger(triggerRef.current);
            setShowAddDialog(true);
          }}
        />
      )}
      {showAddDialog && (
        <DocumentDialog
          mode="add"
          {...parent}
          trigger={addTrigger}
          onClose={() => {
            setShowAddDialog(false);
          }}
        />
      )}
      {/* 本文の読み出しに失敗したときはパネルを閉じているので、外へ出す。 */}
      <MutationError error={editing.error} />
      {editing.dialog}
      {deletion.dialog}
    </div>
  );
}

function DocumentReferencesPanel({
  id,
  labelledBy,
  documents,
  readOnly,
  onPreview,
  onDelete,
  onEdit,
  onAdd,
}: {
  id: string;
  labelledBy: string;
  documents: TaskNodeDocument[];
  readOnly: boolean;
  onPreview: (document: TaskNodeDocument) => void;
  onDelete: (document: TaskNodeDocument) => void;
  onEdit: (document: TaskNodeDocument) => void;
  onAdd: () => void;
}) {
  const { t } = useTranslation();
  return (
    <section
      id={id}
      className="document-references-panel"
      aria-labelledby={labelledBy}
    >
      <div className="document-references-list">
        {documents.map((document) => (
          <DocumentRow
            key={document.id}
            document={document}
            onPreview={onPreview}
            onDelete={onDelete}
            onEdit={onEdit}
            canEdit={!readOnly}
            canDelete={!readOnly}
          />
        ))}
        {documents.length === 0 && (
          <p className="read-only-empty">{t("inspector.noReferences")}</p>
        )}
      </div>
      {!readOnly && (
        <div className="document-references-footer">
          <IconButton
            icon={Plus}
            label={t("workspace.addReference")}
            variant="quiet"
            onClick={onAdd}
          />
        </div>
      )}
    </section>
  );
}
