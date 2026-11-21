import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { GraphTaskGroups } from "../src/views/GraphTaskGroups";
import {
  DeleteTaskGroupDialog,
  TaskGroupNameDialog,
} from "../src/views/TaskGroupDialogs";
import { TaskGroupSelect } from "../src/views/TaskGroupSelect";
import { makeTaskGroup } from "./factories";

const mocks = vi.hoisted(() => ({
  createTaskGroup: vi.fn(),
  renameTaskGroup: vi.fn(),
  deleteTaskGroup: vi.fn(),
}));
vi.mock("../src/api", () => ({ mutations: mocks }));
vi.mock("@xyflow/react", () => ({
  ViewportPortal: ({ children }: { children: ReactNode }) => children,
}));
vi.mock("../src/views/TaskGroupLink", () => ({
  TaskGroupLink: () => null,
}));

beforeEach(() => {
  for (const mutation of Object.values(mocks)) mutation.mockReset();
});
afterEach(cleanup);

function renderWithMutations(component: ReactNode) {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      {component}
    </QueryClientProvider>,
  );
}

it("adds a group from the select menu and selects the trimmed result", async () => {
  const group = makeTaskGroup({ name: "Payments" });
  const selectGroup = vi.fn();
  mocks.createTaskGroup.mockResolvedValue({ taskGroup: group });
  renderWithMutations(
    <TaskGroupSelect
      createInFeature={group.featureId}
      groups={[]}
      value=""
      onChange={selectGroup}
    />,
  );
  const select = screen.getByRole("combobox", { name: "Task group" });
  fireEvent.change(select, { target: { value: "new" } });
  const name = screen.getByRole("textbox", { name: "Group name" });
  const create = screen.getByRole("button", { name: "Create group" });
  fireEvent.change(name, { target: { value: "   " } });
  expect(create).toBeDisabled();
  fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
  const discard = screen.getByRole("dialog", {
    name: "Discard unsaved changes?",
  });
  fireEvent.click(within(discard).getByRole("button", { name: "Cancel" }));
  expect(name).toHaveValue("   ");
  fireEvent.change(name, { target: { value: "  Payments  " } });
  fireEvent.click(create);
  await waitFor(() => {
    expect(mocks.createTaskGroup).toHaveBeenCalledWith({
      featureId: group.featureId,
      name: "Payments",
    });
    expect(selectGroup).toHaveBeenCalledExactlyOnceWith(group.id);
  });
  expect(name).not.toBeInTheDocument();
  fireEvent.change(select, { target: { value: "new" } });
  fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
  expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
});

it("renames a group and rejects duplicate names", async () => {
  const group = makeTaskGroup();
  const other = makeTaskGroup({ id: "group-2", name: "Delivery" });
  const onSaved = vi.fn();
  const renamed = { ...group, name: "Launch" };
  mocks.renameTaskGroup
    .mockRejectedValueOnce(new Error("Could not rename the group"))
    .mockResolvedValue({ taskGroup: renamed });
  renderWithMutations(
    <TaskGroupNameDialog
      featureId={group.featureId}
      group={group}
      groups={[group, other]}
      onSaved={onSaved}
      onClose={vi.fn()}
    />,
  );
  const name = screen.getByRole("textbox", { name: "Group name" });
  const rename = screen.getByRole("button", { name: "Rename group" });
  expect(name).toHaveValue(group.name);
  fireEvent.change(name, { target: { value: other.name } });
  expect(rename).toBeDisabled();
  expect(screen.getByRole("alert")).toHaveTextContent("already exists");
  fireEvent.change(name, { target: { value: "Launch" } });
  fireEvent.click(rename);
  await waitFor(() => {
    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(rename).toBeEnabled();
  });
  expect(name).toHaveValue("Launch");
  expect(onSaved).not.toHaveBeenCalled();
  fireEvent.click(rename);
  await waitFor(() => {
    expect(mocks.renameTaskGroup).toHaveBeenCalledWith({
      id: group.id,
      name: "Launch",
    });
    expect(onSaved).toHaveBeenCalledExactlyOnceWith(renamed);
  });
});

it("deletes only after confirmation", async () => {
  const group = makeTaskGroup();
  const onClose = vi.fn();
  mocks.deleteTaskGroup.mockResolvedValue({});
  renderWithMutations(
    <DeleteTaskGroupDialog group={group} onClose={onClose} />,
  );
  const dialog = screen.getByRole("dialog", { name: `Delete ${group.name}` });
  expect(dialog).toHaveTextContent("Group references will be deleted");
  fireEvent.click(within(dialog).getByRole("button", { name: "Cancel" }));
  expect(mocks.deleteTaskGroup).not.toHaveBeenCalled();
  fireEvent.click(within(dialog).getByRole("button", { name: "Delete group" }));
  await waitFor(() => {
    expect(mocks.deleteTaskGroup.mock.calls[0]?.[0]).toBe(group.id);
    expect(onClose).toHaveBeenCalledTimes(2);
  });
});

it("dispatches references, rename, and delete actions from the group card", () => {
  const group = makeTaskGroup();
  const onGroupAction = vi.fn();
  const frames = [{ group, x: 0, y: 0, width: 336, height: 48 }];
  const { rerender } = render(
    <GraphTaskGroups frames={frames} onGroupAction={onGroupAction} />,
  );
  fireEvent.click(screen.getByRole("button", { name: `Rename ${group.name}` }));
  expect(onGroupAction).toHaveBeenLastCalledWith({ type: "rename", group });
  fireEvent.click(screen.getByRole("button", { name: `Delete ${group.name}` }));
  expect(onGroupAction).toHaveBeenLastCalledWith({ type: "delete", group });
  rerender(
    <GraphTaskGroups frames={frames} onGroupAction={onGroupAction} readOnly />,
  );
  expect(
    screen.queryByRole("button", { name: `Rename ${group.name}` }),
  ).not.toBeInTheDocument();
  expect(
    screen.queryByRole("button", { name: `Delete ${group.name}` }),
  ).not.toBeInTheDocument();
  fireEvent.click(
    screen.getByRole("button", { name: `Group references for ${group.name}` }),
  );
  expect(onGroupAction).toHaveBeenLastCalledWith({ type: "references", group });
});
