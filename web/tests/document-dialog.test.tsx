import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import type { PropsWithChildren } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DocumentKind } from "../src/gen/nnx/v1/nnx_pb";
import { DocumentDialog } from "../src/views/DocumentDialog";
import { makeDocument } from "./factories";

const dialogMocks = vi.hoisted(() => ({
  addDocument: vi.fn(),
  updateDocument: vi.fn(),
  selectLocalFile: vi.fn(),
}));

vi.mock("../src/api", () => ({
  mutations: {
    addDocument: dialogMocks.addDocument,
    updateDocument: dialogMocks.updateDocument,
  },
  selectLocalFile: dialogMocks.selectLocalFile,
}));

function Wrapper({ children }: PropsWithChildren) {
  const queryClient = new QueryClient({
    defaultOptions: { mutations: { retry: false }, queries: { retry: false } },
  });
  return (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

function renderDialog(props: { featureId?: string; taskId?: string } = {}) {
  const trigger = document.createElement("button");
  document.body.append(trigger);
  const onClose = vi.fn();
  const parent = props.taskId
    ? { taskId: props.taskId }
    : { featureId: props.featureId ?? "feature-1" };
  render(
    <DocumentDialog
      mode="add"
      {...parent}
      trigger={trigger}
      onClose={onClose}
    />,
    { wrapper: Wrapper },
  );
  return { onClose, trigger };
}

function renderEditDialog(
  overrides: Parameters<typeof makeDocument>[0] = {},
  content = "# Decision",
) {
  const trigger = document.createElement("button");
  document.body.append(trigger);
  const onClose = vi.fn();
  const target = makeDocument({
    id: "document-1",
    featureId: "feature-1",
    taskId: "",
    kind: DocumentKind.MARKDOWN,
    title: "Decision log",
    locator: "",
    ...overrides,
  });
  render(
    <DocumentDialog
      mode="edit"
      document={target}
      content={content}
      trigger={trigger}
      onClose={onClose}
    />,
    { wrapper: Wrapper },
  );
  return { onClose, trigger, document: target };
}

describe("DocumentDialog", () => {
  beforeEach(() => {
    dialogMocks.addDocument.mockReset().mockResolvedValue({});
    dialogMocks.updateDocument.mockReset().mockResolvedValue({});
    dialogMocks.selectLocalFile
      .mockReset()
      .mockResolvedValue({ path: "/tmp/plan.md", canceled: false });
  });

  afterEach(() => {
    cleanup();
  });

  it("keeps tab inputs and adds task Markdown as an implementation plan", async () => {
    const { onClose } = renderDialog({ taskId: "task-1" });
    const urlTab = screen.getByRole("tab", { name: "URL" });
    await waitFor(() => {
      expect(urlTab).toHaveFocus();
    });
    fireEvent.change(screen.getByLabelText("Document URL"), {
      target: { value: "https://example.com/spec" },
    });

    fireEvent.keyDown(urlTab, { key: "ArrowRight" });
    const localTab = screen.getByRole("tab", { name: "Local file" });
    expect(localTab).toHaveFocus();
    fireEvent.click(screen.getByRole("button", { name: "Choose file…" }));
    await waitFor(() => {
      expect(screen.getByLabelText("File path")).toHaveValue("/tmp/plan.md");
    });

    fireEvent.keyDown(localTab, { key: "End" });
    const markdownTab = screen.getByRole("tab", { name: "Markdown" });
    expect(markdownTab).toHaveFocus();
    fireEvent.change(screen.getByLabelText("Markdown content"), {
      target: { value: "# Delivery\n\nShip safely." },
    });
    fireEvent.click(
      screen.getByLabelText("Use as this task's implementation plan"),
    );
    fireEvent.change(screen.getByLabelText("Reference title (optional)"), {
      target: { value: "Delivery plan" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Add reference" }));

    await waitFor(() => {
      expect(dialogMocks.addDocument).toHaveBeenCalledWith(
        {
          taskId: "task-1",
          kind: DocumentKind.MARKDOWN,
          title: "Delivery plan",
          value: "# Delivery\n\nShip safely.",
          isImplementationPlan: true,
        },
        expect.anything(),
      );
    });
    expect(onClose).toHaveBeenCalledOnce();
    fireEvent.click(urlTab);
    expect(screen.getByLabelText("Document URL")).toHaveValue(
      "https://example.com/spec",
    );
  });

  it("treats picker cancellation as recoverable and displays picker errors", async () => {
    dialogMocks.selectLocalFile
      .mockResolvedValueOnce({ path: "", canceled: true })
      .mockRejectedValueOnce(new Error("picker unavailable"));
    renderDialog();
    fireEvent.click(screen.getByRole("tab", { name: "Local file" }));
    fireEvent.click(screen.getByRole("button", { name: "Choose file…" }));
    expect(await screen.findByText(/No file was selected/)).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("File path"), {
      target: { value: "/tmp/manual.md" },
    });
    expect(screen.queryByText(/No file was selected/)).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "Choose file…" }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "picker unavailable",
    );
  });

  it("supports Home, ArrowLeft, Escape, and the feature-only form", async () => {
    const { onClose } = renderDialog({ featureId: "feature-2" });
    const urlTab = screen.getByRole("tab", { name: "URL" });
    await waitFor(() => {
      expect(urlTab).toHaveFocus();
    });
    fireEvent.keyDown(urlTab, { key: "ArrowLeft" });
    const markdownTab = screen.getByRole("tab", { name: "Markdown" });
    expect(markdownTab).toHaveFocus();
    fireEvent.keyDown(markdownTab, { key: "Home" });
    expect(urlTab).toHaveFocus();
    expect(
      screen.queryByLabelText("Use as this task's implementation plan"),
    ).toBeNull();
    fireEvent.keyDown(screen.getByRole("dialog"), { key: "Escape" });
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("edits a feature reference and can switch its source kind", async () => {
    const { onClose } = renderEditDialog();
    expect(
      screen.getByRole("dialog", { name: "Edit feature reference" }),
    ).toBeInTheDocument();
    expect(screen.getByLabelText("Reference title (optional)")).toHaveValue(
      "Decision log",
    );
    expect(screen.getByLabelText("Markdown content")).toHaveValue("# Decision");
    // feature の資料には実装プランの指定を出さない。
    expect(
      screen.queryByLabelText("Use as this task's implementation plan"),
    ).toBeNull();
    const save = screen.getByRole("button", { name: "Save" });
    expect(save).toBeDisabled();

    fireEvent.click(screen.getByRole("tab", { name: "URL" }));
    fireEvent.change(screen.getByLabelText("Document URL"), {
      target: { value: "https://example.com/decision" },
    });
    fireEvent.change(screen.getByLabelText("Reference title (optional)"), {
      target: { value: "Decision record" },
    });
    expect(save).toBeEnabled();
    fireEvent.click(save);

    await waitFor(() => {
      expect(dialogMocks.updateDocument).toHaveBeenCalledWith(
        {
          id: "document-1",
          title: "Decision record",
          source: { case: "url", value: "https://example.com/decision" },
        },
        expect.anything(),
      );
    });
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("keeps the implementation plan flag for task references", async () => {
    renderEditDialog({
      featureId: "",
      taskId: "task-1",
      kind: DocumentKind.LOCAL_FILE,
      title: "Delivery plan",
      locator: "docs/delivery.md",
      isImplementationPlan: true,
    });
    expect(
      screen.getByRole("dialog", { name: "Edit task reference" }),
    ).toBeInTheDocument();
    expect(screen.getByLabelText("File path")).toHaveValue("docs/delivery.md");
    const plan = screen.getByLabelText(
      "Use as this task's implementation plan",
    );
    expect(plan).toBeChecked();
    fireEvent.click(plan);
    fireEvent.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => {
      expect(dialogMocks.updateDocument).toHaveBeenCalledWith(
        {
          id: "document-1",
          title: "Delivery plan",
          source: { case: "localFile", value: "docs/delivery.md" },
          isImplementationPlan: false,
        },
        expect.anything(),
      );
    });
  });

  it("confirms before discarding unsaved edits", () => {
    const { onClose } = renderEditDialog();
    fireEvent.keyDown(screen.getByRole("dialog"), { key: "Escape" });
    expect(onClose).toHaveBeenCalledOnce();

    onClose.mockClear();
    fireEvent.change(screen.getByLabelText("Markdown content"), {
      target: { value: "# Revised" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(onClose).not.toHaveBeenCalled();
    expect(
      screen.getByRole("dialog", { name: "Discard unsaved changes?" }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(screen.getByLabelText("Markdown content")).toHaveValue("# Revised");

    fireEvent.click(screen.getByRole("button", { name: "Close" }));
    fireEvent.click(screen.getByRole("button", { name: "Discard" }));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("shows the project heading and keeps failed edits for retry", async () => {
    dialogMocks.updateDocument.mockRejectedValueOnce(
      new Error("Update failed"),
    );
    renderEditDialog({ featureId: "", taskId: "", projectId: "project-1" });
    expect(
      screen.getByRole("dialog", { name: "Edit project reference" }),
    ).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Markdown content"), {
      target: { value: "# Revised" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Update failed");
    expect(screen.getByLabelText("Markdown content")).toHaveValue("# Revised");
  });
});
