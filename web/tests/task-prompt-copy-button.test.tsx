import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { TaskPromptKind } from "../src/gen/prx/v1/prx_pb";
import { setDisplayLanguage } from "../src/i18n";
import { TaskPromptCopyButton } from "../src/views/TaskPromptCopyButton";

const promptMocks = vi.hoisted(() => ({ getTaskPrompt: vi.fn() }));

vi.mock("../src/api", () => ({ getTaskPrompt: promptMocks.getTaskPrompt }));

function renderButton(hasImplementationPlan: boolean) {
  return render(
    <TaskPromptCopyButton
      taskId="T-1"
      hasImplementationPlan={hasImplementationPlan}
    />,
  );
}

function openDialog() {
  fireEvent.click(screen.getByRole("button", { name: "Copy task prompt" }));
}

function stubClipboard(writeText: ReturnType<typeof vi.fn>) {
  Object.defineProperty(navigator, "clipboard", {
    configurable: true,
    value: { writeText },
  });
}

function preview(): HTMLElement {
  return screen.getByLabelText("Prompt preview");
}

describe("TaskPromptCopyButton", () => {
  beforeEach(async () => {
    vi.clearAllMocks();
    await setDisplayLanguage("en");
  });

  afterEach(() => {
    cleanup();
  });

  // 既定のタブはサーバーの導出と同じ。これまでどおりのプロンプトはタブを
  // 触らずにコピーできる。
  it("previews the design prompt of a task that has no plan", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    stubClipboard(writeText);
    promptMocks.getTaskPrompt.mockResolvedValue({ prompt: "Design T-1" });
    renderButton(false);
    openDialog();

    await waitFor(() => {
      expect(preview()).toHaveValue("Design T-1");
    });
    expect(promptMocks.getTaskPrompt).toHaveBeenCalledWith(
      "T-1",
      TaskPromptKind.DESIGN,
    );
    expect(
      screen.getByRole("tab", { name: "Design", selected: true }),
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Copy prompt" }));
    await waitFor(() => {
      expect(screen.getByText("Design prompt copied.")).toBeInTheDocument();
    });
    expect(writeText).toHaveBeenCalledWith("Design T-1");
  });

  it("opens on the implementation prompt once the task has a plan", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    stubClipboard(writeText);
    promptMocks.getTaskPrompt.mockResolvedValue({ prompt: "Build T-1" });
    renderButton(true);
    openDialog();

    await waitFor(() => {
      expect(preview()).toHaveValue("Build T-1");
    });
    expect(promptMocks.getTaskPrompt).toHaveBeenCalledWith(
      "T-1",
      TaskPromptKind.IMPLEMENTATION,
    );

    fireEvent.click(screen.getByRole("button", { name: "Copy prompt" }));
    await waitFor(() => {
      expect(
        screen.getByText("Implementation prompt copied."),
      ).toBeInTheDocument();
    });
    expect(writeText).toHaveBeenCalledWith("Build T-1");
  });

  // タスクの状態に合わない種類も選べるが、選んでいる間は何がずれているかを
  // 伝える。
  it("warns while the selected kind does not match the task", async () => {
    stubClipboard(vi.fn().mockResolvedValue(undefined));
    promptMocks.getTaskPrompt.mockResolvedValue({ prompt: "Design T-1" });
    renderButton(false);
    openDialog();

    await waitFor(() => {
      expect(preview()).toHaveValue("Design T-1");
    });
    expect(
      screen.queryByText(/no implementation plan yet/),
    ).not.toBeInTheDocument();

    promptMocks.getTaskPrompt.mockResolvedValue({ prompt: "Build T-1" });
    fireEvent.click(screen.getByRole("tab", { name: "Implementation" }));

    await waitFor(() => {
      expect(preview()).toHaveValue("Build T-1");
    });
    expect(
      screen.getByText(/This task has no implementation plan yet/),
    ).toBeInTheDocument();
    expect(promptMocks.getTaskPrompt).toHaveBeenLastCalledWith(
      "T-1",
      TaskPromptKind.IMPLEMENTATION,
    );
  });

  it("warns when the design prompt would replace an existing plan", async () => {
    stubClipboard(vi.fn().mockResolvedValue(undefined));
    promptMocks.getTaskPrompt.mockResolvedValue({ prompt: "Build T-1" });
    renderButton(true);
    openDialog();
    await waitFor(() => {
      expect(preview()).toHaveValue("Build T-1");
    });

    promptMocks.getTaskPrompt.mockResolvedValue({ prompt: "Design T-1" });
    fireEvent.click(screen.getByRole("tab", { name: "Design" }));
    await waitFor(() => {
      expect(
        screen.getByText(/registering a new one from the design prompt/),
      ).toBeInTheDocument();
    });
  });

  it("reports a server failure and refuses to copy", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    stubClipboard(writeText);
    promptMocks.getTaskPrompt.mockRejectedValue(
      new Error('task "T-1" was not found'),
    );
    renderButton(false);
    openDialog();

    await waitFor(() => {
      expect(screen.getByText('task "T-1" was not found')).toBeInTheDocument();
    });
    expect(screen.getByRole("button", { name: "Copy prompt" })).toBeDisabled();
    expect(writeText).not.toHaveBeenCalled();
  });

  // クリップボードの失敗はブラウザ内部のメッセージを伴うので、読み手には
  // 表示言語で何が起きたかを伝える。
  it("reports a clipboard failure in the display language", async () => {
    stubClipboard(vi.fn().mockRejectedValue(new Error("clipboard blocked")));
    promptMocks.getTaskPrompt.mockResolvedValue({ prompt: "Design T-1" });
    renderButton(false);
    openDialog();
    await waitFor(() => {
      expect(preview()).toHaveValue("Design T-1");
    });

    fireEvent.click(screen.getByRole("button", { name: "Copy prompt" }));
    await waitFor(() => {
      expect(
        screen.getByText("The prompt could not be copied."),
      ).toBeInTheDocument();
    });
    expect(screen.queryByText("clipboard blocked")).not.toBeInTheDocument();
  });

  // secure context 外では clipboard API 自体が存在せず、そのままだと生の
  // TypeError が表に出てしまう。
  it("reports a missing clipboard API", async () => {
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: undefined,
    });
    promptMocks.getTaskPrompt.mockResolvedValue({ prompt: "Design T-1" });
    renderButton(false);
    openDialog();
    await waitFor(() => {
      expect(preview()).toHaveValue("Design T-1");
    });

    fireEvent.click(screen.getByRole("button", { name: "Copy prompt" }));
    await waitFor(() => {
      expect(
        screen.getByText("The prompt could not be copied."),
      ).toBeInTheDocument();
    });
  });

  // モーダルは Tab を内側に閉じ込め、閉じたら開く前のボタンへ焦点を戻す。
  // docs/design/webui.md を参照。
  it("traps Tab inside the dialog and restores the focus on close", async () => {
    vi.useFakeTimers();
    stubClipboard(vi.fn().mockResolvedValue(undefined));
    promptMocks.getTaskPrompt.mockResolvedValue({ prompt: "Design T-1" });
    renderButton(false);
    const trigger = screen.getByRole("button", { name: "Copy task prompt" });
    trigger.focus();
    openDialog();
    await vi.waitFor(() => {
      expect(preview()).toHaveValue("Design T-1");
    });

    const close = screen.getByRole("button", { name: "Close" });
    const copy = screen.getByRole("button", { name: "Copy prompt" });
    expect(close).toHaveFocus();
    fireEvent.keyDown(screen.getByRole("presentation"), {
      key: "Tab",
      shiftKey: true,
    });
    expect(copy).toHaveFocus();
    fireEvent.keyDown(screen.getByRole("presentation"), { key: "Tab" });
    expect(close).toHaveFocus();
    // Tab 以外はそのまま通す。
    fireEvent.keyDown(screen.getByRole("presentation"), { key: "a" });
    expect(close).toHaveFocus();

    fireEvent.click(close);
    act(() => {
      vi.runOnlyPendingTimers();
    });
    expect(trigger).toHaveFocus();
    vi.useRealTimers();
  });

  // 重なったモーダルの Escape は最前面だけを閉じる。ここには 1 つしかない
  // ので、その 1 つが閉じてボタンだけが残る。
  it("closes on Escape", async () => {
    stubClipboard(vi.fn().mockResolvedValue(undefined));
    promptMocks.getTaskPrompt.mockResolvedValue({ prompt: "Design T-1" });
    renderButton(false);
    openDialog();
    await waitFor(() => {
      expect(screen.getByRole("dialog")).toBeInTheDocument();
    });

    fireEvent.keyDown(window, { key: "Escape" });
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });
});
