import { Link } from "@tanstack/react-router";
import { ChevronRight } from "lucide-react";
import type { TaskGroup } from "../gen/nnx/v1/nnx_pb";

export function TaskGroupLink({
  group,
  className,
}: {
  group: TaskGroup;
  className?: string;
}) {
  return (
    <Link
      className={className ? `task-group-link ${className}` : "task-group-link"}
      to="/features/$featureId"
      params={{ featureId: group.featureId }}
      search={{ group: group.id }}
      title={group.name}
    >
      <span className="task-group-link-name">{group.name}</span>
      <ChevronRight
        className="task-group-link-icon"
        size={14}
        aria-hidden="true"
      />
    </Link>
  );
}
