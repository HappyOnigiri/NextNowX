import type { ElkNode } from "elkjs/lib/elk-api.js";
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

type TaskDependency = Pick<Dependency, "blockerTaskId" | "blockedTaskId">;

export type GraphDependency = TaskDependency & { summary?: true };

export function graphOverviewDependencies(
  tasks: Task[],
  dependencies: Dependency[],
  groups: TaskGroup[] | undefined,
): GraphDependency[] {
  if (!groups?.length) return dependencies;
  const groupIds = new Set(groups.map((group) => group.id));
  const byTask = new Map(
    tasks
      .filter((task) => groupIds.has(task.groupId))
      .map((task) => [task.id, task.groupId]),
  );
  const overview = new Map<string, GraphDependency>();
  for (const dependency of dependencies) {
    const blocker = byTask.get(dependency.blockerTaskId);
    const blocked = byTask.get(dependency.blockedTaskId);
    if (blocker !== undefined && blocker === blocked) continue;
    const blockerTaskId = blocker ?? dependency.blockerTaskId;
    const blockedTaskId = blocked ?? dependency.blockedTaskId;
    overview.set(
      dependencyEdgeId(blockerTaskId, blockedTaskId),
      blocker || blocked
        ? { blockerTaskId, blockedTaskId, summary: true }
        : dependency,
    );
  }
  return [...overview.values()];
}

export function buildTaskGroupLayout(
  nodes: GroupLayoutNode[],
  dependencies: GraphDependency[],
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
      };
      containers.set(groupId, container);
      children.push(container);
    }
    const top = (container.height ?? 0) - bottomPadding;
    rows.set(id, { groupId, top, width: groupWidth, height: rowHeight });
    container.height = top + rowHeight + bottomPadding;
  }

  for (const group of groupList) {
    if (!containers.has(group.id)) {
      children.push({
        id: group.id,
        width: groupWidth,
        height: headerHeight + bottomPadding,
      });
    }
  }

  const edges = dependencies.map(({ blockerTaskId, blockedTaskId }) => ({
    id: dependencyEdgeId(blockerTaskId, blockedTaskId),
    sources: [blockerTaskId],
    targets: [blockedTaskId],
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
