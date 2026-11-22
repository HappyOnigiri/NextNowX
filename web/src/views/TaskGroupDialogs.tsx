import { Trash2, X } from "lucide-react";
import {
  useEffect,
  useId,
  useRef,
  useState,
  type Ref,
  type SyntheticEvent,
} from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import type { TaskGroup } from "../gen/nnx/v1/nnx_pb";
import { useDomainMutation } from "../hooks";
import { ConfirmationDialog } from "./ConfirmationDialog";
import type { GroupAction } from "./GraphTaskGroups";
import { IconButton } from "./IconButton";
import { MarkdownPreview } from "./MarkdownPreview";
import { MutationError } from "./MutationError";
import { DocumentRow } from "./TaskInspectorReferences";
import type { TaskNodeDocument } from "./TaskNode";
import { DiscardChangesDialog, SaveButton } from "./UnsavedChanges";
import { useCloseOnEscape } from "./useCloseOnEscape";
import { useDialogFocusTrap } from "./useDialogFocusTrap";

export function TaskGroupDialog({
  featureId,
  groups,
  group,
  onSaved,
  onClose,
}: {
  featureId: string;
  groups: TaskGroup[];
  group?: TaskGroup | undefined;
  onSaved: (group: TaskGroup) => void;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const titleId = useId();
  const input = useRef<HTMLInputElement>(null);
  const {
    name,
    setName,
    color,
    setColor,
    normalizedColor,
    invalidColor,
    duplicate,
    canSave,
    dirty,
    save,
  } = useTaskGroupDraft(featureId, groups, group);
  const [confirmation, setConfirmation] = useState<"discard" | "delete">();
  const covered = confirmation !== undefined;

  function requestClose() {
    if (dirty) setConfirmation("discard");
    else onClose();
  }
  useCloseOnEscape(requestClose, !save.isPending);
  const { dialogRef, onKeyDown } = useDialogFocusTrap<HTMLElement>(input);

  async function submit(event: SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!canSave || save.isPending) return;
    const result = await save.mutateAsync().catch(() => undefined);
    if (result?.taskGroup) onSaved(result.taskGroup);
  }

  return createPortal(
    <>
      <div className="scrim" role="presentation" onKeyDown={onKeyDown}>
        <section
          ref={dialogRef}
          className="dialog task-groups-dialog"
          role="dialog"
          aria-modal="true"
          aria-labelledby={titleId}
          inert={covered}
          aria-hidden={covered || undefined}
        >
          <header>
            <h2 id={titleId}>
              {group
                ? t("taskGroups.editGroup", { name: group.name })
                : t("taskGroups.create")}
            </h2>
          </header>
          <form onSubmit={submit}>
            <TaskGroupFields
              input={input}
              name={name}
              onNameChange={setName}
              duplicate={duplicate}
              color={group ? color : undefined}
              pickerValue={invalidColor ? "" : normalizedColor}
              invalidColor={invalidColor}
              onColorChange={setColor}
              disabled={save.isPending}
            />
            <MutationError error={save.error} />
            <TaskGroupActions
              canSave={canSave}
              pending={save.isPending}
              onCancel={requestClose}
              onDelete={
                group
                  ? () => {
                      setConfirmation("delete");
                    }
                  : undefined
              }
            />
          </form>
        </section>
      </div>
      {confirmation === "discard" && (
        <DiscardChangesDialog
          onCancel={() => {
            setConfirmation(undefined);
          }}
          onConfirm={onClose}
        />
      )}
      {group && confirmation === "delete" && (
        <DeleteTaskGroupDialog
          group={group}
          onCancel={() => {
            setConfirmation(undefined);
          }}
          onDeleted={onClose}
        />
      )}
    </>,
    document.body,
  );
}

function useTaskGroupDraft(
  featureId: string,
  groups: TaskGroup[],
  group: TaskGroup | undefined,
) {
  const baseline = group?.name ?? "";
  const [name, setName] = useState(baseline);
  const savedColor = group?.color ?? "";
  const [color, setColor] = useState(savedColor);
  const normalizedColor = color.trim().toLowerCase();
  const invalidColor =
    normalizedColor !== "" && !/^#[0-9a-f]{6}$/u.test(normalizedColor);
  const colorChanged = normalizedColor !== savedColor;
  const trimmed = name.trim();
  const duplicate = groups.some(
    (other) => other.id !== group?.id && other.name === trimmed,
  );
  const nameChanged = trimmed !== baseline;
  const canSave =
    trimmed !== "" &&
    !duplicate &&
    !invalidColor &&
    (nameChanged || colorChanged);
  const save = useDomainMutation(async () =>
    group
      ? mutations.updateTaskGroup({
          id: group.id,
          ...(nameChanged ? { name: trimmed } : {}),
          ...(colorChanged ? { color: normalizedColor } : {}),
        })
      : mutations.createTaskGroup({ featureId, name: trimmed }),
  );

  return {
    name,
    setName,
    color,
    setColor,
    normalizedColor,
    invalidColor,
    duplicate,
    canSave,
    save,
    dirty: name !== baseline || color !== savedColor,
  };
}

