import { create } from "@bufbuild/protobuf";
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  FeatureStatus,
  UpdateStatusSchema,
  type UpdateStatus,
} from "../src/gen/prx/v1/prx_pb";
import { setDisplayLanguage } from "../src/i18n";
import { AppShell } from "../src/shell";
import { makeFeature, makeProject, makeSnapshot } from "./factories";

const shellMocks = vi.hoisted(() => ({
  navigate: vi.fn().mockResolvedValue(undefined),
  mutation: {
    mutateAsync: vi.fn(),
    isPending: false,
    error: null as Error | null,
  },
  autoSync: vi.fn((enabled: boolean) => ({
    enabled,
    status: { data: undefined, isError: false },
    checking: false,
    error: null,
  })),
  revisionStream: vi.fn(() => ({ connected: true, stale: false })),
  updateStatus: vi.fn<() => { data: UpdateStatus | undefined }>(() => ({
    data: undefined,
  })),
}));

const snapshot = makeSnapshot({
  projects: [makeProject({ id: "P-1", title: "Delivery platform" })],
  features: [
    makeFeature({
      id: "active",
      title: "Active feature",
      projectId: "P-1",
      readyCount: 1,
    }),
    makeFeature({
      id: "conflict",
      title: "Conflict feature",
      projectId: "P-1",
      conflictCount: 1,
      readyCount: 0,
    }),
    makeFeature({
      id: "archived",
      title: "Archived feature",
      projectId: "P-1",
      archived: true,
    }),
    makeFeature({
      id: "completed",
      title: "Completed feature",
      projectId: "P-1",
      displayStatus: FeatureStatus.COMPLETED,
      taskCount: 2,
      finishedCount: 2,
      readyCount: 0,
    }),
  ],
});

vi.mock("@tanstack/react-router", () => ({
  Link: ({
    children,
    className,
  }: {
    children: ReactNode;
    className?: string;
  }) => <span className={className}>{children}</span>,
  useNavigate: () => shellMocks.navigate,
}));
vi.mock("../src/api", () => ({
  mutations: { createFeature: vi.fn() },
  configMutations: {
    addHost: vi.fn(),
    updateHost: vi.fn(),
    deleteHost: vi.fn(),
    addAuth: vi.fn(),
    updateAuth: vi.fn(),
    deleteAuth: vi.fn(),
    reorderAuth: vi.fn(),
  },
}));
vi.mock("../src/hooks", () => ({
  useDisplayLanguage: () => undefined,
  // 言語はサーバーの設定なので、保存すると実効言語が返ってくる。
  useLanguageMutation: () => ({
    mutateAsync: (language: string) =>
      Promise.resolve({ config: { effectiveLanguage: language } }),
    isError: false,
  }),
  useSnapshot: () => ({ data: snapshot, isError: false }),
  useAutoSync: (enabled: boolean) => shellMocks.autoSync(enabled),
  useRevisionStream: () => shellMocks.revisionStream(),
  useDomainMutation: () => shellMocks.mutation,
  useConfig: () => ({ data: { hosts: [], authMethods: [] }, isPending: false }),
  useConfigMutation: () => shellMocks.mutation,
  useUpdateStatus: () => shellMocks.updateStatus(),
}));
// モーダルの中身は update-dialog.test.tsx が受け持つ。ここでは案内から開くことだけを見る。
vi.mock("../src/views/UpdateDialog", () => ({
  UpdateDialog: ({
    status,
    onClose,
  }: {
    status: { latestVersion: string };
    onClose: () => void;
  }) => (
    <div role="dialog" aria-label={`PRX ${status.latestVersion} is available`}>
      <button type="button" onClick={onClose}>
        Close update
      </button>
    </div>
  ),
}));

