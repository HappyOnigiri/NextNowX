import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { TaskBlockLabel, TaskDisplayState } from "../src/gen/nnx/v1/nnx_pb";
import { useGraphLayout } from "../src/views/useGraphLayout";
import {
  makeDependency,
  makeDocument,
  makePullRequest,
  makeTask,
} from "./factories";

const layoutMocks = vi.hoisted(() => ({
  layout: vi.fn(),
  terminateWorker: vi.fn(),
}));

vi.mock("elkjs/lib/elk-api.js", () => ({
  default: class {
    constructor(options: { workerFactory?: (url?: string) => Worker }) {
      // ELK は constructor で worker を作るので、例外を投げる factory は
      // 実装と同じく constructor の失敗として現れる。
      options.workerFactory?.(undefined);
    }

    layout(...args: unknown[]) {
      return layoutMocks.layout(...args) as Promise<unknown>;
    }

    terminateWorker() {
      layoutMocks.terminateWorker();
    }
  },
}));

const workerStubs = {
  created: [] as StubWorker[],
  failCreation: false,
};

class StubWorker extends EventTarget {
  postMessage = vi.fn();
  terminate = vi.fn();

  constructor() {
    super();
    if (workerStubs.failCreation) throw new Error("worker script missing");
    workerStubs.created.push(this);
  }
}

vi.stubGlobal("Worker", StubWorker);

function lastCreatedWorker() {
  const worker = workerStubs.created.at(-1);
  if (!worker) throw new Error("no worker was created");
  return worker;
}

