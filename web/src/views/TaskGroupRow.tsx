import type {
  TaskDisplayState,
  TaskKind,
  TaskLabelAppearances,
} from "../gen/nnx/v1/nnx_pb";
import { TaskStatusBadge } from "./TaskStateBadges";

export function TaskGroupRow({
  state,
  kind,
  appearances,
  title,
}: {
  state: TaskDisplayState;
  kind: TaskKind | undefined;
  appearances: TaskLabelAppearances | undefined;
  title: string;
}) {
  return (
    <div className="task-group-row">
      <TaskStatusBadge state={state} kind={kind} appearances={appearances} />
      <span className="task-group-row-title" title={title}>
        {title}
      </span>
    </div>
  );
}
