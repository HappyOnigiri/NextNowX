import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { TaskGroupsDialog } from "../src/views/TaskGroupsDialog";
import { makeTaskGroup } from "./factories";

const mocks = vi.hoisted(() => ({
  createTaskGroup: vi.fn(),
  renameTaskGroup: vi.fn(),
  deleteTaskGroup: vi.fn(),
}));

vi.mock("../src/api", () => ({ mutations: mocks }));

beforeEach(() => {
  for (const mutation of Object.values(mocks)) {
    mutation.mockReset();
    mutation.mockResolvedValue({});
  }
});
afterEach(cleanup);

function renderManager() {
  const first = makeTaskGroup();
  const second = makeTaskGroup({ id: "group-2", name: "Delivery" });
  const onClose = vi.fn();
  render(
    <QueryClientProvider client={new QueryClient()}>
      <TaskGroupsDialog
        featureId={first.featureId}
        groups={[first, second]}
        onClose={onClose}
      />
    </QueryClientProvider>,
  );
  return { first, second, onClose };
}

it("creates a trimmed group and edits a name without allowing a duplicate", async () => {
  const { first, second } = renderManager();
  const newName = screen.getByRole("textbox", { name: "New group name" });
  const create = screen.getByRole("button", { name: "Create group" });
  fireEvent.change(newName, { target: { value: "   " } });
  expect(create).toBeDisabled();
  fireEvent.change(newName, { target: { value: "  Payments  " } });
  fireEvent.click(create);
  await waitFor(() => {
    expect(mocks.createTaskGroup).toHaveBeenCalledWith({
      featureId: first.featureId,
      name: "Payments",
    });
    expect(newName).toHaveValue("");
  });
  fireEvent.click(screen.getByRole("button", { name: `Rename ${first.name}` }));
  const rename = screen.getByRole("button", { name: "Rename group" });
  const name = screen.getByRole("textbox", { name: "Group name" });
  expect(name).toHaveValue(first.name);
  fireEvent.change(name, { target: { value: second.name } });
  expect(rename).toBeDisabled();
  expect(screen.getByRole("alert")).toHaveTextContent("already exists");
  fireEvent.change(name, { target: { value: "Launch" } });
  fireEvent.click(rename);
  await waitFor(() => {
    expect(mocks.renameTaskGroup).toHaveBeenCalledWith({
      id: first.id,
      name: "Launch",
    });
    expect(screen.getByRole("textbox", { name: "New group name" })).toHaveValue(
      "",
    );
  });
});

it("keeps a draft when discard is cancelled and deletes only after confirmation", async () => {
  const { first, onClose } = renderManager();
  fireEvent.change(screen.getByRole("textbox", { name: "New group name" }), {
    target: { value: "Draft" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Close" }));
  const discard = screen.getByRole("dialog", {
    name: "Discard unsaved changes?",
  });
  fireEvent.click(within(discard).getByRole("button", { name: "Cancel" }));
  expect(screen.getByRole("textbox", { name: "New group name" })).toHaveValue(
    "Draft",
  );
  expect(onClose).not.toHaveBeenCalled();
  fireEvent.click(screen.getByRole("button", { name: "Cancel" }));

  const deleteButton = screen.getByRole("button", {
    name: `Delete ${first.name}`,
  });
  fireEvent.click(deleteButton);
  const deletion = screen.getByRole("dialog", { name: `Delete ${first.name}` });
  fireEvent.click(within(deletion).getByRole("button", { name: "Cancel" }));
  expect(mocks.deleteTaskGroup).not.toHaveBeenCalled();
  fireEvent.click(deleteButton);
  fireEvent.click(
    within(
      screen.getByRole("dialog", { name: `Delete ${first.name}` }),
    ).getByRole("button", { name: "Delete group" }),
  );
  await waitFor(() => {
    expect(mocks.deleteTaskGroup.mock.calls[0]?.[0]).toBe(first.id);
    expect(deletion).not.toBeInTheDocument();
  });
});
