import { describe, expect, it } from "vitest";
import { TaskDisplayState, type Task } from "../src/gen/prx/v1/prx_pb";
import {
  batchCandidates,
  isSelectable,
  prunedSelection,
} from "../src/views/batchPromptTasks";
import { makeTask } from "./factories";

const designed = {
  displayState: TaskDisplayState.DESIGNED,
  hasImplementationPlan: true,
  ready: true,
};

function tasks(): Task[] {
  return [
    makeTask({ id: "task-1", title: "Build API", ...designed }),
    makeTask({ id: "task-2", title: "Draft the schema" }),
    makeTask({
      id: "task-3",
      title: "Sketch the flow",
      displayState: TaskDisplayState.DESIGNING,
    }),
    makeTask({
      id: "task-4",
      title: "Close the books",
      displayState: TaskDisplayState.COMPLETED,
      ready: false,
    }),
    makeTask({
      id: "task-5",
      title: "Ship the change",
      displayState: TaskDisplayState.NOT_STARTED,
      ready: false,
      pendingBlockerTaskIds: ["task-2"],
    }),
  ];
}

function ids(
  kind: "design" | "implementation",
  options?: Partial<{
    includeBlocked: boolean;
    includeDesigned: boolean;
    includeUndesigned: boolean;
  }>,
): string[] {
  return batchCandidates(tasks(), {
    kind,
    includeBlocked: options?.includeBlocked ?? false,
    includeDesigned: options?.includeDesigned ?? false,
    includeUndesigned: options?.includeUndesigned ?? false,
  }).map((candidate) => candidate.task.id);
}

describe("batchCandidates", () => {
  // 実装タブのベース集合は従来どおり設計済みのタスクだけ。
  it("keeps the implementation tab on the designed tasks", () => {
    expect(ids("implementation")).toEqual(["task-1"]);
  });

  // 設計タブは実装計画がまだないタスクを扱う。決着した表示状態は出さない。
  it("offers the undesigned tasks on the design tab", () => {
    expect(ids("design")).toEqual(["task-2", "task-3"]);
  });

  it("adds the designed tasks to the design tab on request", () => {
    expect(ids("design", { includeDesigned: true })).toEqual([
      "task-1",
      "task-2",
      "task-3",
    ]);
  });

  // 設計済みを加えるかどうかは実装タブには効かない。実装タブのベース集合は
  // もともと設計済みのタスクだけである。
  it("ignores the designed toggle on the implementation tab", () => {
    expect(ids("implementation", { includeDesigned: true })).toEqual([
      "task-1",
    ]);
  });

  it("adds the undesigned tasks to the implementation tab on request", () => {
    expect(ids("implementation", { includeUndesigned: true })).toEqual([
      "task-1",
      "task-2",
      "task-3",
    ]);
  });

  // 実装計画がないものを加えるトグルは設計タブには効かない。設計タブは
  // もともとそれらを扱う。
  it("ignores the undesigned toggle on the design tab", () => {
    expect(ids("design", { includeUndesigned: true })).toEqual([
      "task-2",
      "task-3",
    ]);
  });

  // 依存の扱いは設計でも実装でも同じ。設計の順序にも blocker の計画が要る。
  it("orders a blocked design candidate behind its blocker", () => {
    const candidates = batchCandidates(tasks(), {
      kind: "design",
      includeBlocked: true,
      includeDesigned: false,
      includeUndesigned: false,
    });
    expect(candidates.map((candidate) => candidate.task.id)).toEqual([
      "task-2",
      "task-3",
      "task-5",
    ]);
    const blocked = candidates[2];
    if (!blocked) throw new Error("the blocked candidate is missing");
    expect(blocked.pendingBlockerIds).toEqual(["task-2"]);
    expect(isSelectable(blocked, new Set())).toBe(false);
    expect(isSelectable(blocked, new Set(["task-2"]))).toBe(true);
  });

  // ブロッカーを外すと、その上に積んだ作業も選択から落ちる。
  it("prunes a selection that lost its blocker", () => {
    const candidates = batchCandidates(tasks(), {
      kind: "design",
      includeBlocked: true,
      includeDesigned: false,
      includeUndesigned: false,
    });
    const kept = prunedSelection(candidates, new Set(["task-5"]));
    expect([...kept]).toEqual([]);
    const both = prunedSelection(candidates, new Set(["task-2", "task-5"]));
    expect([...both].sort()).toEqual(["task-2", "task-5"]);
  });
});
