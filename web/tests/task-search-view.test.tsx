import {
  cleanup,
  fireEvent,
  render,
  screen,
  within,
} from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { TaskDisplayState, type Snapshot } from "../src/gen/nnx/v1/nnx_pb";
import { TaskSearch } from "../src/views/TaskSearch";
import {
  makeFeature,
  makeProject,
  makePullRequest,
  makeSnapshot,
  makeTask,
  makeTaskGroup,
} from "./factories";

const mocks = vi.hoisted(() => ({
  query: "task-status:ready",
  navigate: vi.fn(),
  state: {
    isPending: false,
    error: null as Error | null,
    refetch: vi.fn(),
    data: undefined as Snapshot | undefined,
  },
}));

vi.mock("@tanstack/react-router", () => ({
  Link: ({ children }: { children: ReactNode }) => (
    <a href="#feature">{children}</a>
  ),
  useNavigate: () => mocks.navigate,
  useSearch: () => ({ q: mocks.query }),
}));
vi.mock("../src/hooks", () => ({
  useSnapshot: () => mocks.state,
}));

describe("TaskSearch view", () => {
  afterEach(cleanup);
  beforeEach(() => {
    mocks.query = "task-status:ready";
    mocks.navigate.mockReset();
    mocks.state.isPending = false;
    mocks.state.error = null;
    mocks.state.refetch.mockReset();
    mocks.state.data = makeSnapshot({
      projects: [makeProject()],
      features: [makeFeature({ projectId: "project-1" })],
      pullRequests: [
        makePullRequest({ taskId: "ready-task", syncError: "offline" }),
      ],
      tasks: [
        makeTask({
          id: "ready-task",
          title: "Ready task",
          displayState: TaskDisplayState.NOT_STARTED,
          ready: true,
        }),
        makeTask({
          id: "blocked-task",
          title: "Blocked task",
          displayState: TaskDisplayState.IN_PROGRESS,
          ready: false,
        }),
      ],
    });
  });

  it("renders active search results and keeps the query in navigation", () => {
    render(<TaskSearch />);

    expect(
      screen.getByRole("heading", { name: "Task search" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("textbox", { name: "Search active tasks" }),
    ).toHaveValue("task-status:ready");
    expect(screen.getByText("Ready task")).toBeInTheDocument();
    expect(screen.queryByText("Blocked task")).not.toBeInTheDocument();

    const input = screen.getByRole("textbox", { name: "Search active tasks" });
    fireEvent.change(input, { target: { value: "payments" } });
    fireEvent.submit(screen.getByRole("search"));
    expect(mocks.navigate).toHaveBeenCalledWith({
      to: "/tasks",
      search: { q: "payments" },
    });
  });

  // カードは概要画面と共通なので、検索結果は所属を示し、状態を書き直す
  // 代わりに紐づく pull request を GitHub で開く。
  it("names the owners of a result and links its pull request", () => {
    render(<TaskSearch />);

    expect(screen.getByText("Project").nextElementSibling).toHaveTextContent(
      "Delivery platform",
    );
    expect(screen.getByText("Feature").nextElementSibling).toHaveTextContent(
      "Payments rollout",
    );
    // ID は検索に貼ったりエージェントに渡したりするものなので、カードは
    // 名前の隣に ID を置き、クリックでコピーできるようにする。
    const identifier = screen.getAllByRole("button", {
      name: "Copy Task ID",
    })[0];
    expect(identifier).toHaveTextContent("ready-task");
    const pullRequest = screen.getByRole("link", { name: /acme\/nnx #42/ });
    expect(pullRequest).toHaveAttribute(
      "href",
      "https://github.com/acme/nnx/pull/42",
    );
    expect(screen.getByText("open")).toBeInTheDocument();
    // pull request 自身の異常はアイコンで示し、理由は tooltip と名前が持つ。
    expect(
      screen.getByRole("button", {
        name: "GitHub status refresh error: offline",
      }),
    ).toHaveClass("pr-flag", "is-sync-error");
  });

  it("groups only the filtered cards, keeping the first member's position", () => {
    const group = makeTaskGroup();
    const sameName = makeTaskGroup({ id: "group-2", featureId: "feature-2" });
    const empty = makeTaskGroup({ id: "group-3", name: "Empty group" });
    mocks.state.data = makeSnapshot({
      features: [makeFeature(), makeFeature({ id: sameName.featureId })],
      taskGroups: [group, sameName, empty],
      tasks: [
        makeTask({
          id: "hidden",
          title: "Hidden first member",
          groupId: group.id,
          ready: false,
        }),
        makeTask({ id: "loose-1", title: "Ungrouped first" }),
        makeTask({
          id: "grouped-1",
          title: "Grouped first",
          groupId: group.id,
        }),
        makeTask({ id: "loose-2", title: "Ungrouped second" }),
        makeTask({
          id: "grouped-2",
          title: "Grouped second",
          groupId: group.id,
        }),
        makeTask({
          id: "other",
          title: "Other feature",
          featureId: sameName.featureId,
          groupId: sameName.id,
        }),
      ],
    });
    render(<TaskSearch />);

    const results = screen.getByRole("list", { name: "Matching active tasks" });
    const [first, second, ...rest] = within(results).getAllByRole("list", {
      name: group.name,
    });
    if (!first || !second) throw new Error("Both group frames must be present");
    expect(rest).toHaveLength(0);
    expect(within(first).getAllByRole("listitem")).toHaveLength(2);
    expect(
      within(second).getByRole("link", { name: "Other feature" }),
    ).toBeInTheDocument();
    expect(results.children).toHaveLength(4);
    expect(results.children[0]).toHaveTextContent("Ungrouped first");
    expect(results.children[1]).toHaveTextContent(
      /Grouped first.*Grouped second/,
    );
    expect(results.children[2]).toHaveTextContent("Ungrouped second");
    expect(results.children[3]).toContainElement(second);
    expect(
      within(results).queryByText("Hidden first member"),
    ).not.toBeInTheDocument();
    expect(within(results).queryByText(empty.name)).not.toBeInTheDocument();
  });

  it("distinguishes invalid syntax from an empty result", () => {
    mocks.query = "github-status:waiting";
    const { rerender } = render(<TaskSearch />);
    expect(screen.getByRole("alert")).toHaveTextContent(
      "Search needs a correction",
    );

    mocks.query = "does-not-exist";
    rerender(<TaskSearch />);
    expect(
      screen.getByRole("heading", { name: "No matching active tasks" }),
    ).toBeInTheDocument();
  });

  it("offers a retry when the snapshot cannot be loaded", () => {
    mocks.state.error = new Error("snapshot unavailable");
    render(<TaskSearch />);

    expect(
      screen.getByRole("heading", {
        name: "Task search could not be loaded",
      }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    expect(mocks.state.refetch).toHaveBeenCalledOnce();
  });
});
