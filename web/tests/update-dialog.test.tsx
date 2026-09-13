import { create } from "@bufbuild/protobuf";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { UpdateStatusSchema } from "../src/gen/prx/v1/prx_pb";
import { setDisplayLanguage } from "../src/i18n";
import { UpdateDialog } from "../src/views/UpdateDialog";

const updateMocks = vi.hoisted(() => ({
  applyUpdate: vi.fn(),
  skipUpdateVersion: vi.fn(),
  invalidate: vi.fn().mockResolvedValue(undefined),
}));

vi.mock("../src/api", () => ({
  applyUpdate: updateMocks.applyUpdate,
  skipUpdateVersion: updateMocks.skipUpdateVersion,
}));
vi.mock("../src/hooks", () => ({
  useUpdateStatusInvalidation: () => updateMocks.invalidate,
}));

function status() {
  return create(UpdateStatusSchema, {
    enabled: true,
    currentVersion: "0.3.0",
    updateAvailable: true,
    shouldNotify: true,
    latestVersion: "v0.5.0",
    releases: [
      {
        version: "v0.5.0",
        publishedAt: "2026-09-02T10:00:00Z",
        body: "## Highlights\n\n- Faster [graph](https://example.test/graph)",
        url: "https://example.test/v0.5.0",
      },
      { version: "v0.4.0", publishedAt: "2026-09-01T10:00:00Z", body: "" },
    ],
  });
}

function renderDialog(onClose = vi.fn()) {
  const client = new QueryClient({
    defaultOptions: { mutations: { retry: false } },
  });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  render(<UpdateDialog status={status()} onClose={onClose} />, { wrapper });
  return onClose;
}

describe("UpdateDialog", () => {
  afterEach(cleanup);
  beforeEach(async () => {
    await setDisplayLanguage("en");
    updateMocks.applyUpdate.mockReset();
    updateMocks.skipUpdateVersion.mockReset();
    updateMocks.invalidate.mockClear();
  });

  // 本文は Markdown として描き、リリースは新しい順に並べる。
  it("renders every release note newest first", () => {
    renderDialog();
    expect(
      screen.getByRole("dialog", { name: "PRX v0.5.0 is available" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("heading", { name: "Highlights" }),
    ).toBeInTheDocument();
    expect(screen.getByText("This release has no notes.")).toBeInTheDocument();
    // 本文のリンクは新しいタブで開く。
    expect(screen.getByRole("link", { name: "graph" })).toHaveAttribute(
      "target",
      "_blank",
    );
    const versions = screen
      .getAllByRole("article")
      .map((article) => article.querySelector("h3")?.textContent);
    expect(versions).toEqual(["v0.5.02026-09-02", "v0.4.02026-09-01"]);
  });

  it("installs the newest release and reports the restart", async () => {
    updateMocks.applyUpdate.mockResolvedValue({
      version: "v0.5.0",
      installedPath: "/home/example/.local/bin/prx",
      restartRequired: true,
    });
    renderDialog();
    fireEvent.click(screen.getByRole("button", { name: "Update now" }));
    await waitFor(() => {
      expect(updateMocks.applyUpdate).toHaveBeenCalledWith("v0.5.0");
    });
    expect(
      await screen.findByText(
        "Installed v0.5.0. Restart the running prx serve to use it.",
      ),
    ).toBeInTheDocument();
    expect(updateMocks.invalidate).toHaveBeenCalled();
  });

  // 常駐は置き換えを自分で検知するので、再起動は案内しない。
  it("says nothing about restarting when the daemon handles it", async () => {
    updateMocks.applyUpdate.mockResolvedValue({
      version: "v0.5.0",
      installedPath: "/home/example/.local/bin/prx",
      restartRequired: false,
    });
    renderDialog();
    fireEvent.click(screen.getByRole("button", { name: "Update now" }));
    expect(
      await screen.findByText(
        "Installed v0.5.0. The background server restarts itself.",
      ),
    ).toBeInTheDocument();
  });

  it("skips the latest version and closes", async () => {
    updateMocks.skipUpdateVersion.mockResolvedValue({});
    const onClose = renderDialog();
    fireEvent.click(screen.getByRole("button", { name: "Skip this version" }));
    await waitFor(() => {
      expect(updateMocks.skipUpdateVersion).toHaveBeenCalledWith("v0.5.0");
    });
    expect(onClose).toHaveBeenCalled();
  });

  it("closes on cancel and on Escape", () => {
    const onClose = renderDialog();
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(onClose).toHaveBeenCalledTimes(1);
    fireEvent.keyDown(window, { key: "Escape" });
    expect(onClose).toHaveBeenCalledTimes(2);
  });

  it("reports a failed install without closing", async () => {
    updateMocks.applyUpdate.mockRejectedValue(
      new Error("checksum verification failed"),
    );
    const onClose = renderDialog();
    fireEvent.click(screen.getByRole("button", { name: "Update now" }));
    expect(
      await screen.findByText(/checksum verification failed/),
    ).toBeInTheDocument();
    expect(onClose).not.toHaveBeenCalled();
  });

  // 書き込みの完了を待っている間は閉じさせず、処理中であることを支援技術へ伝える。
  it("keeps the dialog open and busy while installing", async () => {
    let finish = (value: unknown) => value;
    updateMocks.applyUpdate.mockReturnValue(
      new Promise((resolve) => {
        finish = resolve;
      }),
    );
    const onClose = renderDialog();
    fireEvent.click(screen.getByRole("button", { name: "Update now" }));
    await waitFor(() => {
      expect(screen.getByRole("dialog")).toHaveAttribute("aria-busy", "true");
    });
    fireEvent.keyDown(window, { key: "Escape" });
    expect(onClose).not.toHaveBeenCalled();
    finish({ version: "v0.5.0", installedPath: "", restartRequired: false });
    await waitFor(() => {
      expect(screen.getByRole("dialog")).not.toHaveAttribute("aria-busy");
    });
  });

  // Tab はモーダルの中だけを回る。末尾の「今すぐ更新」から先頭のリリースリンクへ戻る。
  it("keeps focus inside the dialog", () => {
    renderDialog();
    const install = screen.getByRole("button", { name: "Update now" });
    const link = screen.getByRole("link", { name: "v0.5.0" });
    expect(install).toHaveFocus();
    fireEvent.keyDown(install, { key: "Tab" });
    expect(link).toHaveFocus();
    fireEvent.keyDown(link, { key: "Tab", shiftKey: true });
    expect(install).toHaveFocus();
  });
});
