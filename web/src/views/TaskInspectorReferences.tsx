import { Pencil, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { DocumentKind } from "../gen/nnx/v1/nnx_pb";
import { documentKindLabel } from "../i18n/domain";
import { IconButton } from "./IconButton";
import { MutationError } from "./MutationError";
import { type TaskNodeDocument } from "./TaskNode";
import { useDocumentDeletion } from "./useDocumentDeletion";
import { useDocumentEditing } from "./useDocumentEditing";

interface ReferencesSectionProps {
  documents: TaskNodeDocument[];
  onPreview: (document: TaskNodeDocument) => void;
  readOnly?: boolean;
  compact?: boolean;
}

export function ReferencesSection({
  documents,
  onPreview,
  readOnly = false,
  compact = false,
}: ReferencesSectionProps) {
  const { t } = useTranslation();
  const deletion = useDocumentDeletion();
  const editing = useDocumentEditing();
  const ordered = [...documents].sort(
    (left, right) =>
      Number(right.isImplementationPlan) - Number(left.isImplementationPlan),
  );

  return (
    <section
      className={compact ? "documents-section is-compact" : "documents-section"}
    >
      <h3>{t("inspector.references")}</h3>
      {ordered.map((document) => (
        <DocumentRow
          key={document.id}
          document={document}
          onPreview={onPreview}
          onDelete={deletion.request}
          onEdit={editing.request}
          canEdit={!readOnly}
          canDelete={!readOnly}
        />
      ))}
      {documents.length === 0 && (
        <p className="read-only-empty">{t("inspector.noReferences")}</p>
      )}
      {!readOnly && <MutationError error={editing.error} />}
      {editing.dialog}
      {deletion.dialog}
    </section>
  );
}

export function DocumentRow({
  document,
  onPreview,
  onDelete,
  onEdit,
  canEdit,
  canDelete,
}: {
  document: TaskNodeDocument;
  onPreview: (document: TaskNodeDocument) => void;
  onDelete?: (document: TaskNodeDocument) => void;
  onEdit: (document: TaskNodeDocument, trigger: HTMLElement | null) => void;
  canEdit: boolean;
  canDelete: boolean;
}) {
  const { t } = useTranslation();
  const title = document.title || documentKindLabel(document.kind, t);
  return (
    <div
      className={`document-chip ${document.isImplementationPlan ? "is-plan" : ""}`}
    >
      {document.kind === DocumentKind.URL ? (
        <a href={document.locator} target="_blank" rel="noreferrer">
          <DocumentTitle title={title} plan={document.isImplementationPlan} />
          <small>{document.locator}</small>
        </a>
      ) : (
        <button
          type="button"
          className="document-preview"
          onClick={() => {
            onPreview(document);
          }}
        >
          <span className="document-preview-copy">
            <DocumentTitle title={title} plan={document.isImplementationPlan} />
            <small>
              {document.locator || documentKindLabel(document.kind, t)}
            </small>
          </span>
        </button>
      )}
      {(canEdit || canDelete) && (
        <>
          {canEdit && (
            <IconButton
              icon={Pencil}
              label={t("inspector.editReference", { title })}
              variant="secondary"
              size="compact"
              iconOnly
              onClick={(event) => {
                onEdit(document, event.currentTarget);
              }}
            />
          )}
          {canDelete && (
            <IconButton
              icon={Trash2}
              label={t("inspector.deleteReference", {
                title: document.title || t("inspector.referenceFallback"),
              })}
              variant="danger"
              size="compact"
              iconOnly
              onClick={() => {
                onDelete?.(document);
              }}
            />
          )}
        </>
      )}
    </div>
  );
}

function DocumentTitle({ title, plan }: { title: string; plan: boolean }) {
  return (
    <b>
      {plan && <span className="document-plan-label">PLAN</span>}
      {title}
    </b>
  );
}
