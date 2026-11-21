import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  DocumentKind,
  TaskDisplayState,
  type Snapshot,
} from "../src/gen/nnx/v1/nnx_pb";
import { readHideCompletedTasks } from "../src/i18n/settings";
import { FeatureWorkspace } from "../src/views/FeatureWorkspace";
import type { GroupAction } from "../src/views/GraphTaskGroups";
import {
  makeDependency,
  makeDocument,
  makeFeature,
  makeProject,
  makePullRequest,
  makeSnapshot,
  makeTask,
  makeTaskGroup,
} from "./factories";

const workspaceMocks = vi.hoisted(() => ({
  navigate: vi.fn().mockResolvedValue(undefined),
  snapshot: { data: undefined as Snapshot | undefined, isPending: false },
  hookIndex: 0,
  groupId: undefined as string | undefined,
  taskId: undefined as string | undefined,
  mutations: Array.from({ length: 5 }, () => ({
    mutate: vi.fn(),
    mutateAsync: vi.fn().mockResolvedValue({}),
    isPending: false,
    error: null as Error | null,
  })),
  api: {
    deleteTaskGroup: vi.fn(),
    sync: vi.fn(),
    updateFeature: vi.fn(),
    deleteFeature: vi.fn(),
    addDocument: vi.fn(),
    deleteDocument: vi.fn(),
    getDocument: vi.fn(),
    updateDocument: vi.fn(),
  },
  previewDocument: {
    id: "preview-doc",
    kind: 3,
    title: "Preview document",
    locator: "docs/preview.md",
    isImplementationPlan: false,
  },
}));

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children, params }: { children: ReactNode; params?: unknown }) => (
    <a href="#test" data-params={JSON.stringify(params)}>
      {children}
    </a>
  ),
  useNavigate: () => workspaceMocks.navigate,
  useParams: () => ({ featureId: "feature-1" }),
  useSearch: () => ({
    group: workspaceMocks.groupId,
    task: workspaceMocks.taskId,
  }),
}));
vi.mock("../src/api", () => ({
  mutations: workspaceMocks.api,
}));
vi.mock("../src/hooks", () => ({
  useSnapshot: () => workspaceMocks.snapshot,
  useDomainMutation: (mutationFn: (input: unknown) => unknown) => {
    workspaceMocks.hookIndex++;
    const index =
      mutationFn === workspaceMocks.api.addDocument
        ? 1
        : mutationFn === workspaceMocks.api.deleteDocument
          ? 2
          : mutationFn === workspaceMocks.api.updateDocument
            ? 3
            : mutationFn === workspaceMocks.api.getDocument
              ? 4
              : 0;
    const mutation = workspaceMocks.mutations[index];
    if (!mutation) throw new Error("mutation mock missing");
    mutation.mutate.mockImplementation(
      (input: unknown, options?: { onSuccess?: (data: unknown) => void }) => {
        const result = mutationFn(input);
        if (options?.onSuccess) {
          void Promise.resolve(result).then(options.onSuccess);
        }
        return result;
      },
    );
    return mutation;
  },
}));
vi.mock("../src/views/FeatureGraph", () => ({
  FeatureGraph: ({
    tasks,
    groups,
    onGroupAction,
    hiddenDependencies,
    hiddenTaskCount,
    searching,
    onCreateTask,
    onEditTask,
    onPreviewDocument,
    onAddDocument,
    readOnly,
  }: {
    tasks: { id: string }[];
    groups?: GroupAction["group"][];
    onGroupAction?: (action: GroupAction) => void;
    hiddenDependencies: Map<string, { blockers: string[] }>;
    hiddenTaskCount: number;
    searching: boolean;
    onCreateTask: (dependency?: {
      taskId: string;
      direction: "blocks" | "blockedBy";
    }) => void;
    onEditTask: (taskId: string) => void;
    onPreviewDocument: (document: unknown) => void;
    onAddDocument: (taskId: string, trigger: HTMLButtonElement) => void;
    readOnly: boolean;
  }) => (
    <div data-testid="feature-graph">
      <span>{readOnly ? "Mock read-only graph" : "Mock active graph"}</span>
      {groups?.map((group) => (
        <div key={group.id}>
          <button
            onClick={() => onGroupAction?.({ type: "references", group })}
          >
            View references for {group.name}
          </button>
          <button onClick={() => onGroupAction?.({ type: "rename", group })}>
            Edit {group.name}
          </button>
          <button onClick={() => onGroupAction?.({ type: "delete", group })}>
            Delete {group.name}
          </button>
        </div>
      ))}
      <span data-testid="mock-graph-tasks">
        {tasks.map((task) => task.id).join(",")}
      </span>
      <span data-testid="mock-graph-hidden">{hiddenTaskCount}</span>
      <span data-testid="mock-graph-searching">{String(searching)}</span>
      <span data-testid="mock-graph-hidden-blockers">
        {[...hiddenDependencies]
          .map(([id, hidden]) => `${id}:${hidden.blockers.join("|")}`)
          .join(",")}
      </span>
      {!readOnly && (
        <button
          onClick={() => {
            onCreateTask();
          }}
        >
          Mock create task
        </button>
      )}
      {!readOnly && (
        <button
          onClick={() => {
            onCreateTask({ taskId: "task-1", direction: "blockedBy" });
          }}
        >
          Mock drop on empty space
        </button>
      )}
      {!readOnly && (
        <button
          onClick={(event) => {
            onAddDocument("task-1", event.currentTarget);
          }}
        >
          Mock add task reference
        </button>
      )}
      <button
        onClick={() => {
          onEditTask("task-1");
        }}
      >
        Mock edit task
      </button>
      <button
        onClick={() => {
          onPreviewDocument(workspaceMocks.previewDocument);
        }}
      >
        Mock preview document
      </button>
    </div>
  ),
}));
vi.mock("../src/views/DocumentDialog", () => ({
  DocumentDialog: ({
    taskId,
    onClose,
  }: {
    taskId: string;
    onClose: () => void;
  }) => (
    <div role="dialog" aria-label="Mock add document">
      <span>{taskId}</span>
      <button onClick={onClose}>Mock close document</button>
    </div>
  ),
}));
vi.mock("../src/views/TaskInspector", () => ({
  TaskInspector: ({
    onClose,
    readOnly,
  }: {
    onClose: () => void;
    readOnly: boolean;
  }) => (
    <aside aria-label="Mock task inspector">
      <span>
        {readOnly ? "Mock read-only inspector" : "Mock active inspector"}
      </span>
      <button onClick={onClose}>Mock close inspector</button>
    </aside>
  ),
}));
vi.mock("../src/views/MarkdownPreview", () => ({
  MarkdownPreview: ({
    document,
    onClose,
  }: {
    document: { title: string };
    onClose: () => void;
  }) => (
    <div role="dialog" aria-label="Mock Markdown preview">
      <p>{document.title}</p>
      <button onClick={onClose}>Mock close preview</button>
    </div>
  ),
}));
vi.mock("../src/views/CreateTaskDialog", () => ({
  CreateTaskDialog: ({
    onClose,
    dependency,
  }: {
    onClose: () => void;
    dependency?: {
      value: { taskId: string; direction: string };
      title: string;
    };
  }) => (
    <div role="dialog" aria-label="Mock create task">
      <span data-testid="mock-task-dependency">
        {dependency
          ? `${dependency.value.direction}:${dependency.title}`
          : "none"}
      </span>
      <button onClick={onClose}>Mock close task</button>
    </div>
  ),
}));
vi.mock("../src/views/BatchPromptDialog", () => ({
  BatchPromptDialog: ({
    featureId,
    onClose,
  }: {
    featureId: string;
    onClose: () => void;
  }) => (
    <div role="dialog" aria-label={`Mock batch prompt ${featureId}`}>
      <button onClick={onClose}>Mock close batch prompt</button>
    </div>
  ),
}));
vi.mock("../src/views/EditFeatureDialog", () => ({
  EditFeatureDialog: ({
    onClose,
    onDeleted,
  }: {
    onClose: () => void;
    onDeleted: () => void;
  }) => (
    <div role="dialog" aria-label="Mock edit feature">
      <button onClick={onClose}>Mock close feature edit</button>
      <button onClick={onDeleted}>Mock delete feature</button>
    </div>
  ),
}));

