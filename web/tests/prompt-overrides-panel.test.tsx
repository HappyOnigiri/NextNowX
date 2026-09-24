import { create } from "@bufbuild/protobuf";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { PromptTemplateSettings } from "../src/api";
import {
  PromptTemplateOverridesSchema,
  PromptTemplatesSchema,
} from "../src/gen/nnx/v1/nnx_pb";
import { PromptOverridesPanel } from "../src/views/PromptOverridesPanel";

const apiMocks = vi.hoisted(() => ({
  getPromptTemplates: vi.fn(),
}));

vi.mock("../src/api", () => ({
  getPromptTemplates: apiMocks.getPromptTemplates,
}));

const settings: PromptTemplateSettings = {
  ...create(PromptTemplatesSchema, {
    design: "Global design {{task_id}}",
    implementation: "Global implementation {{task_id}}",
    batch: "Global batch {{task_list}}",
  }),
  supportedPlaceholders: ["task_id", "feature_id"],
  requiredPlaceholder: "task_id",
  batchSupportedPlaceholders: ["task_list", "feature_id"],
  batchRequiredPlaceholder: "task_list",
  builtIn: create(PromptTemplatesSchema, {
    design: "Built-in design {{task_id}}",
    implementation: "Built-in implementation {{task_id}}",
    batch: "Built-in batch {{task_list}}",
  }),
};

describe("PromptOverridesPanel", () => {
  afterEach(cleanup);

  beforeEach(() => {
    apiMocks.getPromptTemplates.mockReset();
    apiMocks.getPromptTemplates.mockResolvedValue(settings);
  });

  it("loads inherited values and keeps an override draft while switching modes", async () => {
    const onStateChange = vi.fn();
    render(
      <PromptOverridesPanel
        scope="feature"
        overrides={create(PromptTemplateOverridesSchema, {
          design: "Feature design {{task_id}}",
          implementation: "",
          batch: "",
        })}
        parentOverrides={create(PromptTemplateOverridesSchema, {
          design: "Project design {{task_id}}",
          implementation: "Project implementation {{task_id}}",
          batch: "Project batch {{task_list}}",
        })}
        editable
        onStateChange={onStateChange}
      />,
    );

    await waitFor(() => {
      expect(screen.getByLabelText("Design prompt")).toHaveValue(
        "Feature design {{task_id}}",
      );
    });
    expect(screen.getByLabelText("Implementation prompt")).toHaveValue(
      "Project implementation {{task_id}}",
    );
    expect(screen.getByLabelText("Batch implementation prompt")).toHaveValue(
      "Project batch {{task_list}}",
    );
    expect(onStateChange).toHaveBeenCalledWith(
      {
        design: "Feature design {{task_id}}",
        implementation: "",
        batch: "",
        batchDesign: "",
      },
      true,
      false,
    );

    const designSource = screen.getByRole("combobox", {
      name: "Source for Design prompt",
    });
    fireEvent.change(designSource, { target: { value: "inherit" } });
    expect(screen.getByLabelText("Design prompt")).toBeDisabled();
    expect(screen.getByLabelText("Design prompt")).toHaveValue(
      "Project design {{task_id}}",
    );
    fireEvent.change(designSource, { target: { value: "override" } });
    expect(screen.getByLabelText("Design prompt")).toHaveValue(
      "Feature design {{task_id}}",
    );

    const implementationSource = screen.getByRole("combobox", {
      name: "Source for Implementation prompt",
    });
    fireEvent.change(implementationSource, {
      target: { value: "override" },
    });
    const implementation = screen.getByLabelText("Implementation prompt");
    expect(implementation).toHaveValue("Project implementation {{task_id}}");
    fireEvent.change(implementation, { target: { value: "" } });
    fireEvent.change(implementationSource, { target: { value: "inherit" } });
    fireEvent.change(implementationSource, { target: { value: "override" } });
    expect(implementation).toHaveValue("");
    fireEvent.change(implementation, {
      target: { value: "Invalid {{unknown}}" },
    });
    expect(
      screen.getByText("Unsupported placeholder {{unknown}}."),
    ).toBeInTheDocument();
    fireEvent.change(implementation, {
      target: { value: "Feature implementation {{task_id}}" },
    });
    expect(
      screen.queryByText("Unsupported placeholder {{unknown}}."),
    ).not.toBeInTheDocument();

    const batchSource = screen.getByRole("combobox", {
      name: "Source for Batch implementation prompt",
    });
    fireEvent.change(batchSource, { target: { value: "override" } });
    expect(screen.getByLabelText("Batch implementation prompt")).toHaveValue(
      "Project batch {{task_list}}",
    );
    expect(onStateChange).toHaveBeenLastCalledWith(
      {
        design: "Feature design {{task_id}}",
        implementation: "Feature implementation {{task_id}}",
        batch: "Project batch {{task_list}}",
        batchDesign: "",
      },
      true,
      false,
    );
  });

  it("reports metadata failures without making the panel editable", async () => {
    apiMocks.getPromptTemplates.mockRejectedValueOnce(
      new Error("metadata down"),
    );
    const onStateChange = vi.fn();
    render(
      <PromptOverridesPanel
        scope="project"
        overrides={create(PromptTemplateOverridesSchema)}
        editable
        onStateChange={onStateChange}
      />,
    );

    expect(await screen.findByText("metadata down")).toBeInTheDocument();
    expect(onStateChange).toHaveBeenCalledWith(
      { design: "", implementation: "", batch: "", batchDesign: "" },
      false,
      false,
    );
  });
});
