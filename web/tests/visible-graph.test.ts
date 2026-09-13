import { describe, expect, it } from "vitest";
import { TaskDisplayState } from "../src/gen/prx/v1/prx_pb";
import {
  emptyHiddenDependencies,
  hideTasks,
  isFinishedTask,
} from "../src/views/visibleGraph";
import { makeDependency, makeTask } from "./factories";

const notStarted = TaskDisplayState.NOT_STARTED;

// 完了済みの非表示は述語 1 つ分の違いなので、既存の挙動はこの薄い包みで確かめる。
function hideFinishedTasks(
  tasks: Parameters<typeof hideTasks>[0],
  dependencies: Parameters<typeof hideTasks>[1],
) {
  return hideTasks(tasks, dependencies, isFinishedTask);
}

describe("hideTasks", () => {
  it("keeps the given arrays when nothing is finished", () => {
    const tasks = [makeTask({ id: "task-1", displayState: notStarted })];
    const dependencies = [makeDependency()];
    const visible = hideFinishedTasks(tasks, dependencies);
    expect(visible.tasks).toBe(tasks);
    expect(visible.dependencies).toBe(dependencies);
    expect(visible.hiddenDependencies).toBe(emptyHiddenDependencies);
  });

  it.each([
    TaskDisplayState.COMPLETED,
    TaskDisplayState.CLOSED,
    TaskDisplayState.MERGED,
  ])("hides a task presented as %s", (displayState) => {
    const visible = hideFinishedTasks(
      [
        makeTask({ id: "done", displayState }),
        makeTask({ id: "open", displayState: notStarted }),
      ],
      [],
    );
    expect(visible.tasks.map((task) => task.id)).toEqual(["open"]);
  });

  it.each([
    TaskDisplayState.APPROVED,
    TaskDisplayState.IN_REVIEW,
    TaskDisplayState.IN_PROGRESS,
  ])("keeps a task presented as %s", (displayState) => {
    const visible = hideFinishedTasks(
      [
        makeTask({ id: "open", displayState }),
        makeTask({ id: "done", displayState: TaskDisplayState.COMPLETED }),
      ],
      [],
    );
    expect(visible.tasks.map((task) => task.id)).toEqual(["open"]);
  });

  it("reports a chain of hidden blockers on the task that waited on it", () => {
    const visible = hideFinishedTasks(
      [
        makeTask({
          id: "a",
          title: "Design schema",
          displayState: TaskDisplayState.COMPLETED,
        }),
        makeTask({
          id: "b",
          title: "Migrate schema",
          displayState: TaskDisplayState.MERGED,
        }),
        makeTask({ id: "c", title: "Ship API", displayState: notStarted }),
      ],
      [
        makeDependency({ blockerTaskId: "a", blockedTaskId: "b" }),
        makeDependency({ blockerTaskId: "b", blockedTaskId: "c" }),
      ],
    );
    expect(visible.tasks.map((task) => task.id)).toEqual(["c"]);
    expect(visible.dependencies).toEqual([]);
    expect(visible.hiddenDependencies.get("c")).toEqual({
      blockers: ["Migrate schema", "Design schema"],
      blocked: [],
    });
  });

  it("reports hidden dependents and keeps the edges between visible tasks", () => {
    const visible = hideFinishedTasks(
      [
        makeTask({ id: "a", title: "Build API", displayState: notStarted }),
        makeTask({ id: "b", title: "Wire UI", displayState: notStarted }),
        makeTask({
          id: "c",
          title: "Announce",
          displayState: TaskDisplayState.COMPLETED,
        }),
      ],
      [
        makeDependency({ blockerTaskId: "a", blockedTaskId: "b" }),
        makeDependency({ blockerTaskId: "a", blockedTaskId: "c" }),
      ],
    );
    expect(visible.dependencies.map((item) => item.blockedTaskId)).toEqual([
      "b",
    ]);
    expect(visible.hiddenDependencies.get("a")).toEqual({
      blockers: [],
      blocked: ["Announce"],
    });
    expect(visible.hiddenDependencies.has("b")).toBe(false);
  });

  it("stops at a visible task instead of counting it as hidden", () => {
    const visible = hideFinishedTasks(
      [
        makeTask({ id: "a", title: "Build API", displayState: notStarted }),
        makeTask({
          id: "b",
          title: "Migrate",
          displayState: TaskDisplayState.COMPLETED,
        }),
        makeTask({ id: "c", title: "Ship", displayState: notStarted }),
        makeTask({ id: "d", title: "Later", displayState: notStarted }),
      ],
      [
        makeDependency({ blockerTaskId: "a", blockedTaskId: "b" }),
        makeDependency({ blockerTaskId: "b", blockedTaskId: "c" }),
        makeDependency({ blockerTaskId: "c", blockedTaskId: "d" }),
      ],
    );
    expect(visible.hiddenDependencies.get("c")).toEqual({
      blockers: ["Migrate"],
      blocked: [],
    });
    expect(visible.hiddenDependencies.has("d")).toBe(false);
  });

  it("merges two hiding reasons into one chain of hidden blockers", () => {
    const visible = hideTasks(
      [
        makeTask({
          id: "a",
          title: "Design schema",
          displayState: TaskDisplayState.COMPLETED,
        }),
        makeTask({
          id: "b",
          title: "Migrate schema",
          displayState: notStarted,
        }),
        makeTask({ id: "c", title: "Ship API", displayState: notStarted }),
      ],
      [
        makeDependency({ blockerTaskId: "a", blockedTaskId: "b" }),
        makeDependency({ blockerTaskId: "b", blockedTaskId: "c" }),
      ],
      // 完了済みと検索の不一致を 1 つの述語に束ねても、間に挟まった連鎖は
      // 残ったノード 1 つにまとめて代表される。
      (task) => isFinishedTask(task) || task.id === "b",
    );
    expect(visible.tasks.map((task) => task.id)).toEqual(["c"]);
    expect(visible.hiddenDependencies.get("c")).toEqual({
      blockers: ["Migrate schema", "Design schema"],
      blocked: [],
    });
  });

  it("ignores a dependency that names a task outside the feature", () => {
    const visible = hideFinishedTasks(
      [makeTask({ id: "open", displayState: notStarted })],
      [makeDependency({ blockerTaskId: "elsewhere", blockedTaskId: "open" })],
    );
    expect(visible.tasks).toHaveLength(1);
    expect(visible.hiddenDependencies).toBe(emptyHiddenDependencies);
  });
});
