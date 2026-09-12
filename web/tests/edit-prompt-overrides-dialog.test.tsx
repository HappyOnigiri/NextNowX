import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { FeatureStatus } from "../src/gen/prx/v1/prx_pb";
import { EditFeatureDialog } from "../src/views/EditFeatureDialog";
import { EditProjectDialog } from "../src/views/EditProjectDialog";
import { makeFeature, makeProject } from "./factories";

const dialogMocks = vi.hoisted(() => ({
  mutations: Array.from({ length: 4 }, () => ({
    mutateAsync: vi.fn(),
    isPending: false,
    error: null as Error | null,
  })),
  api: {
    updateProject: vi.fn(),
    deleteProject: vi.fn(),
    updateFeature: vi.fn(),
    deleteFeature: vi.fn(),
  },
  getPromptTemplates: vi.fn(),
}));

vi.mock("../src/api", () => ({
  mutations: {
    updateProject: dialogMocks.api.updateProject,
    deleteProject: dialogMocks.api.deleteProject,
    updateFeature: dialogMocks.api.updateFeature,
    deleteFeature: dialogMocks.api.deleteFeature,
  },
  getPromptTemplates: dialogMocks.getPromptTemplates,
}));
vi.mock("../src/hooks", () => ({
  useDomainMutation: (mutation: unknown) => {
    if (mutation === dialogMocks.api.updateProject)
      return dialogMocks.mutations[0];
    if (mutation === dialogMocks.api.deleteProject)
      return dialogMocks.mutations[1];
    if (mutation === dialogMocks.api.updateFeature)
      return dialogMocks.mutations[2];
    if (mutation === dialogMocks.api.deleteFeature)
      return dialogMocks.mutations[3];
    throw new Error("mutation mock missing");
  },
}));

const promptSettings = {
  design: "Global design {{task_id}}",
  implementation: "Global implementation {{task_id}}",
  batch: "Global batch {{task_list}}",
  supportedPlaceholders: ["task_id", "feature_id"],
  requiredPlaceholder: "task_id",
  batchSupportedPlaceholders: ["task_list", "feature_id"],
  batchRequiredPlaceholder: "task_list",
  builtIn: {
    design: "Built-in design {{task_id}}",
    implementation: "Built-in implementation {{task_id}}",
    batch: "Built-in batch {{task_list}}",
  },
};

describe("entity prompt override dialogs", () => {
  afterEach(cleanup);

  beforeEach(() => {
    dialogMocks.getPromptTemplates.mockReset();
    dialogMocks.getPromptTemplates.mockResolvedValue(promptSettings);
    for (const mutation of dialogMocks.mutations) {
      mutation.mutateAsync.mockReset();
      mutation.mutateAsync.mockResolvedValue({});
      mutation.isPending = false;
      mutation.error = null;
    }
  });

  it("saves project prompt changes from the Prompts tab", async () => {
    const onClose = vi.fn();
    render(
      <EditProjectDialog
        project={makeProject({ id: "project-1" })}
        referenceCount={0}
        onClose={onClose}
        onDeleted={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByRole("tab", { name: "Prompts" }));
    const designSource = await screen.findByRole("combobox", {
      name: "Source for Design prompt",
    });
    fireEvent.change(designSource, { target: { value: "override" } });
    fireEvent.change(screen.getByLabelText("Design prompt"), {
      target: { value: "Project design {{task_id}}" },
    });
    fireEvent.submit(screen.getByRole("form", { name: "Edit project" }));

    await waitFor(() => {
      expect(onClose).toHaveBeenCalledOnce();
    });
    expect(dialogMocks.mutations[0]?.mutateAsync).toHaveBeenCalledWith({
      id: "project-1",
      title: "Delivery platform",
      description: "",
      promptOverrides: { design: "Project design {{task_id}}" },
    });
  });

  it("saves feature prompt changes and reads a project's inherited value", async () => {
    const onClose = vi.fn();
    render(
      <EditFeatureDialog
        projects={[
          makeProject({
            id: "project-1",
            promptOverrides: {
              design: "Project design {{task_id}}",
            },
          }),
        ]}
        feature={makeFeature({ status: FeatureStatus.ACTIVE })}
        onClose={onClose}
        onDeleted={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByRole("tab", { name: "Prompts" }));
    const design = await screen.findByLabelText("Design prompt");
    expect(design).toHaveValue("Project design {{task_id}}");
    const batchSource = screen.getByRole("combobox", {
      name: "Source for Batch implementation prompt",
    });
    fireEvent.change(batchSource, { target: { value: "override" } });
    fireEvent.change(screen.getByLabelText("Batch implementation prompt"), {
      target: { value: "Feature batch {{task_list}}" },
    });
    fireEvent.submit(screen.getByRole("form", { name: "Edit feature" }));

    await waitFor(() => {
      expect(onClose).toHaveBeenCalledOnce();
    });
    expect(dialogMocks.mutations[2]?.mutateAsync).toHaveBeenCalledWith({
      id: "feature-1",
      title: "Payments rollout",
      description: "",
      status: FeatureStatus.ACTIVE,
      projectId: "project-1",
      promptOverrides: { batch: "Feature batch {{task_list}}" },
    });
  });

  it("shows prompt metadata for archived features without enabling edits", async () => {
    render(
      <EditFeatureDialog
        projects={[makeProject({ id: "project-1" })]}
        feature={makeFeature({ archived: true })}
        onClose={vi.fn()}
        onDeleted={vi.fn()}
      />,
    );

    fireEvent.click(screen.getByRole("tab", { name: "Prompts" }));
    const designSource = await screen.findByRole("combobox", {
      name: "Source for Design prompt",
    });
    expect(designSource).toBeDisabled();
    expect(screen.getByLabelText("Design prompt")).toBeDisabled();
  });
});
