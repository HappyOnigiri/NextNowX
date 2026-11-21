import { Pencil, Trash2, X } from "lucide-react";
import {
  useEffect,
  useId,
  useRef,
  useState,
  type Ref,
  type SyntheticEvent,
} from "react";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import type { TaskGroup } from "../gen/nnx/v1/nnx_pb";
import { useDomainMutation } from "../hooks";
import { ConfirmationDialog } from "./ConfirmationDialog";
import { IconButton } from "./IconButton";
import { MutationError } from "./MutationError";
import { DiscardChangesDialog, SaveButton } from "./UnsavedChanges";
import { useCloseOnEscape } from "./useCloseOnEscape";

export function TaskGroupsDialog({
  featureId,
  groups,
  onClose,
}: {
  featureId: string;
  groups: TaskGroup[];
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const titleId = useId();
  const input = useRef<HTMLInputElement>(null);
  const [editing, setEditing] = useState<TaskGroup>();
  const [name, setName] = useState("");
  const [deleting, setDeleting] = useState<TaskGroup>();
  const [discarding, setDiscarding] = useState(false);
  const trimmedName = name.trim();
  const baseline = editing?.name ?? "";
  const dirty = name !== baseline;
  const duplicate = groups.some(
    (group) => group.id !== editing?.id && group.name === trimmedName,
  );
  const canSave = trimmedName !== "" && trimmedName !== baseline && !duplicate;
  const save = useDomainMutation(async (value: string) => {
    if (editing)
      await mutations.renameTaskGroup({ id: editing.id, name: value });
    else await mutations.createTaskGroup({ featureId, name: value });
  });
  const confirmationOpen = !!deleting || discarding;
  function requestClose() {
    if (dirty) setDiscarding(true);
    else onClose();
  }
  useCloseOnEscape(requestClose, !save.isPending);

  useEffect(() => {
    const origin = document.activeElement;
    input.current?.focus();
    return () => {
      if (origin instanceof HTMLElement && origin.isConnected) origin.focus();
    };
  }, []);

  function edit(group?: TaskGroup) {
    save.reset();
    setEditing(group);
    setName(group?.name ?? "");
    input.current?.focus();
  }

  async function submit(event: SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!canSave || save.isPending) return;
    try {
      await save.mutateAsync(trimmedName);
    } catch {
      return;
    }
    edit();
  }

  return (
    <>
      <div className="scrim">
        <section
          className="dialog task-groups-dialog"
          role="dialog"
          aria-modal="true"
          aria-labelledby={titleId}
          inert={confirmationOpen}
          aria-hidden={confirmationOpen || undefined}
        >
          <header>
            <h2 id={titleId}>{t("taskGroups.manage")}</h2>
          </header>
          <TaskGroupRows
            groups={groups}
            disabled={save.isPending || dirty}
            onEdit={edit}
            onDelete={setDeleting}
          />
          <TaskGroupNameForm
            input={input}
            name={name}
            editing={editing !== undefined}
            dirty={dirty}
            duplicate={duplicate}
            canSave={canSave}
            pending={save.isPending}
            error={save.error}
            onNameChange={setName}
            onSubmit={submit}
            onClose={requestClose}
            onReset={() => {
              edit();
            }}
          />
        </section>
      </div>
      {deleting && (
        <DeleteTaskGroupDialog
          group={deleting}
          onCancel={() => {
            setDeleting(undefined);
          }}
          onDeleted={() => {
            if (editing?.id === deleting.id) edit();
            setDeleting(undefined);
          }}
        />
      )}
      {discarding && (
        <DiscardChangesDialog
          onCancel={() => {
            setDiscarding(false);
          }}
          onConfirm={onClose}
        />
      )}
    </>
  );
}

function TaskGroupNameForm({
  input,
  name,
  editing,
  dirty,
  duplicate,
  canSave,
  pending,
  error,
  onNameChange,
  onSubmit,
  onClose,
  onReset,
}: {
  input: Ref<HTMLInputElement>;
  name: string;
  editing: boolean;
  dirty: boolean;
  duplicate: boolean;
  canSave: boolean;
  pending: boolean;
  error: Error | null;
  onNameChange: (name: string) => void;
  onSubmit: (event: SyntheticEvent<HTMLFormElement>) => Promise<void>;
  onClose: () => void;
  onReset: () => void;
}) {
  const { t } = useTranslation();
  return (
    <form onSubmit={onSubmit}>
      <label>
        {t(editing ? "taskGroups.newName" : "taskGroups.createName")}
        <input
          ref={input}
          value={name}
          disabled={pending}
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
      <MutationError error={error} />
      <footer>
        <IconButton
          icon={X}
          label={t("common.close")}
          disabled={pending}
          onClick={onClose}
        />
        {(editing || dirty) && (
          <IconButton
            icon={X}
            label={t("common.cancel")}
            disabled={pending}
            onClick={onReset}
          />
        )}
        <SaveButton
          type="submit"
          label={t(editing ? "taskGroups.rename" : "taskGroups.create")}
          dirty={canSave}
          pending={pending}
          busy={pending}
        />
      </footer>
    </form>
  );
}

function TaskGroupRows({
  groups,
  disabled,
  onEdit,
  onDelete,
}: {
  groups: TaskGroup[];
  disabled: boolean;
  onEdit: (group: TaskGroup) => void;
  onDelete: (group: TaskGroup) => void;
}) {
  const { t } = useTranslation();
  if (!groups.length) return null;
  return (
    <ul className="task-groups-list">
      {groups.map((group) => (
        <li key={group.id}>
          <span>{group.name}</span>
          <IconButton
            icon={Pencil}
            iconOnly
            label={t("taskGroups.renameGroup", { name: group.name })}
            size="compact"
            disabled={disabled}
            onClick={() => {
              onEdit(group);
            }}
          />
          <IconButton
            icon={Trash2}
            iconOnly
            label={t("taskGroups.deleteGroup", { name: group.name })}
            size="compact"
            variant="danger"
            disabled={disabled}
            onClick={() => {
              onDelete(group);
            }}
          />
        </li>
      ))}
    </ul>
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