const feature = makeFeature({
  id: "feature-1",
  title: "Payments rollout",
  description: "Ship payments",
  archived: false,
});
const task = makeTask({ id: "task-1", featureId: "feature-1" });
const otherTask = makeTask({ id: "task-2", featureId: "other-feature" });

function populatedSnapshot() {
  return makeSnapshot({
    features: [feature],
    tasks: [task, otherTask],
    dependencies: [
      makeDependency({ blockerTaskId: task.id, blockedTaskId: task.id }),
      makeDependency({ blockerTaskId: otherTask.id, blockedTaskId: task.id }),
    ],
    pullRequests: [makePullRequest({ taskId: task.id })],
    documents: [
      makeDocument({ taskId: task.id }),
      makeDocument({ id: "feature-doc", featureId: feature.id, taskId: "" }),
    ],
  });
}

function mutationAt(index: number) {
  const mutation = workspaceMocks.mutations[index];
  if (!mutation) throw new Error(`mutation mock missing at ${index}`);
  return mutation;
}

describe("FeatureWorkspace", () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  beforeEach(() => {
    localStorage.clear();
    workspaceMocks.hookIndex = 0;
    workspaceMocks.snapshot.data = populatedSnapshot();
    workspaceMocks.snapshot.isPending = false;
    workspaceMocks.navigate.mockClear();
    for (const mutation of workspaceMocks.mutations) {
      mutation.mutate.mockReset();
      mutation.mutateAsync.mockReset();
      mutation.mutateAsync.mockResolvedValue({});
      mutation.isPending = false;
      mutation.error = null;
    }
  });

  it("hides completed tasks only while the header switch is on", () => {
    const open = makeTask({
      id: "open",
      featureId: "feature-1",
      title: "Ship API",
      displayState: TaskDisplayState.NOT_STARTED,
    });
    const done = makeTask({
      id: "done",
      featureId: "feature-1",
      title: "Migrate schema",
      displayState: TaskDisplayState.COMPLETED,
    });
    workspaceMocks.snapshot.data = makeSnapshot({
      features: [feature],
      tasks: [done, open],
      dependencies: [
        makeDependency({ blockerTaskId: done.id, blockedTaskId: open.id }),
      ],
    });
    render(<FeatureWorkspace />);

    const toggle = screen.getByRole("switch", { name: "Hide completed" });
    expect(toggle).not.toBeChecked();
    expect(screen.getByTestId("mock-graph-tasks")).toHaveTextContent(
      "done,open",
    );
    expect(screen.getByTestId("mock-graph-hidden")).toHaveTextContent("0");

    fireEvent.click(toggle);
    expect(toggle).toBeChecked();
    expect(screen.getByTestId("mock-graph-tasks")).toHaveTextContent("open");
    expect(screen.getByTestId("mock-graph-hidden")).toHaveTextContent("1");
    expect(screen.getByTestId("mock-graph-hidden-blockers")).toHaveTextContent(
      "open:Migrate schema",
    );

    fireEvent.click(toggle);
    expect(toggle).not.toBeChecked();
    expect(screen.getByTestId("mock-graph-tasks")).toHaveTextContent(
      "done,open",
    );
  });

  it("restores the header switch from browser-local settings on the next visit", () => {
    workspaceMocks.snapshot.data = makeSnapshot({
      features: [feature],
      tasks: [
        makeTask({
          id: "open",
          featureId: "feature-1",
          displayState: TaskDisplayState.NOT_STARTED,
        }),
        makeTask({
          id: "done",
          featureId: "feature-1",
          displayState: TaskDisplayState.COMPLETED,
        }),
      ],
    });
    render(<FeatureWorkspace />);

    fireEvent.click(screen.getByRole("switch", { name: "Hide completed" }));
    expect(readHideCompletedTasks()).toBe(true);

    cleanup();
    workspaceMocks.hookIndex = 0;
    render(<FeatureWorkspace />);

    expect(
      screen.getByRole("switch", { name: "Hide completed" }),
    ).toBeChecked();
    expect(screen.getByTestId("mock-graph-tasks")).toHaveTextContent("open");
  });

  it("closes the inspector when hiding takes the selected task off the canvas", () => {
    workspaceMocks.snapshot.data = makeSnapshot({
      features: [feature],
      tasks: [
        makeTask({
          id: "task-1",
          featureId: "feature-1",
          displayState: TaskDisplayState.MERGED,
        }),
      ],
    });
    render(<FeatureWorkspace />);

    fireEvent.click(screen.getByRole("button", { name: "Mock edit task" }));
    expect(
      screen.getByRole("complementary", { name: "Mock task inspector" }),
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole("switch", { name: "Hide completed" }));
    expect(
      screen.queryByRole("complementary", { name: "Mock task inspector" }),
    ).not.toBeInTheDocument();
  });

  describe("graph search", () => {
    const shipApi = makeTask({
      id: "task-1",
      featureId: "feature-1",
      title: "Ship API",
      displayState: TaskDisplayState.NOT_STARTED,
    });
    const migrate = makeTask({
      id: "migrate",
      featureId: "feature-1",
      title: "Migrate schema",
      displayState: TaskDisplayState.NOT_STARTED,
    });
    const announce = makeTask({
      id: "announce",
      featureId: "feature-1",
      title: "Announce release",
      displayState: TaskDisplayState.COMPLETED,
    });

    beforeEach(() => {
      workspaceMocks.snapshot.data = makeSnapshot({
        features: [feature],
        tasks: [shipApi, migrate, announce],
        dependencies: [
          makeDependency({ blockerTaskId: "migrate", blockedTaskId: "task-1" }),
        ],
      });
    });

    const graphTasks = () => screen.getByTestId("mock-graph-tasks").textContent;

    it("filters the canvas from the toolbar and restores it on Escape", async () => {
      render(<FeatureWorkspace />);

      fireEvent.click(screen.getByRole("button", { name: "Search tasks" }));
      const field = screen.getByRole("textbox", {
        name: "Search tasks on this graph",
      });
      expect(field).toHaveFocus();

      fireEvent.change(field, { target: { value: "migrate" } });
      await waitFor(() => {
        expect(graphTasks()).toBe("migrate");
      });
      expect(screen.getByTestId("mock-graph-searching")).toHaveTextContent(
        "true",
      );
      expect(screen.getByText("1 of 3 shown")).toBeInTheDocument();
      // 絞り込みで消えたタスクも、残ったノードに依存の文脈を残す。migrate が
      // ブロックしていた Ship API は、隠れたブロック対象として代表される。
      expect(
        screen.getByTestId("mock-graph-hidden-blockers"),
      ).toHaveTextContent("migrate:");

      fireEvent.keyDown(window, { key: "Escape" });
      expect(
        screen.queryByRole("textbox", { name: "Search tasks on this graph" }),
      ).not.toBeInTheDocument();
      expect(graphTasks()).toBe("task-1,migrate,announce");
      expect(
        screen.getByRole("button", { name: "Search tasks" }),
      ).toHaveFocus();
    });

    it("opens on Cmd+F and combines the filter with hiding completed tasks", async () => {
      render(<FeatureWorkspace />);

      fireEvent.click(screen.getByRole("switch", { name: "Hide completed" }));
      fireEvent.keyDown(window, { key: "f", metaKey: true });
      const field = screen.getByRole("textbox", {
        name: "Search tasks on this graph",
      });
      fireEvent.change(field, { target: { value: "SHIP" } });
      await waitFor(() => {
        expect(graphTasks()).toBe("task-1");
      });
      // Ship API を塞いでいた migrate は検索で消えるので、残った側にスタブが付く。
      expect(
        screen.getByTestId("mock-graph-hidden-blockers"),
      ).toHaveTextContent("task-1:Migrate schema");
      expect(screen.getByTestId("mock-graph-hidden")).toHaveTextContent("2");
    });

    it("explains an empty result through the graph instead of the hide toggle", async () => {
      render(<FeatureWorkspace />);

      fireEvent.keyDown(window, { key: "f", ctrlKey: true });
      fireEvent.change(
        screen.getByRole("textbox", { name: "Search tasks on this graph" }),
        { target: { value: "nothing here" } },
      );
      await waitFor(() => {
        expect(graphTasks()).toBe("");
      });
      expect(screen.getByTestId("mock-graph-searching")).toHaveTextContent(
        "true",
      );
    });

    it("closes the inspector when the filter takes the selected task away", async () => {
      render(<FeatureWorkspace />);

      fireEvent.click(screen.getByRole("button", { name: "Mock edit task" }));
      expect(
        screen.getByRole("complementary", { name: "Mock task inspector" }),
      ).toBeInTheDocument();

      fireEvent.keyDown(window, { key: "f", metaKey: true });
      fireEvent.change(
        screen.getByRole("textbox", { name: "Search tasks on this graph" }),
        { target: { value: "announce" } },
      );
      await waitFor(() => {
        expect(
          screen.queryByRole("complementary", { name: "Mock task inspector" }),
        ).not.toBeInTheDocument();
      });
    });

    it("leaves the browser shortcut alone while a dialog is in front", () => {
      render(<FeatureWorkspace />);

      fireEvent.click(screen.getByRole("button", { name: "Mock create task" }));
      const event = new KeyboardEvent("keydown", {
        key: "f",
        metaKey: true,
        bubbles: true,
        cancelable: true,
      });
      window.dispatchEvent(event);

      expect(event.defaultPrevented).toBe(false);
      expect(
        screen.queryByRole("textbox", { name: "Search tasks on this graph" }),
      ).not.toBeInTheDocument();
    });
  });

  it("coordinates active sync, feature management, task inspection, and previews", () => {
    render(<FeatureWorkspace />);

    const workspaceActions = document.querySelector(".workspace-actions");
    expect(workspaceActions).not.toBeNull();
    expect(
      within(workspaceActions as HTMLElement)
        .getAllByRole("button")
        .map(
          (button) => button.getAttribute("aria-label") ?? button.textContent,
        ),
    ).toEqual([
      "Search tasks",
      "References",
      "Refresh",
      "Copy batch prompt",
      "Add task",
      "Edit feature",
    ]);
    expect(
      screen.getByRole("heading", { name: "Payments rollout" }),
    ).toBeInTheDocument();
    const featureIdButton = screen.getByRole("button", {
      name: "Copy Feature ID",
    });
    expect(featureIdButton).toHaveTextContent("feature-1");
    // ヘッダーはサーバーが導出した状態を出すため、自動のままの feature は
    // 一覧と同じ表示になる。
    const status = document.querySelector(".status-badge");
    expect(status).toHaveTextContent("Active");
    expect(status).toHaveAttribute("title", "Feature status");
    expect(featureIdButton.querySelector("svg")).not.toBeInTheDocument();
    expect(screen.queryByText("Feature ID")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "References" })).toHaveAttribute(
      "aria-expanded",
      "false",
    );
    expect(
      screen.queryByRole("button", { name: "Add reference" }),
    ).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Refresh" }));
    expect(mutationAt(0).mutate).toHaveBeenCalledWith("feature-1");
    // batch プロンプトはヘッダーから開き、ワークスペースが表示中の feature を
    // 読む。
    fireEvent.click(screen.getByRole("button", { name: "Copy batch prompt" }));
    expect(
      screen.getByRole("dialog", { name: "Mock batch prompt feature-1" }),
    ).toBeInTheDocument();
    fireEvent.click(
      screen.getByRole("button", { name: "Mock close batch prompt" }),
    );
    expect(
      screen.queryByRole("dialog", { name: "Mock batch prompt feature-1" }),
    ).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Edit feature" }));
    expect(
      screen.getByRole("dialog", { name: "Mock edit feature" }),
    ).toBeInTheDocument();
    fireEvent.click(
      screen.getByRole("button", { name: "Mock close feature edit" }),
    );
    expect(
      screen.queryByRole("dialog", { name: "Mock edit feature" }),
    ).not.toBeInTheDocument();

    // ツールバーとグラフの空状態はどちらも依存なしで同じダイアログを開く。
    for (const name of ["Add task", "Mock create task"]) {
      fireEvent.click(screen.getByRole("button", { name }));
      expect(
        screen.getByRole("dialog", { name: "Mock create task" }),
      ).toBeInTheDocument();
      expect(screen.getByTestId("mock-task-dependency")).toHaveTextContent(
        "none",
      );
      fireEvent.click(screen.getByRole("button", { name: "Mock close task" }));
    }

    // グラフの空白ドロップは、相手のタイトル付きで同じダイアログを開く。
    fireEvent.click(
      screen.getByRole("button", { name: "Mock drop on empty space" }),
    );
    expect(screen.getByTestId("mock-task-dependency")).toHaveTextContent(
      "blockedBy:Build API",
    );
    // 背景のツールバーを押しても、開いているダイアログの依存は外れない。
    fireEvent.click(screen.getByRole("button", { name: "Add task" }));
    expect(screen.getByTestId("mock-task-dependency")).toHaveTextContent(
      "blockedBy:Build API",
    );
    fireEvent.click(screen.getByRole("button", { name: "Mock close task" }));
    expect(
      screen.queryByRole("dialog", { name: "Mock create task" }),
    ).not.toBeInTheDocument();
    fireEvent.click(
      screen.getByRole("button", { name: "Mock add task reference" }),
    );
    expect(
      screen.getByRole("dialog", { name: "Mock add document" }),
    ).toHaveTextContent("task-1");
    fireEvent.click(
      screen.getByRole("button", { name: "Mock close document" }),
    );
    fireEvent.click(screen.getByRole("button", { name: "Mock edit task" }));
    expect(
      screen.getByRole("complementary", { name: "Mock task inspector" }),
    ).toBeInTheDocument();
    fireEvent.click(
      screen.getByRole("button", { name: "Mock close inspector" }),
    );
    fireEvent.click(
      screen.getByRole("button", { name: "Mock preview document" }),
    );
    expect(
      screen.getByRole("dialog", { name: "Mock Markdown preview" }),
    ).toHaveTextContent("Preview document");
    fireEvent.click(screen.getByRole("button", { name: "Mock close preview" }));

    fireEvent.click(screen.getByRole("button", { name: "Edit feature" }));
    fireEvent.click(
      screen.getByRole("button", { name: "Mock delete feature" }),
    );
    expect(workspaceMocks.navigate).toHaveBeenCalledWith({ to: "/" });
  });

  it("renders loading, missing-feature, and sync-pending states", () => {
    workspaceMocks.snapshot.isPending = true;
    const { rerender } = render(<FeatureWorkspace />);
    expect(
      screen.getByRole("heading", { name: "Tracing feature graph…" }),
    ).toBeInTheDocument();

    workspaceMocks.snapshot.isPending = false;
    workspaceMocks.snapshot.data = makeSnapshot({ features: [] });
    rerender(<FeatureWorkspace />);
    expect(
      screen.getByRole("heading", { name: "Feature not found" }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Return to overview" }));
    expect(workspaceMocks.navigate).toHaveBeenCalledWith({ to: "/" });

    workspaceMocks.snapshot.data = populatedSnapshot();
    mutationAt(0).isPending = true;
    rerender(<FeatureWorkspace />);
    const syncing = screen.getByRole("button", { name: "Refresh" });
    expect(syncing).toBeDisabled();
    expect(syncing).toHaveClass("icon-button-busy");
    expect(syncing).toHaveAttribute("aria-busy", "true");
  });

  // 削除後はその feature に辿り着けた一覧へ戻る。読み取り専用ならプロジェクト
  // のアーカイブタブになる。
  it("makes archived workspaces read-only and returns deletion to the archive", () => {
    workspaceMocks.snapshot.data = makeSnapshot({
      features: [{ ...feature, archived: true, readOnly: true }],
      tasks: [task],
    });
    render(<FeatureWorkspace />);

    expect(screen.getByText("Archived · read-only")).toBeInTheDocument();
    expect(screen.getByText("Mock read-only graph")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Refresh" }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Mock create task" }),
    ).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "References" }));
    expect(screen.getByText("No references.")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Add reference" }),
    ).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Mock edit task" }));
    expect(screen.getByText("Mock read-only inspector")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Manage feature" }));
    fireEvent.click(
      screen.getByRole("button", { name: "Mock delete feature" }),
    );
    expect(workspaceMocks.navigate).toHaveBeenCalledWith({
      to: "/projects/$projectId",
      params: { projectId: "project-1" },
      search: { features: "archived" },
    });
  });

  it("returns deletion of a project member to that project's archive tab", () => {
    workspaceMocks.snapshot.data = makeSnapshot({
      projects: [makeProject({ id: "project-1", title: "Delivery platform" })],
      features: [
        { ...feature, projectId: "project-1", archived: true, readOnly: true },
      ],
      tasks: [task],
    });
    render(<FeatureWorkspace />);

    fireEvent.click(screen.getByRole("button", { name: "Manage feature" }));
    fireEvent.click(
      screen.getByRole("button", { name: "Mock delete feature" }),
    );
    expect(workspaceMocks.navigate).toHaveBeenCalledWith({
      to: "/projects/$projectId",
      params: { projectId: "project-1" },
      search: { features: "archived" },
    });
  });

  // feature 自体がアーカイブでなくても読み取り専用になりうる。対処は
  // プロジェクト側なので、サーバーが拒否する復元を出さず、プロジェクト名と
  // リンクを示す。
  it("attributes read-only state to an archived project and links to it", () => {
    workspaceMocks.snapshot.data = makeSnapshot({
      projects: [
        makeProject({
          id: "project-1",
          title: "Delivery platform",
          archived: true,
        }),
      ],
      features: [{ ...feature, projectId: "project-1", readOnly: true }],
      tasks: [task],
    });
    render(<FeatureWorkspace />);

    expect(
      screen.getByText("Project archived · read-only"),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        "This feature is read-only because Delivery platform is archived. Activate the project to edit or refresh it.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByText("Open project")).toBeInTheDocument();
    expect(screen.getByText("Delivery platform")).toBeInTheDocument();
    expect(screen.getByText("Mock read-only graph")).toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Refresh" }),
    ).not.toBeInTheDocument();
  });

  // 両方のフラグが立つこともあり、feature だけ復元しても読み取り専用のままな
  // ので、通知の内容はプロジェクト側が決め続ける。
  it("names the archived project even when the feature is archived too", () => {
    workspaceMocks.snapshot.data = makeSnapshot({
      projects: [
        makeProject({
          id: "project-1",
          title: "Delivery platform",
          archived: true,
        }),
      ],
      features: [
        { ...feature, projectId: "project-1", archived: true, readOnly: true },
      ],
      tasks: [task],
    });
    render(<FeatureWorkspace />);

    expect(
      screen.getByText("Project archived · read-only"),
    ).toBeInTheDocument();
    expect(screen.queryByText("Archived · read-only")).not.toBeInTheDocument();
    expect(screen.getByText("Open project")).toBeInTheDocument();
  });
});

