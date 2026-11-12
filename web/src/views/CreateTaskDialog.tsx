import { Plus, X } from "lucide-react";
import { useRef, useState, type SyntheticEvent } from "react";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import { formValue } from "../form";
import { TaskKind } from "../gen/nnx/v1/nnx_pb";
import { useDomainMutation, useSnapshotRefresh } from "../hooks";
import { formatError } from "../i18n/domain";
import { dependencyPair, type PendingDependency } from "./dependencyGraph";
import { IconButton } from "./IconButton";
import { MutationError } from "./MutationError";
import { TaskKindSelect } from "./TaskKindSelect";
import { useCloseOnEscape } from "./useCloseOnEscape";

interface CreateTaskInput {
  featureId: string;
  title: string;
  assignee: string;
  kind: TaskKind;
}

// 依存の相手のタイトルは呼び出し側で解決する。ダイアログ側にも控えを置くと、
// どちらのフォールバックが効いているのか追えなくなる。
interface CreateTaskDialogProps {
  featureId: string;
  onClose: () => void;
  dependency?: { value: PendingDependency; title: string };
}

export function CreateTaskDialog({
  featureId,
  onClose,
  dependency,
}: CreateTaskDialogProps) {
  const { t } = useTranslation();
  const refreshSnapshot = useSnapshotRefresh();
  const [dependencyFailed, setDependencyFailed] = useState(false);
  // 依存の追加だけが失敗したときの再送で、タスクを二重に作らないための控え。
  const createdTaskId = useRef<string>(undefined);
  const createTask = useDomainMutation(async (input: CreateTaskInput) => {
    const created = createdTaskId.current;
    // 再送では作成をやり直さないので、その間に直したフォームの値は更新で送る。
    if (created)
      await mutations.updateTask({
        id: created,
        title: input.title,
        assignee: input.assignee,
        kind: input.kind,
      });
    const taskId = created ?? (await mutations.createTask(input)).task?.id;
    // ID が取れないと依存を張れないので、成功として閉じずにエラーを見せる。
    if (!taskId) throw new Error("createTask did not return a task id");
    createdTaskId.current = taskId;
    if (dependency) {
      const { blocker, blocked } = dependencyPair(dependency.value, taskId);
      try {
        await mutations.addDependency(blocker, blocked);
      } catch (error) {
        // タスクだけが書き込まれた状態なので、依存が欠けたノードをグラフに出す。
        setDependencyFailed(true);
        await refreshSnapshot();
        throw error;
      }
    }
  });

  useCloseOnEscape(onClose);

  async function submitTask(event: SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = new FormData(event.currentTarget);
    try {
      await createTask.mutateAsync({
        featureId,
        title: formValue(form, "title"),
        assignee: formValue(form, "assignee"),
        kind: Number(formValue(form, "kind")),
      });
    } catch {
      return;
    }
    onClose();
  }

  return (
    <div className="scrim">
      <form
        className="dialog"
        onSubmit={submitTask}
        aria-label={t("taskCreate.formLabel")}
      >
        <header>
          <h2>{t("taskCreate.title")}</h2>
        </header>
        {dependency && (
          <p className="dialog-lead">
            {t(
              dependency.value.direction === "blocks"
                ? "taskCreate.dependencyBlocks"
                : "taskCreate.dependencyBlockedBy",
              { title: dependency.title },
            )}
          </p>
        )}
        <label>
          {t("common.title")}
          <input
            name="title"
            required
            placeholder={t("taskCreate.titlePlaceholder")}
          />
        </label>
        <TaskKindSelect defaultValue={TaskKind.DEVELOPMENT} />
        <label>
          {t("common.assignee")}
          <input
            name="assignee"
            placeholder={t("taskCreate.assigneePlaceholder")}
          />
        </label>
        {dependencyFailed && (
          <p className="form-error">{t("taskCreate.dependencyFailed")}</p>
        )}
        {createTask.error && (
          <p className="form-error">{formatError(createTask.error, t)}</p>
        )}
        <footer>
          <IconButton
            icon={X}
            label={t("common.cancel")}
            variant="secondary"
            onClick={onClose}
          />
          <IconButton
            icon={Plus}
            label={t("taskCreate.submit")}
            variant="primary"
            type="submit"
            disabled={createTask.isPending}
          />
        </footer>
        <MutationError error={createTask.error} />
      </form>
    </div>
  );
}
