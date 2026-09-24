import { TaskStatus } from "../gen/nnx/v1/nnx_pb";

export function taskStatusLabelKey(status: TaskStatus): string {
  switch (status) {
    case TaskStatus.UNSPECIFIED:
      return "status.unknown";
    case TaskStatus.NOT_STARTED:
      return "status.not_started";
    case TaskStatus.DESIGNING:
      return "status.designing";
    case TaskStatus.IN_PROGRESS:
      return "status.in_progress";
    case TaskStatus.COMPLETED:
      return "status.completed";
    case TaskStatus.CLOSED:
      return "status.closed";
  }
}