function TaskGroupFields({
  input,
  name,
  onNameChange,
  duplicate,
  color,
  pickerValue,
  invalidColor,
  onColorChange,
  disabled,
}: {
  input: Ref<HTMLInputElement>;
  name: string;
  onNameChange: (name: string) => void;
  duplicate: boolean;
  color: string | undefined;
  pickerValue: string;
  invalidColor: boolean;
  onColorChange: (color: string) => void;
  disabled: boolean;
}) {
  const { t } = useTranslation();
  const colorId = useId();
  return (
    <>
      <label>
        {t("taskGroups.newName")}
        <input
          ref={input}
          value={name}
          disabled={disabled}
          aria-invalid={duplicate}
          onChange={(event) => {
            onNameChange(event.target.value);
          }}
        />
      </label>
      {duplicate && (
        <p className="form-error" role="alert">
          {t("taskGroups.duplicateName")}
        </p>
      )}
      {color !== undefined && (
        <>
          <label htmlFor={colorId}>{t("taskGroups.cardColor")}</label>
          <div className="task-group-color-fields">
            <input
              type="color"
              aria-label={t("taskGroups.pickColor")}
              value={pickerValue || "#6366f1"}
              disabled={disabled}
              onChange={(event) => {
                onColorChange(event.target.value);
              }}
            />
            <input
              id={colorId}
              type="text"
              value={color}
              placeholder={t("taskGroups.defaultColor")}
              aria-invalid={invalidColor}
              aria-describedby={invalidColor ? colorId + "-error" : undefined}
              disabled={disabled}
              onChange={(event) => {
                onColorChange(event.target.value);
              }}
            />
            <button
              type="button"
              className="button secondary"
              disabled={!color || disabled}
              onClick={() => {
                onColorChange("");
              }}
            >
              {t("taskGroups.resetColor")}
            </button>
          </div>
          {invalidColor && (
            <p id={colorId + "-error"} className="form-error" role="alert">
              {t("taskGroups.invalidColor")}
            </p>
          )}
        </>
      )}
    </>
  );
}

function TaskGroupActions({
  canSave,
  pending,
  onCancel,
  onDelete,
}: {
  canSave: boolean;
  pending: boolean;
  onCancel: () => void;
  onDelete: (() => void) | undefined;
}) {
  const { t } = useTranslation();
  return (
    <footer>
      {onDelete && (
        <IconButton
          className="task-group-delete"
          icon={Trash2}
          aria-haspopup="dialog"
          label={t("taskGroups.delete")}
          variant="danger"
          disabled={pending}
          onClick={onDelete}
        />
      )}
      <IconButton
        icon={X}
        label={t("common.cancel")}
        disabled={pending}
        onClick={onCancel}
      />
      <SaveButton
        type="submit"
        label={t(onDelete ? "taskGroups.save" : "taskGroups.create")}
        dirty={canSave}
        pending={pending}
        busy={pending}
      />
    </footer>
  );
}

function DeleteTaskGroupDialog({
  group,
  onCancel,
  onDeleted,
}: {
  group: TaskGroup;
  onCancel: () => void;
  onDeleted: () => void;
}) {
  const { t } = useTranslation();
  const remove = useDomainMutation(mutations.deleteTaskGroup);
  return (
    <ConfirmationDialog
      title={t("taskGroups.deleteGroup", { name: group.name })}
      description={t("taskGroups.deleteDetail")}
      confirmLabel={t("taskGroups.delete")}
      danger
      pending={remove.isPending}
      error={remove.error}
      onCancel={onCancel}
      onConfirm={() => {
        remove.mutate(group.id, { onSuccess: onDeleted });
      }}
    />
  );
}

export function TaskGroupOverlay({
  action,
  documents,
  groups,
  readOnly,
  onClose,
}: {
  action: GroupAction;
  documents: TaskNodeDocument[];
  groups: TaskGroup[];
  readOnly: boolean;
  onClose: () => void;
}) {
  const { group } = action;
  if (action.type === "references")
    return (
      <TaskGroupReferencesDialog
        group={group}
        documents={documents}
        onClose={onClose}
      />
    );
  if (readOnly) return null;
  return (
    <TaskGroupDialog
      featureId={group.featureId}
      group={group}
      groups={groups}
      onSaved={onClose}
      onClose={onClose}
    />
  );
}

function TaskGroupReferencesDialog({
  group,
  documents,
  onClose,
}: {
  group: TaskGroup;
  documents: TaskNodeDocument[];
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const titleId = useId();
  const close = useRef<HTMLButtonElement>(null);
  const [preview, setPreview] = useState<TaskNodeDocument>();
  const { dialogRef, onKeyDown } = useDialogFocusTrap<HTMLElement>(close);
  useCloseOnEscape(onClose);
  useEffect(() => {
    if (!preview) close.current?.focus();
  }, [preview]);

  return (
    <>
      <div className="scrim" role="presentation" onKeyDown={onKeyDown}>
        <section
          ref={dialogRef}
          className="dialog task-groups-dialog"
          role="dialog"
          aria-modal="true"
          aria-labelledby={titleId}
          aria-hidden={preview ? true : undefined}
          inert={preview !== undefined}
        >
          <header className="task-group-references-header">
            <h2 id={titleId}>
              {t("taskGroups.referencesFor", { name: group.name })}
            </h2>
            <IconButton
              ref={close}
              icon={X}
              label={t("common.close")}
              iconOnly
              onClick={onClose}
            />
          </header>
          {documents.length === 0 && (
            <p className="read-only-empty">{t("inspector.noReferences")}</p>
          )}
          {documents.map((document) => (
            <DocumentRow
              key={document.id}
              document={document}
              onPreview={setPreview}
              canEdit={false}
              canDelete={false}
            />
          ))}
        </section>
      </div>
      {preview && (
        <MarkdownPreview
          document={preview}
          onClose={() => {
            setPreview(undefined);
          }}
        />
      )}
    </>
  );
}