it("opens rename and deletion from group cards belonging to this feature", () => {
  workspaceMocks.snapshot.data = makeSnapshot({
    features: [makeFeature()],
    taskGroups: [
      makeTaskGroup(),
      makeTaskGroup({
        id: "group-2",
        featureId: "feature-2",
        name: "Other group",
      }),
    ],
  });
  render(<FeatureWorkspace />);
  expect(
    screen.queryByRole("button", { name: "Task groups" }),
  ).not.toBeInTheDocument();
  expect(
    screen.queryByRole("button", { name: "Edit Other group" }),
  ).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: "Edit Checkout" }));
  const rename = screen.getByRole("dialog", { name: "Rename Checkout" });
  fireEvent.keyDown(window, { key: "Escape" });
  expect(rename).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole("button", { name: "Delete Checkout" }));
  const deletion = screen.getByRole("dialog", { name: "Delete Checkout" });
  fireEvent.click(within(deletion).getByRole("button", { name: "Cancel" }));
  expect(deletion).not.toBeInTheDocument();
  cleanup();
});

it("lists only the selected group's references and returns from preview to the list", async () => {
  workspaceMocks.snapshot.data = makeSnapshot({
    features: [makeFeature({ readOnly: true })],
    taskGroups: [
      makeTaskGroup(),
      makeTaskGroup({ id: "group-2", name: "Delivery" }),
    ],
    documents: [
      makeDocument({
        title: "Group notes",
        groupId: "group-1",
        taskId: "",
        kind: DocumentKind.MARKDOWN,
        locator: "",
      }),
      makeDocument({
        id: "doc-url",
        title: "Group runbook",
        groupId: "group-1",
        taskId: "",
      }),
      makeDocument({ id: "doc-2", title: "Task notes" }),
    ],
  });
  render(<FeatureWorkspace />);
  const emptyTrigger = screen.getByRole("button", {
    name: "View references for Delivery",
  });
  fireEvent.click(emptyTrigger);
  expect(
    screen.getByRole("dialog", { name: "Group references for Delivery" }),
  ).toHaveTextContent("No references.");
  fireEvent.keyDown(window, { key: "Escape" });

  const trigger = screen.getByRole("button", {
    name: "View references for Checkout",
  });
  trigger.focus();
  fireEvent.click(trigger);
  const dialog = screen.getByRole("dialog", {
    name: "Group references for Checkout",
  });
  expect(dialog).not.toHaveTextContent("Task notes");
  expect(
    within(dialog).getByRole("link", { name: /Group runbook/ }),
  ).toHaveAttribute("href", "https://example.com/runbook");
  expect(
    within(dialog).queryByRole("button", { name: "Edit Group notes" }),
  ).not.toBeInTheDocument();
  fireEvent.click(within(dialog).getByRole("button", { name: /Group notes/ }));
  expect(
    screen.getByRole("dialog", { name: "Mock Markdown preview" }),
  ).toHaveTextContent("Group notes");
  expect(dialog).toHaveAttribute("inert");
  fireEvent.click(screen.getByRole("button", { name: "Mock close preview" }));
  const close = within(dialog).getByRole("button", { name: "Close" });
  expect(close).toHaveFocus();
  fireEvent.click(close);
  await waitFor(() => {
    expect(trigger).toHaveFocus();
  });
  cleanup();
});