describe("useGraphLayout", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    workerStubs.created = [];
    workerStubs.failCreation = false;
  });

  it("builds positioned task nodes with pull requests, documents, and edges", async () => {
    layoutMocks.layout.mockResolvedValue({
      children: [{ id: "task-1", x: 120, y: 48 }],
    });
    const onEditTask = vi.fn();
    const onPreviewDocument = vi.fn();
    const onAddDocument = vi.fn();
    const document = makeDocument({
      id: "document-1",
      kind: 3,
      title: "Plan",
      locator: "docs/plan.md",
    });
    const options = {
      tasks: [makeTask({ assignee: "Carol" })],
      dependencies: [makeDependency()],
      pullRequests: new Map([["task-1", makePullRequest({ stale: true })]]),
      documentsByTask: new Map([["task-1", [document]]]),
      onEditTask,
      onPreviewDocument,
      onAddDocument,
    };
    const { result, unmount } = renderHook(() => useGraphLayout(options));

    await waitFor(() => {
      expect(result.current.nodes).toHaveLength(1);
    });
    const node = result.current.nodes[0];
    expect(node).toMatchObject({
      id: "task-1",
      type: "task",
      position: { x: 120, y: 48 },
      data: {
        title: "Build API",
        assignee: "Carol",
        stale: true,
        // フィーチャーの pull request が 1 つの owner に揃うので名前から落ちる。
        pullRequest: {
          label: "nnx #42",
          url: "https://github.com/acme/nnx/pull/42",
        },
        documents: [document],
      },
    });
    node?.data.onEdit();
    expect(onEditTask).toHaveBeenCalledWith("task-1");
    const trigger = globalThis.document.createElement("button");
    node?.data.onAddReference?.(trigger);
    expect(onAddDocument).toHaveBeenCalledWith("task-1", trigger);
    expect(layoutMocks.layout).toHaveBeenCalledWith(
      expect.objectContaining({
        edges: [
          {
            id: "task-1-task-2",
            sources: ["task-1"],
            targets: ["task-2"],
          },
        ],
      }),
    );

    unmount();
    expect(layoutMocks.terminateWorker).toHaveBeenCalledOnce();
  });

  it("keeps the owner in a node label when the feature spans owners", async () => {
    layoutMocks.layout.mockResolvedValue({
      children: [
        { id: "task-1", x: 0, y: 0 },
        { id: "task-2", x: 400, y: 0 },
      ],
    });
    const options = {
      tasks: [makeTask({ id: "task-1" }), makeTask({ id: "task-2" })],
      dependencies: [],
      pullRequests: new Map([
        ["task-1", makePullRequest({ taskId: "task-1" })],
        [
          "task-2",
          makePullRequest({ taskId: "task-2", owner: "other", number: 7n }),
        ],
      ]),
      documentsByTask: new Map(),
      onEditTask: vi.fn(),
      onPreviewDocument: vi.fn(),
    };
    const { result } = renderHook(() => useGraphLayout(options));

    await waitFor(() => {
      expect(result.current.nodes).toHaveLength(2);
    });
    expect(result.current.nodes[0]?.data.pullRequest?.label).toBe(
      "acme/nnx #42",
    );
    expect(result.current.nodes[1]?.data.pullRequest?.label).toBe(
      "other/nnx #7",
    );
  });

  it("carries hidden dependencies to the node that stands in for them", async () => {
    layoutMocks.layout.mockResolvedValue({
      children: [
        { id: "task-1", x: 0, y: 0 },
        { id: "task-2", x: 400, y: 0 },
      ],
    });
    const hidden = { blockers: ["Migrate schema"], blocked: [] };
    const options = {
      tasks: [makeTask({ id: "task-1" }), makeTask({ id: "task-2" })],
      dependencies: [],
      pullRequests: new Map(),
      documentsByTask: new Map(),
      hiddenDependencies: new Map([["task-1", hidden]]),
      onEditTask: vi.fn(),
      onPreviewDocument: vi.fn(),
    };
    const { result } = renderHook(() => useGraphLayout(options));

    await waitFor(() => {
      expect(result.current.nodes).toHaveLength(2);
    });
    expect(result.current.nodes[0]?.data.hiddenDependencies).toBe(hidden);
    expect(result.current.nodes[1]?.data).not.toHaveProperty(
      "hiddenDependencies",
    );
  });

  it("keeps ELK edge routes and assigns a distinct port to each endpoint", async () => {
    layoutMocks.layout.mockResolvedValue({
      children: [
        { id: "task-1", x: 12, y: 20 },
        { id: "task-2", x: 406, y: 80 },
      ],
      edges: [
        {
          id: "task-1-task-2",
          sources: ["task-1"],
          targets: ["task-2"],
          sections: [
            {
              id: "route",
              startPoint: { x: 296, y: 92 },
              bendPoints: [
                { x: 340, y: 92 },
                { x: 340, y: 154 },
              ],
              endPoint: { x: 406, y: 154 },
            },
          ],
        },
      ],
    });
    const options = {
      tasks: [makeTask(), makeTask({ id: "task-2" })],
      dependencies: [makeDependency()],
      pullRequests: new Map(),
      documentsByTask: new Map(),
      onEditTask: vi.fn(),
      onPreviewDocument: vi.fn(),
    };
    const { result } = renderHook(() => useGraphLayout(options));

    await waitFor(() => {
      expect(result.current.edgeRoutes.size).toBe(1);
    });

    expect(result.current.edgeRoutes.get("task-1-task-2")).toEqual({
      points: [
        { x: 296, y: 92 },
        { x: 340, y: 92 },
        { x: 340, y: 154 },
        { x: 406, y: 154 },
      ],
      sourcePortId: "task-1-task-2-source",
      sourcePortTop: 72,
      targetPortId: "task-1-task-2-target",
      targetPortTop: 74,
    });
    expect(result.current.nodes[0]?.data.outgoingPorts).toEqual([
      { id: "task-1-task-2-source", top: 72 },
    ]);
    expect(result.current.nodes[1]?.data.incomingPorts).toEqual([
      { id: "task-1-task-2-target", top: 74 },
    ]);
    const layoutInput: unknown = layoutMocks.layout.mock.calls[0]?.[0];
    if (
      !layoutInput ||
      typeof layoutInput !== "object" ||
      !("layoutOptions" in layoutInput)
    )
      throw new Error("ELK layout options missing");
    expect(layoutInput.layoutOptions).toMatchObject({
      "elk.edgeRouting": "ORTHOGONAL",
      "elk.layered.mergeEdges": "false",
    });
  });

  it("spaces disconnected task groups farther apart", async () => {
    layoutMocks.layout.mockResolvedValue({ children: [] });
    const options = {
      tasks: Array.from({ length: 5 }, (_, index) =>
        makeTask({ id: `task-${String(index + 1)}` }),
      ),
      dependencies: [
        makeDependency(),
        makeDependency({
          blockerTaskId: "task-3",
          blockedTaskId: "task-4",
        }),
      ],
      pullRequests: new Map(),
      documentsByTask: new Map(),
      onEditTask: vi.fn(),
      onPreviewDocument: vi.fn(),
    };
    renderHook(() => useGraphLayout(options));

    await waitFor(() => {
      expect(layoutMocks.layout).toHaveBeenCalledOnce();
    });

    const layoutInput: unknown = layoutMocks.layout.mock.calls[0]?.[0];
    if (
      !layoutInput ||
      typeof layoutInput !== "object" ||
      !("layoutOptions" in layoutInput)
    )
      throw new Error("ELK layout options missing");
    expect(layoutInput.layoutOptions).toMatchObject({
      "elk.spacing.componentComponent": "120",
    });
    expect(layoutInput).toMatchObject({
      children: [
        { id: "task-1" },
        { id: "task-2" },
        { id: "task-3" },
        { id: "task-4" },
        { id: "task-5" },
      ],
      edges: [
        { sources: ["task-1"], targets: ["task-2"] },
        { sources: ["task-3"], targets: ["task-4"] },
      ],
    });
  });

  it("gives every node the same size whatever its status", async () => {
    layoutMocks.layout.mockResolvedValue({ children: [] });
    const labels = [
      [],
      [TaskBlockLabel.CONFLICT],
      [TaskBlockLabel.CONFLICT, TaskBlockLabel.CI_FAILED],
      [
        TaskBlockLabel.CONFLICT,
        TaskBlockLabel.CI_FAILED,
        TaskBlockLabel.CHANGES_REQUESTED,
      ],
      [
        TaskBlockLabel.CONFLICT,
        TaskBlockLabel.CI_FAILED,
        TaskBlockLabel.CHANGES_REQUESTED,
        TaskBlockLabel.DEPENDENCY_UNRESOLVED,
      ],
    ];
    const options = {
      tasks: labels.map((blockLabels, index) =>
        makeTask({ id: `task-${String(index + 1)}`, blockLabels }),
      ),
      dependencies: [
        makeDependency({ blockerTaskId: "task-2", blockedTaskId: "task-3" }),
        makeDependency({ blockerTaskId: "task-1", blockedTaskId: "task-2" }),
      ],
      pullRequests: new Map([
        ["task-2", makePullRequest({ taskId: "task-2" })],
      ]),
      documentsByTask: new Map([
        [
          "task-3",
          [
            makeDocument({ id: "document-1" }),
            makeDocument({ id: "document-2" }),
            makeDocument({ id: "document-3" }),
          ],
        ],
      ]),
      onEditTask: vi.fn(),
      onPreviewDocument: vi.fn(),
      readOnly: true,
    };
    const { rerender } = renderHook(
      (props: typeof options) => useGraphLayout(props),
      { initialProps: options },
    );

    await waitFor(() => {
      expect(layoutMocks.layout).toHaveBeenCalledOnce();
    });
    const layoutInput = layoutMocks.layout.mock.calls[0]?.[0] as {
      children: { width: number; height: number }[];
    };
    expect(layoutInput.children).toHaveLength(5);
    // 高さは最も嵩む task-3 の 3 アセットに揃う。ブロックラベルの数は寸法に
    // 反映しない。
    for (const child of layoutInput.children)
      expect(child).toMatchObject({ width: 284, height: 196 + 3 * 34 });

    // 本丸の回帰ガード。ステータスが変わった同じタスク集合が、ELK へまったく
    // 同じ入力として届く。
    rerender({
      ...options,
      tasks: options.tasks.map((task, index) =>
        makeTask({
          id: task.id,
          blockLabels: index === 0 ? [TaskBlockLabel.CI_FAILED] : [],
          displayState: TaskDisplayState.IN_REVIEW,
        }),
      ),
    });
    await waitFor(() => {
      expect(layoutMocks.layout).toHaveBeenCalledTimes(2);
    });
    const second = layoutMocks.layout.mock.calls[1]?.[0] as {
      children: unknown;
      edges: unknown;
    };
    expect(second.children).toEqual(layoutInput.children);
    expect(second.edges).toEqual(
      (layoutMocks.layout.mock.calls[0]?.[0] as { edges: unknown }).edges,
    );
  });

  it("sizes nodes for the heaviest task in the feature", async () => {
    layoutMocks.layout.mockResolvedValue({ children: [] });

    async function layoutHeight(options: Parameters<typeof useGraphLayout>[0]) {
      layoutMocks.layout.mockClear();
      const { unmount } = renderHook(() => useGraphLayout(options));
      await waitFor(() => {
        expect(layoutMocks.layout).toHaveBeenCalledOnce();
      });
      const input = layoutMocks.layout.mock.calls[0]?.[0] as {
        children: { height: number }[];
      };
      unmount();
      return input.children.map((child) => child.height);
    }

    const tasks = [makeTask({ id: "task-1" }), makeTask({ id: "task-2" })];
    const bare = {
      tasks,
      dependencies: [],
      pullRequests: new Map(),
      documentsByTask: new Map(),
      onEditTask: vi.fn(),
      onPreviewDocument: vi.fn(),
      readOnly: true,
    };

    // 何も提げていないフィーチャーはアセットの行を取らない。
    expect(await layoutHeight(bare)).toEqual([196, 196]);
    // 読み取り専用でなければ参照の追加ボタンがどのノードにも並ぶ。
    expect(await layoutHeight({ ...bare, readOnly: false })).toEqual([
      230, 230,
    ]);
    // 最も嵩むタスクに全ノードが揃う。
    expect(
      await layoutHeight({
        ...bare,
        pullRequests: new Map([
          ["task-2", makePullRequest({ taskId: "task-2" })],
        ]),
        documentsByTask: new Map([
          ["task-2", [makeDocument({ id: "document-1" })]],
        ]),
      }),
    ).toEqual([264, 264]);
    // 4 行を超えるアセットはノードの中でスクロールするので、高さは頭打ちになる。
    expect(
      await layoutHeight({
        ...bare,
        documentsByTask: new Map([
          [
            "task-2",
            Array.from({ length: 9 }, (_, index) =>
              makeDocument({ id: `document-${String(index)}` }),
            ),
          ],
        ]),
      }),
    ).toEqual([332, 332]);
  });

  it("orders nodes and edges by task id without disturbing the caller", async () => {
    layoutMocks.layout.mockResolvedValue({ children: [] });
    const tasks = [
      makeTask({ id: "T-10" }),
      makeTask({ id: "T-2" }),
      makeTask({ id: "T-9" }),
      makeTask({ id: "T-1" }),
    ];
    const dependencies = [
      makeDependency({ blockerTaskId: "T-9", blockedTaskId: "T-10" }),
      makeDependency({ blockerTaskId: "T-1", blockedTaskId: "T-9" }),
      makeDependency({ blockerTaskId: "T-1", blockedTaskId: "T-2" }),
    ];
    const options = {
      tasks,
      dependencies,
      pullRequests: new Map(),
      documentsByTask: new Map(),
      onEditTask: vi.fn(),
      onPreviewDocument: vi.fn(),
    };
    renderHook(() => useGraphLayout(options));

    await waitFor(() => {
      expect(layoutMocks.layout).toHaveBeenCalledOnce();
    });
    const layoutInput = layoutMocks.layout.mock.calls[0]?.[0] as {
      children: { id: string }[];
      edges: { id: string }[];
      layoutOptions: Record<string, string>;
    };
    expect(layoutInput.children.map((child) => child.id)).toEqual([
      "T-1",
      "T-2",
      "T-9",
      "T-10",
    ]);
    expect(layoutInput.edges.map((edge) => edge.id)).toEqual([
      "T-1-T-2",
      "T-1-T-9",
      "T-9-T-10",
    ]);
    expect(layoutInput.layoutOptions).toMatchObject({
      "elk.layered.considerModelOrder.strategy": "NODES_AND_EDGES",
    });
    // 呼び出し側の配列はインスペクタや件数表示と共有されているので動かさない。
    expect(tasks.map((task) => task.id)).toEqual(["T-10", "T-2", "T-9", "T-1"]);
    expect(dependencies.map((dependency) => dependency.blockedTaskId)).toEqual([
      "T-10",
      "T-9",
      "T-2",
    ]);
  });

  it("reports layout errors and retries the layout", async () => {
    layoutMocks.layout
      .mockRejectedValueOnce(new Error("worker unavailable"))
      .mockResolvedValueOnce({ children: [] });
    const options = {
      tasks: [makeTask()],
      dependencies: [],
      pullRequests: new Map(),
      documentsByTask: new Map(),
      onEditTask: vi.fn(),
      onPreviewDocument: vi.fn(),
    };
    const { result, unmount } = renderHook(() => useGraphLayout(options));

    await waitFor(() => {
      expect(result.current.layoutError).toEqual({
        message: "worker unavailable",
      });
    });
    act(() => {
      result.current.retryLayout();
    });
    await waitFor(() => {
      expect(layoutMocks.layout).toHaveBeenCalledTimes(2);
    });
    expect(result.current.layoutError).toBeUndefined();
    unmount();
  });

  it("reports a failure when the worker cannot be created", async () => {
    workerStubs.failCreation = true;
    layoutMocks.layout.mockResolvedValue({ children: [] });
    const options = {
      tasks: [makeTask()],
      dependencies: [],
      pullRequests: new Map(),
      documentsByTask: new Map(),
      onEditTask: vi.fn(),
      onPreviewDocument: vi.fn(),
    };
    const { result, unmount } = renderHook(() => useGraphLayout(options));

    await waitFor(() => {
      expect(result.current.layoutError).toEqual({
        message: "worker script missing",
      });
    });
    expect(result.current.layoutPending).toBe(false);
    expect(layoutMocks.layout).not.toHaveBeenCalled();

    workerStubs.failCreation = false;
    act(() => {
      result.current.retryLayout();
    });
    await waitFor(() => {
      expect(layoutMocks.layout).toHaveBeenCalledOnce();
    });
    expect(result.current.layoutError).toBeUndefined();
    unmount();
  });

  it("reports a failure when the worker fails to load", async () => {
    // 読み込みに失敗した worker は応答しないので、ELK の promise は pending のまま。
    layoutMocks.layout
      .mockReturnValueOnce(new Promise(() => undefined))
      .mockResolvedValueOnce({ children: [] });
    const options = {
      tasks: [makeTask()],
      dependencies: [],
      pullRequests: new Map(),
      documentsByTask: new Map(),
      onEditTask: vi.fn(),
      onPreviewDocument: vi.fn(),
    };
    const { result, unmount } = renderHook(() => useGraphLayout(options));

    const worker = lastCreatedWorker();
    act(() => {
      worker.dispatchEvent(
        new ErrorEvent("error", { message: "Unexpected token '<'" }),
      );
    });

    await waitFor(() => {
      expect(result.current.layoutError).toEqual({
        message: "Unexpected token '<'",
      });
    });
    expect(result.current.layoutPending).toBe(false);

    act(() => {
      result.current.retryLayout();
    });
    await waitFor(() => {
      expect(layoutMocks.layout).toHaveBeenCalledTimes(2);
    });
    expect(result.current.layoutError).toBeUndefined();
    unmount();
  });

  it("reports a failure when a worker message cannot be deserialized", async () => {
    layoutMocks.layout.mockReturnValue(new Promise(() => undefined));
    const options = {
      tasks: [makeTask()],
      dependencies: [],
      pullRequests: new Map(),
      documentsByTask: new Map(),
      onEditTask: vi.fn(),
      onPreviewDocument: vi.fn(),
    };
    const { result, unmount } = renderHook(() => useGraphLayout(options));

    const worker = lastCreatedWorker();
    act(() => {
      worker.dispatchEvent(new MessageEvent("messageerror"));
    });

    await waitFor(() => {
      expect(result.current.layoutError).toEqual({ message: undefined });
    });
    expect(result.current.layoutPending).toBe(false);
    unmount();
  });
});
