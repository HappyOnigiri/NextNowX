import { Check, Trash2, X } from "lucide-react";
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
import { taskGroupColors } from "./taskGroupColors";
import { useCloseOnEscape } from "./useCloseOnEscape";
import { useDialogFocusTrap } from "./useDialogFocusTrap";

const cardColorPalette = [
  { name: "slate", value: "#64748b" },
  { name: "blue", value: "#3b82f6" },
  { name: "teal", value: "#14b8a6" },
  { name: "lime", value: "#65a30d" },
  { name: "amber", value: "#d97706" },
  { name: "purple", value: "#8b5cf6" },
  { name: "pink", value: "#db2777" },
] as const;

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
  const { name, setName, color, setColor, duplicate, canSave, dirty, save } =
    useTaskGroupDraft(featureId, groups, group);
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
              disabled={save.isPending}
            />
            {group && (
              <TaskGroupColorPalette
                color={color}
                savedColor={group.color}
                onChange={setColor}
                disabled={save.isPending}
              />
            )}
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
  const colorChanged = color !== savedColor;
  const trimmed = name.trim();
  const duplicate = groups.some(
    (other) => other.id !== group?.id && other.name === trimmed,
  );
  const nameChanged = trimmed !== baseline;
  const canSave = trimmed !== "" && !duplicate && (nameChanged || colorChanged);
  const save = useDomainMutation(async () =>
    group
      ? mutations.updateTaskGroup({
          id: group.id,
          ...(nameChanged ? { name: trimmed } : {}),
          ...(colorChanged ? { color } : {}),
        })
      : mutations.createTaskGroup({ featureId, name: trimmed }),
  );

  return {
    name,
    setName,
    color,
    setColor,
    duplicate,
    canSave,
    save,
    dirty: name !== baseline || colorChanged,
  };
}

function TaskGroupFields({
  input,
  name,
  onNameChange,
  duplicate,
  disabled,
}: {
  input: Ref<HTMLInputElement>;
  name: string;
  onNameChange: (name: string) => void;
  duplicate: boolean;
  disabled: boolean;
}) {
  const { t } = useTranslation();
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
    </>
  );
}

function TaskGroupColorPalette({
  color,
  savedColor,
  onChange,
  disabled,
}: {
  color: string;
  savedColor: string;
  onChange: (color: string) => void;
  disabled: boolean;
}) {
  const { t } = useTranslation();
  const name = useId();
  const options: { label: string; value: string }[] = [
    { label: t("taskGroups.defaultColor"), value: "" },
    ...cardColorPalette.map(({ name, value }) => ({
      label: t(`taskGroups.paletteColors.${name}`),
      value,
    })),
  ];
  if (savedColor && !options.some(({ value }) => value === savedColor)) {
    options.push({
      label: t("taskGroups.savedColor", { color: savedColor }),
      value: savedColor,
    });
  }

  return (
    <fieldset className="task-group-color-palette" disabled={disabled}>
      <legend>{t("taskGroups.cardColor")}</legend>
      <div className="task-group-color-options">
        {options.map(({ label, value }) => (
          <label key={value} title={label}>
            <input
              type="radio"
              name={name}
              aria-label={label}
              value={value}
              checked={color === value}
              onChange={() => {
                onChange(value);
              }}
            />
            <span
              className="task-group-color-swatch"
              style={taskGroupColors(value)}
            >
              <Check size={14} aria-hidden="true" />
              {value === "" && label}
            </span>
          </label>
        ))}
      </div>
    </fieldset>
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
