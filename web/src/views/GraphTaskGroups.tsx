import { ViewportPortal } from "@xyflow/react";
import { Paperclip, Pencil } from "lucide-react";
import type { Ref } from "react";
import { useTranslation } from "react-i18next";
import type { TaskGroup } from "../gen/nnx/v1/nnx_pb";
import { CopyTextButton } from "./CopyableIdentifier";
import { IconButton } from "./IconButton";
import { taskGroupColors } from "./taskGroupColors";
import { taskGroupWidth, type TaskGroupFrame } from "./taskGroupGraphLayout";
import { TaskGroupLink } from "./TaskGroupLink";

export interface GroupAction {
  type: "references" | "edit";
  group: TaskGroup;
}

interface GroupHeaderActions {
  onGroupAction: ((action: GroupAction) => void) | undefined;
  readOnly?: boolean;
}

export function GraphTaskGroups({
  frames,
  onGroupAction,
  readOnly = false,
}: GroupHeaderActions & { frames: TaskGroupFrame[] }) {
  return (
    <ViewportPortal>
      {frames.map(({ group, x, y, width, height }) => (
        <div
          key={group.id}
          className="graph-task-group"
          style={{
            left: x,
            top: y,
            width,
            height,
            ...taskGroupColors(group.color),
          }}
        >
          <GraphTaskGroupHeader
            group={group}
            onGroupAction={onGroupAction}
            readOnly={readOnly}
          />
        </div>
      ))}
    </ViewportPortal>
  );
}

export function GraphTaskGroupMeasurements({
  groups,
  readOnly,
  onGroupAction,
  ref,
}: GroupHeaderActions & {
  groups: TaskGroup[] | undefined;
  ref: Ref<HTMLDivElement>;
}) {
  if (!groups?.length) return null;
  return (
    <div
      ref={ref}
      className="graph-task-group-measurements"
      aria-hidden="true"
      inert
    >
      {groups.map((group) => (
        <div
          key={group.id}
          className="graph-task-group-measurement"
          style={{ width: taskGroupWidth }}
        >
          <GraphTaskGroupHeader
            group={group}
            onGroupAction={onGroupAction}
            readOnly={readOnly ?? false}
          />
        </div>
      ))}
    </div>
  );
}

function GraphTaskGroupHeader({
  group,
  onGroupAction,
  readOnly,
}: GroupHeaderActions & { group: TaskGroup }) {
  const { t } = useTranslation();
  return (
    <div className="graph-task-group-header nopan" data-group-id={group.id}>
      <TaskGroupLink group={group} className="graph-task-group-link" />
      <CopyTextButton label={t("taskGroups.copyName")} value={group.name} />
      {onGroupAction && (
        <IconButton
          icon={Paperclip}
          iconOnly
          size="compact"
          aria-haspopup="dialog"
          label={t("taskGroups.referencesFor", { name: group.name })}
          onClick={() => {
            onGroupAction({ type: "references", group });
          }}
        />
      )}
      {onGroupAction && !readOnly && (
        <IconButton
          icon={Pencil}
          iconOnly
          size="compact"
          aria-haspopup="dialog"
          label={t("taskGroups.editGroup", { name: group.name })}
          onClick={() => {
            onGroupAction({ type: "edit", group });
          }}
        />
      )}
    </div>
  );
}
