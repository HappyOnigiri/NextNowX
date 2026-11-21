import { Link } from "@tanstack/react-router";
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
  ...action
}: {
  state: TaskDisplayState;
  kind: TaskKind | undefined;
  appearances: TaskLabelAppearances | undefined;
  title: string;
} & ({ onOpen: () => void } | { featureId: string; taskId: string })) {
  return (
    <div className="task-group-row">
      <TaskStatusBadge state={state} kind={kind} appearances={appearances} />
      {"onOpen" in action ? (
        <button
          type="button"
          className="task-group-row-title nodrag nopan"
          title={title}
          onClick={action.onOpen}
        >
          {title}
        </button>
      ) : (
        <Link
          to="/features/$featureId"
          params={{ featureId: action.featureId }}
          search={{ task: action.taskId }}
          className="task-group-row-title"
          title={title}
        >
          {title}
        </Link>
      )}
    </div>
  );
}
