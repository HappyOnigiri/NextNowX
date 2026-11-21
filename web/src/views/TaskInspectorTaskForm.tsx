import { useTranslation } from "react-i18next";
import {
  TaskKind,
  TaskStatus,
  type Task,
  type TaskGroup,
  type TaskLabelAppearances,
} from "../gen/nnx/v1/nnx_pb";
import { taskStatusLabel } from "../i18n/domain";
import { MutationError } from "./MutationError";
import { type TaskDraftController } from "./taskDraft";
import { TaskGroupSelect } from "./TaskGroupSelect";
import { TaskKindLabel } from "./TaskKindLabel";
import { TaskKindSelect } from "./TaskKindSelect";
import { taskStatusLabelKey } from "./taskStatusLabelKey";

export function TaskInspectorTaskForm({
  task,
  controller,
  groups,
  appearances,
  readOnly = false,
  hasPullRequest = false,
}: {
  task: Task;
  controller: TaskDraftController;
  groups: TaskGroup[];
  appearances?: TaskLabelAppearances | undefined;
  readOnly?: boolean;
  hasPullRequest?: boolean;
}) {
  const { t } = useTranslation();

  if (readOnly)
    return (
      <TaskInspectorReadOnlyValues
        task={task}
        groupName={groups.find((group) => group.id === task.groupId)?.name}
      />
    );

  const { draft, setDraft } = controller;
  return (
    <div className="inspector-task-fields">
      <label>
        {t("common.title")}
        <input
          name="title"
          value={draft.title}
          onChange={(event) => {
            setDraft({ ...draft, title: event.target.value });
          }}
        />
      </label>
      <TaskKindSelect
        value={draft.kind}
        hasPullRequest={hasPullRequest}
        onChange={(kind) => {
          setDraft({
            ...draft,
            kind,
            status:
              kind === TaskKind.WORK && draft.status === TaskStatus.DESIGNING
                ? TaskStatus.NOT_STARTED
                : draft.status,
          });
        }}
      />
      <TaskGroupSelect
        createInFeature={task.featureId}
        disabled={controller.pending}
        groups={groups}
        value={draft.groupId ?? task.groupId}
        onChange={(groupId) => {
          setDraft({
            ...draft,
            groupId: groupId === task.groupId ? undefined : groupId,
          });
        }}
      />
      <div className="form-row">
        <label>
          {t("common.status")}
          <select
            name="status"
            value={draft.status}
            onChange={(event) => {
              setDraft({ ...draft, status: Number(event.target.value) });
            }}
          >
            {[
              TaskStatus.NOT_STARTED,
              TaskStatus.DESIGNING,
              TaskStatus.IN_PROGRESS,
              TaskStatus.COMPLETED,
              TaskStatus.CLOSED,
            ]
              .filter(
                (status) =>
                  draft.kind !== TaskKind.WORK ||
                  status !== TaskStatus.DESIGNING,
              )
              .map((status) => (
                <option value={status} key={status}>
                  {draft.kind === TaskKind.WORK &&
                  status === TaskStatus.IN_PROGRESS
                    ? t("taskKind.started")
                    : taskStatusLabelFor(status, appearances, t)}
                </option>
              ))}
          </select>
        </label>
        <label>
          {t("common.assignee")}
          <input
            name="assignee"
            value={draft.assignee}
            onChange={(event) => {
              setDraft({ ...draft, assignee: event.target.value });
            }}
          />
        </label>
      </div>
      <MutationError error={controller.error} />
    </div>
  );
}

function taskStatusLabelFor(
  status: TaskStatus,
  appearances: TaskLabelAppearances | undefined,
  t: Parameters<typeof taskStatusLabel>[1],
): string {
  const appearance = appearances?.values.find(
    (item) => item.key === taskStatusLabelKey(status),
  );
  if (appearance?.textOverridden && appearance.text) return appearance.text;
  return taskStatusLabel(status, t);
}

function TaskInspectorReadOnlyValues({
  task,
  groupName,
}: {
  task: Task;
  groupName: string | undefined;
}) {
  const { t } = useTranslation();
  return (
    <dl className="inspector-values">
      <div>
        <dt>{t("common.kind")}</dt>
        <dd>
          <TaskKindLabel kind={task.kind} />
        </dd>
      </div>
      <div>
        <dt>{t("common.title")}</dt>
        <dd>{task.title}</dd>
      </div>
      <div>
        <dt>{t("taskGroups.field")}</dt>
        <dd>{groupName ?? t("taskGroups.ungrouped")}</dd>
      </div>
      <div>
        <dt>{t("common.assignee")}</dt>
        <dd>{task.assignee || t("common.unassigned")}</dd>
      </div>
    </dl>
  );
}
