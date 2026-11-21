import type { ElkNode, ElkPort } from "elkjs/lib/elk-api.js";
import type { Dependency, Task, TaskGroup } from "../gen/nnx/v1/nnx_pb";
import { dependencyEdgeId } from "./dependencyGraph";

const groupWidth = 336;
const headerHeight = 40;
const rowHeight = 36;
const bottomPadding = 8;

interface GroupLayoutNode {
  id: string;
  groupId: string;
  width: number;
  height: number;
}

export interface TaskGroupFrame {
  group: TaskGroup;
  x: number;
  y: number;
  width: number;
  height: number;
}

export function graphOverviewDependencies(
  tasks: Task[],
  dependencies: Dependency[],
  groups: TaskGroup[] | undefined,
) {
  if (!groups?.length) return dependencies;
  const groupIds = new Set(groups.map((group) => group.id));
  const byTask = new Map(
    tasks
      .filter((task) => groupIds.has(task.groupId))
      .map((task) => [task.id, task.groupId]),
  );
  return dependencies.filter(({ blockerTaskId, blockedTaskId }) => {
    const group = byTask.get(blockerTaskId);
    return group === undefined || group !== byTask.get(blockedTaskId);
  });
}

function rowPort(taskId: string, side: "WEST" | "EAST") {
  return `${taskId}:${side}`;
}

function groupPorts(taskId: string, width: number, y: number): ElkPort[] {
  return [
    {
      id: rowPort(taskId, "WEST"),
      x: 0,
      y,
      layoutOptions: { "elk.port.side": "WEST" },
    },
    {
      id: rowPort(taskId, "EAST"),
      x: width,
      y,
      layoutOptions: { "elk.port.side": "EAST" },
    },
  ];
}

export function buildTaskGroupLayout(
  nodes: GroupLayoutNode[],
  dependencies: Dependency[],
  groupList: TaskGroup[],
) {
  const groups = new Map(groupList.map((group) => [group.id, group]));
  const containers = new Map<string, ElkNode>();
  const rows = new Map<
    string,
    { groupId: string; top: number; width: number; height: number }
  >();
  const children: ElkNode[] = [];
  for (const { id, groupId, width, height } of nodes) {
    if (!groups.has(groupId)) {
      children.push({ id, width, height });
      continue;
    }
    let container = containers.get(groupId);
    if (!container) {
      container = {
        id: groupId,
        width: groupWidth,
        height: headerHeight + bottomPadding,
        ports: [],
        layoutOptions: { "elk.portConstraints": "FIXED_POS" },
      };
      containers.set(groupId, container);
      children.push(container);
    }
    const top = (container.height ?? 0) - bottomPadding;
    rows.set(id, { groupId, top, width: groupWidth, height: rowHeight });
    container.height = top + rowHeight + bottomPadding;
    container.ports?.push(...groupPorts(id, groupWidth, top + rowHeight / 2));
  }

  // ELK はグループをひとつの矩形として置き、外部への依存は所属タスクの行に接続する。
  const edges = dependencies.map(({ blockerTaskId, blockedTaskId }) => ({
    id: dependencyEdgeId(blockerTaskId, blockedTaskId),
    sources: [
      rows.has(blockerTaskId) ? rowPort(blockerTaskId, "EAST") : blockerTaskId,
    ],
    targets: [
      rows.has(blockedTaskId) ? rowPort(blockedTaskId, "WEST") : blockedTaskId,
    ],
  }));
  return { groups, rows, children, edges };
}

export function readTaskGroupLayout(
  layout: ElkNode,
  { groups, rows }: ReturnType<typeof buildTaskGroupLayout>,
) {
  const frames: TaskGroupFrame[] = [];
  const positions = new Map<string, { x: number; y: number }>();
  for (const node of layout.children ?? []) {
    const position = { x: node.x ?? 0, y: node.y ?? 0 };
    positions.set(node.id, position);
    const group = groups.get(node.id);
    if (group) {
      frames.push({
        group,
        ...position,
        width: node.width ?? 0,
        height: node.height ?? 0,
      });
    }
  }
  for (const [id, row] of rows) {
    const origin = positions.get(row.groupId);
    if (origin) positions.set(id, { x: origin.x, y: origin.y + row.top });
  }
  return { frames, positions };
}
