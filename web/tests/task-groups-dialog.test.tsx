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
import { TaskGroupDialog } from "../src/views/TaskGroupDialogs";
import { TaskGroupSelect } from "../src/views/TaskGroupSelect";
import { makeTaskGroup } from "./factories";

const mocks = vi.hoisted(() => ({
  createTaskGroup: vi.fn(),
  updateTaskGroup: vi.fn(),
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
  expect(
    screen.queryByRole("button", { name: "Delete group" }),
  ).not.toBeInTheDocument();
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
  mocks.updateTaskGroup
    .mockRejectedValueOnce(new Error("Could not rename the group"))
    .mockResolvedValue({ taskGroup: renamed });
  renderWithMutations(
    <TaskGroupDialog
      featureId={group.featureId}
      group={group}
      groups={[group, other]}
      onSaved={onSaved}
      onClose={vi.fn()}
    />,
  );
  const name = screen.getByRole("textbox", { name: "Group name" });
  const save = screen.getByRole("button", { name: "Save group" });
  expect(name).toHaveValue(group.name);
  fireEvent.change(name, { target: { value: other.name } });
  expect(save).toBeDisabled();
  expect(screen.getByRole("alert")).toHaveTextContent("already exists");
  fireEvent.change(name, { target: { value: "Launch" } });
  fireEvent.click(save);
  await waitFor(() => {
    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(save).toBeEnabled();
  });
  expect(name).toHaveValue("Launch");
  expect(onSaved).not.toHaveBeenCalled();
  fireEvent.click(save);
  await waitFor(() => {
    expect(mocks.updateTaskGroup).toHaveBeenCalledWith({
      id: group.id,
      name: "Launch",
    });
    expect(onSaved).toHaveBeenCalledExactlyOnceWith(renamed);
  });
});

it("validates color-only edits and saves a reset to the default color", async () => {
  const group = makeTaskGroup({ color: "#a1b2c3" });
  const onSaved = vi.fn();
  mocks.updateTaskGroup.mockResolvedValue({
    taskGroup: { ...group, color: "" },
  });
  renderWithMutations(
    <TaskGroupDialog
      featureId={group.featureId}
      group={group}
      groups={[group]}
      onSaved={onSaved}
      onClose={vi.fn()}
    />,
  );
  const color = screen.getByRole("textbox", { name: "Card color" });
  const save = screen.getByRole("button", { name: "Save group" });
  expect(color).toHaveValue(group.color);
  expect(save).toBeDisabled();
  fireEvent.change(color, { target: { value: "#1" } });
  expect(screen.getByRole("alert")).toHaveTextContent("#RRGGBB");
  expect(save).toBeDisabled();
  fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
  const discard = screen.getByRole("dialog", {
    name: "Discard unsaved changes?",
  });
  fireEvent.click(within(discard).getByRole("button", { name: "Cancel" }));
  expect(color).toHaveValue("#1");
  fireEvent.change(screen.getByLabelText("Choose a card color"), {
    target: { value: "#a13fd9" },
  });
  expect(color).toHaveValue("#a13fd9");
  expect(save).toBeEnabled();
  fireEvent.click(screen.getByRole("button", { name: "Reset" }));
  fireEvent.click(save);
  await waitFor(() => {
    expect(mocks.updateTaskGroup).toHaveBeenCalledExactlyOnceWith({
      id: group.id,
      color: "",
    });
    expect(onSaved).toHaveBeenCalledOnce();
  });
});

it("deletes from the edit dialog only after confirmation", async () => {
  const group = makeTaskGroup();
  const onClose = vi.fn();
  mocks.deleteTaskGroup.mockResolvedValue({});
  renderWithMutations(
    <TaskGroupDialog
      featureId={group.featureId}
      group={group}
      groups={[group]}
      onSaved={vi.fn()}
      onClose={onClose}
    />,
  );
  const draft = screen.getByRole("textbox", { name: "Group name" });
  fireEvent.change(draft, { target: { value: "Launch" } });
  const deleteGroup = screen.getByRole("button", { name: "Delete group" });
  fireEvent.click(deleteGroup);
  const dialog = screen.getByRole("dialog", { name: `Delete ${group.name}` });
  expect(dialog).toHaveTextContent("Group references will be deleted");
  fireEvent.click(within(dialog).getByRole("button", { name: "Cancel" }));
  expect(mocks.deleteTaskGroup).not.toHaveBeenCalled();
  expect(onClose).not.toHaveBeenCalled();
  expect(draft).toHaveValue("Launch");
  fireEvent.click(deleteGroup);
  fireEvent.click(
    within(
      screen.getByRole("dialog", { name: `Delete ${group.name}` }),
    ).getByRole("button", { name: "Delete group" }),
  );
  await waitFor(() => {
    expect(mocks.deleteTaskGroup).toHaveBeenCalledOnce();
    expect(mocks.deleteTaskGroup.mock.calls[0]?.[0]).toBe(group.id);
    expect(onClose).toHaveBeenCalledOnce();
  });
  expect(mocks.updateTaskGroup).not.toHaveBeenCalled();
});

it("offers references and editing on the group card without a delete button", () => {
  const group = makeTaskGroup();
  const onGroupAction = vi.fn();
  const frames = [{ group, x: 0, y: 0, width: 336, height: 48 }];
  const { rerender } = render(
    <GraphTaskGroups frames={frames} onGroupAction={onGroupAction} />,
  );
  fireEvent.click(screen.getByRole("button", { name: `Edit ${group.name}` }));
  expect(onGroupAction).toHaveBeenLastCalledWith({ type: "edit", group });
  expect(
    screen.queryByRole("button", { name: /Delete/ }),
  ).not.toBeInTheDocument();
  rerender(
    <GraphTaskGroups frames={frames} onGroupAction={onGroupAction} readOnly />,
  );
  expect(
    screen.queryByRole("button", { name: `Edit ${group.name}` }),
  ).not.toBeInTheDocument();
  fireEvent.click(
    screen.getByRole("button", { name: `Group references for ${group.name}` }),
  );
  expect(onGroupAction).toHaveBeenLastCalledWith({ type: "references", group });
});