describe("AppShell", () => {
  afterEach(cleanup);
  // isDemoMode は document を直接見るので、失敗した検証が meta を残すと
  // 以降のケースがすべてデモモードで動いてしまう。
  afterEach(() => {
    document.querySelector('meta[name="prx-demo"]')?.remove();
    document.querySelector('meta[name="prx-demo-session"]')?.remove();
  });
  beforeEach(async () => {
    localStorage.clear();
    await setDisplayLanguage("en");
    shellMocks.navigate.mockClear();
    shellMocks.autoSync.mockClear();
    shellMocks.revisionStream.mockClear();
    shellMocks.revisionStream.mockReturnValue({
      connected: true,
      stale: false,
    });
    shellMocks.updateStatus.mockClear();
    shellMocks.updateStatus.mockReturnValue({ data: undefined });
    shellMocks.mutation.mutateAsync.mockReset();
    shellMocks.mutation.mutateAsync.mockResolvedValue({
      feature: makeFeature({ id: "created" }),
    });
    shellMocks.mutation.isPending = false;
    shellMocks.mutation.error = null;
  });

  // 単発の切断では出さない。表示は 30 秒以上つながっていないときだけで、
  // それまでは従来どおりフォーカス復帰の再取得に任せる。
  it("shows the stopped automatic updates notice only while the stream stays down", () => {
    shellMocks.revisionStream.mockReturnValue({
      connected: false,
      stale: false,
    });
    const { rerender } = render(
      <AppShell>
        <p>Workspace</p>
      </AppShell>,
    );
    expect(document.querySelector(".rail-foot")).not.toBeInTheDocument();

    shellMocks.revisionStream.mockReturnValue({
      connected: false,
      stale: true,
    });
    rerender(
      <AppShell>
        <p>Workspace</p>
      </AppShell>,
    );
    expect(screen.getByText("Automatic updates stopped")).toBeInTheDocument();
  });

  // 案内は should_notify のときだけ出し、押すとリリース本文のモーダルが開く。
  it("shows the update notice only when the server asks for it", () => {
    const { rerender } = render(
      <AppShell>
        <p>Workspace</p>
      </AppShell>,
    );
    expect(
      screen.queryByRole("button", { name: "Update to v0.5.0" }),
    ).not.toBeInTheDocument();

    shellMocks.updateStatus.mockReturnValue({
      data: create(UpdateStatusSchema, {
        enabled: true,
        currentVersion: "0.3.0",
        updateAvailable: true,
        shouldNotify: true,
        latestVersion: "v0.5.0",
        releases: [{ version: "v0.5.0" }],
      }),
    });
    rerender(
      <AppShell>
        <p>Workspace</p>
      </AppShell>,
    );
    const notice = screen.getByRole("button", { name: "Update to v0.5.0" });
    fireEvent.click(notice);
    expect(
      screen.getByRole("dialog", { name: "PRX v0.5.0 is available" }),
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Close update" }));
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("shows the project tree and changes display settings from Settings", async () => {
    render(
      <AppShell>
        <p>Workspace</p>
      </AppShell>,
    );

    // ツリーには進行中のプロジェクトと、その下に進行中の feature が並ぶ。
    // それ以外は各ページのタブが受け持つ。
    expect(screen.getByText("Delivery platform")).toBeInTheDocument();
    expect(screen.getByText("Active feature")).toBeInTheDocument();
    expect(screen.getByText("Conflict feature")).toBeInTheDocument();
    expect(screen.queryByText("Archived feature")).not.toBeInTheDocument();
    expect(screen.queryByText("Completed feature")).not.toBeInTheDocument();
    expect(screen.getByText("Overview")).toBeInTheDocument();
    expect(screen.getByText(/Projects/)).toHaveTextContent("Projects 1");
    expect(screen.getByText("Workspace")).toBeInTheDocument();
    expect(screen.queryByText("Local database online")).not.toBeInTheDocument();
    expect(document.querySelector(".rail-foot")).not.toBeInTheDocument();
    expect(screen.queryByText("Dependency control")).not.toBeInTheDocument();
    expect(screen.queryByText("GitHub sync")).not.toBeInTheDocument();
    expect(shellMocks.autoSync).toHaveBeenCalledWith(true);
    expect(screen.queryByRole("combobox")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Settings" }));
    expect(screen.getByRole("tab", { name: "Server" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    expect(
      screen.queryByRole("combobox", { name: "Display language" }),
    ).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("tab", { name: "Display" }));
    fireEvent.change(
      screen.getByRole("combobox", { name: "Display language" }),
      {
        target: { value: "ja" },
      },
    );
    // 表示の設定も他のタブと同じく、フッタの保存を押すまで適用されない。
    expect(document.documentElement.lang).not.toBe("ja");
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => {
      expect(document.documentElement.lang).toBe("ja");
    });
    expect(screen.getByRole("dialog", { name: "設定" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "表示" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    fireEvent.change(screen.getByRole("combobox", { name: "表示テーマ" }), {
      target: { value: "dark" },
    });
    expect(document.documentElement.dataset["theme"]).toBeUndefined();
    fireEvent.click(screen.getByRole("button", { name: "保存" }));
    await waitFor(() => {
      expect(document.documentElement.dataset["theme"]).toBe("dark");
    });
    expect(
      JSON.parse(localStorage.getItem("prx.webui.settings") ?? "{}"),
    ).toEqual({ language: "ja", theme: "dark" });
  });

  it("keeps the demo reset warning visible and identifies temporary storage", () => {
    const meta = document.createElement("meta");
    meta.name = "prx-demo";
    meta.content = "true";
    document.head.append(meta);

    render(
      <AppShell>
        <p>Workspace</p>
      </AppShell>,
    );

    expect(screen.getByRole("status")).toHaveTextContent(
      "DEMO — Changes reset on restart / 変更は再起動時にリセットされます",
    );
    expect(
      screen.queryByText("Temporary demo database"),
    ).not.toBeInTheDocument();
    expect(document.querySelector(".rail-foot")).not.toBeInTheDocument();
  });

  it("keeps the dismissed demo warning hidden until another session is served", () => {
    const demoMeta = document.createElement("meta");
    demoMeta.name = "prx-demo";
    demoMeta.content = "true";
    document.head.append(demoMeta);
    const sessionMeta = document.createElement("meta");
    sessionMeta.name = "prx-demo-session";
    sessionMeta.content = "session-1";
    document.head.append(sessionMeta);

    render(
      <AppShell>
        <p>Workspace</p>
      </AppShell>,
    );

    fireEvent.click(
      screen.getByRole("button", {
        name: "Hide the demo notice until the demo server restarts",
      }),
    );
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
    expect(document.querySelector("[data-demo]")).toBeNull();

    // 読み込み直しやホットリロードに相当する再マウントでは戻らない。
    cleanup();
    render(
      <AppShell>
        <p>Workspace</p>
      </AppShell>,
    );
    expect(screen.queryByRole("status")).not.toBeInTheDocument();

    // demo のサーバを起動し直すと ID が変わり、警告が戻る。
    sessionMeta.content = "session-2";
    cleanup();
    render(
      <AppShell>
        <p>Workspace</p>
      </AppShell>,
    );
    expect(screen.getByRole("status")).toBeInTheDocument();
  });

  it("opens and closes Settings from the rail", () => {
    render(
      <AppShell>
        <p>Workspace</p>
      </AppShell>,
    );
    fireEvent.click(screen.getByRole("button", { name: "Settings" }));
    expect(
      screen.getByRole("dialog", { name: "Settings" }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Close" }));
    expect(
      screen.queryByRole("dialog", { name: "Settings" }),
    ).not.toBeInTheDocument();
  });
});