it("opens the selected group with a graph restricted to its tasks", () => {
  workspaceMocks.groupId = "group-1";
  workspaceMocks.snapshot.data = makeSnapshot({
    features: [makeFeature()],
    tasks: [makeTask({ groupId: "group-1" }), makeTask({ id: "task-2" })],
    taskGroups: [makeTaskGroup()],
    dependencies: [makeDependency()],
    documents: [
      makeDocument({ title: "Group notes", groupId: "group-1", taskId: "" }),
    ],
  });
  render(<FeatureWorkspace />);
  expect(screen.getByRole("heading", { name: "Checkout" })).toBeInTheDocument();
  expect(
    screen.getByRole("link", { name: "Back to feature" }),
  ).toBeInTheDocument();
  expect(screen.getByTestId("mock-graph-tasks")).toHaveTextContent(/^task-1$/);
  fireEvent.click(screen.getByRole("button", { name: "Group references" }));
  expect(
    screen.getByRole("region", { name: "Group references" }),
  ).toHaveTextContent("Group notes");
  cleanup();
  workspaceMocks.groupId = undefined;
});

it("opens a task linked from a grouped list and clears the link on close", () => {
  workspaceMocks.taskId = "task-1";
  workspaceMocks.snapshot.data = makeSnapshot({
    features: [makeFeature()],
    tasks: [makeTask({ groupId: "group-1" })],
    taskGroups: [makeTaskGroup()],
  });
  render(<FeatureWorkspace />);
  const inspector = screen.getByRole("complementary", {
    name: "Mock task inspector",
  });
  fireEvent.click(
    within(inspector).getByRole("button", { name: "Mock close inspector" }),
  );
  expect(inspector).not.toBeInTheDocument();
  expect(workspaceMocks.navigate).toHaveBeenCalledWith({
    to: "/features/$featureId",
    params: { featureId: "feature-1" },
    search: {},
    replace: true,
  });
  cleanup();
  workspaceMocks.taskId = undefined;
});
