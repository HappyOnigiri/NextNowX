import { X } from "lucide-react";
import { useId, useRef, useState, type SyntheticEvent } from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import type { TaskGroup } from "../gen/nnx/v1/nnx_pb";
import { useDomainMutation } from "../hooks";
import { ConfirmationDialog } from "./ConfirmationDialog";
import { IconButton } from "./IconButton";
import { MutationError } from "./MutationError";
import { DiscardChangesDialog, SaveButton } from "./UnsavedChanges";
import { useCloseOnEscape } from "./useCloseOnEscape";
import { useDialogFocusTrap } from "./useDialogFocusTrap";

export function TaskGroupNameDialog({
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
  const baseline = group?.name ?? "";
  const [name, setName] = useState(baseline);
  const [discarding, setDiscarding] = useState(false);
  const trimmed = name.trim();
  const duplicate = groups.some(
    (other) => other.id !== group?.id && other.name === trimmed,
  );
  const canSave = trimmed !== "" && trimmed !== baseline && !duplicate;
  const save = useDomainMutation(async (value: string) =>
    group
      ? mutations.renameTaskGroup({ id: group.id, name: value })
      : mutations.createTaskGroup({ featureId, name: value }),
  );

  function requestClose() {
    if (name !== baseline) setDiscarding(true);
    else onClose();
  }
  useCloseOnEscape(requestClose, !save.isPending);
  const { dialogRef, onKeyDown } = useDialogFocusTrap<HTMLElement>(input);

  async function submit(event: SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!canSave || save.isPending) return;
    const result = await save.mutateAsync(trimmed).catch(() => undefined);
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
          inert={discarding}
          aria-hidden={discarding || undefined}
        >
          <header>
            <h2 id={titleId}>
              {group
                ? t("taskGroups.renameGroup", { name: group.name })
                : t("taskGroups.create")}
            </h2>
          </header>
          <form onSubmit={submit}>
            <label>
              {t("taskGroups.newName")}
              <input
                ref={input}
                value={name}
                disabled={save.isPending}
                aria-invalid={duplicate}
                onChange={(event) => {
                  setName(event.target.value);
                }}
              />
            </label>
            {duplicate && (
              <p className="form-error" role="alert">
                {t("taskGroups.duplicateName")}
              </p>
            )}
            <MutationError error={save.error} />
            <footer>
              <IconButton
                icon={X}
                label={t("common.cancel")}
                disabled={save.isPending}
                onClick={requestClose}
              />
              <SaveButton
                type="submit"
                label={t(group ? "taskGroups.rename" : "taskGroups.create")}
                dirty={canSave}
                pending={save.isPending}
                busy={save.isPending}
              />
            </footer>
          </form>
        </section>
      </div>
      {discarding && (
        <DiscardChangesDialog
          onCancel={() => {
            setDiscarding(false);
          }}
          onConfirm={onClose}
        />
      )}
    </>,
    document.body,
  );
}

export function DeleteTaskGroupDialog({
  group,
  onClose,
}: {
  group: TaskGroup;
  onClose: () => void;
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
      onCancel={onClose}
      onConfirm={() => {
        remove.mutate(group.id, { onSuccess: onClose });
      }}
    />
  );
}
