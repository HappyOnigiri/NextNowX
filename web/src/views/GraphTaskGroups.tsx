import { ViewportPortal } from "@xyflow/react";
import { Paperclip, Pencil, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { TaskGroup } from "../gen/nnx/v1/nnx_pb";
import { IconButton } from "./IconButton";
import type { TaskGroupFrame } from "./taskGroupGraphLayout";
import { TaskGroupLink } from "./TaskGroupLink";

export interface GroupAction {
  type: "references" | "rename" | "delete";
  group: TaskGroup;
}

export function GraphTaskGroups({
  frames,
  onGroupAction,
  readOnly = false,
}: {
  frames: TaskGroupFrame[];
  readOnly?: boolean;
  onGroupAction: ((action: GroupAction) => void) | undefined;
}) {
  const { t } = useTranslation();
  return (
    <ViewportPortal>
      {frames.map(({ group, x, y, width, height }) => (
        <div
          key={group.id}
          className="graph-task-group"
          style={{ left: x, top: y, width, height }}
        >
          <div className="graph-task-group-header nopan">
            <TaskGroupLink group={group} className="graph-task-group-link" />
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
              <>
                <IconButton
                  icon={Pencil}
                  iconOnly
                  size="compact"
                  label={t("taskGroups.renameGroup", { name: group.name })}
                  onClick={() => {
                    onGroupAction({ type: "rename", group });
                  }}
                />
                <IconButton
                  icon={Trash2}
                  iconOnly
                  size="compact"
                  variant="danger"
                  label={t("taskGroups.deleteGroup", { name: group.name })}
                  onClick={() => {
                    onGroupAction({ type: "delete", group });
                  }}
                />
              </>
            )}
          </div>
        </div>
      ))}
    </ViewportPortal>
  );
}
