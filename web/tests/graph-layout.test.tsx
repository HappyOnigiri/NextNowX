import { createMemoryHistory, RouterProvider } from "@tanstack/react-router";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { router } from "../src/router";
import { makeSnapshot } from "./factories";

// React Flow はコンテナを計測するが、jsdom はそれを実装していない。
class ResizeObserverStub {
  observe() {
    return undefined;
  }
  unobserve() {
    return undefined;
  }
  disconnect() {
    return undefined;
  }
}
globalThis.ResizeObserver = ResizeObserverStub;

const snapshot = makeSnapshot();

vi.mock("../src/hooks", () => ({
  useDisplayLanguage: () => undefined,
  // 言語はサーバーの設定なので、保存すると実効言語が返ってくる。
  useLanguageMutation: () => ({
    mutateAsync: (language: string) =>
      Promise.resolve({ config: { effectiveLanguage: language } }),
    isError: false,
  }),
  useSnapshot: () => ({
    data: snapshot,
    isPending: false,
    isError: false,
    error: null,
    refetch: vi.fn(),
  }),
  useAutoSync: () => ({
    status: { data: undefined, isError: false },
    checking: false,
    error: null,
  }),
  useDomainMutation: () => ({
    mutate: vi.fn(),
    mutateAsync: vi.fn(),
    isPending: false,
    error: null,
  }),
}));

vi.mock("elkjs/lib/elk-api.js", () => ({
  default: class {
    layout() {
      return Promise.reject(new Error("worker unavailable"));
    }
    terminateWorker() {
      return undefined;
    }
  },
}));

describe("FeatureWorkspace graph layout", () => {
  afterEach(cleanup);

  beforeEach(async () => {
    router.update({
      history: createMemoryHistory({ initialEntries: ["/features/feature-1"] }),
    });
    await router.load();
  });

  it("surfaces a layout failure instead of leaving the canvas blank", async () => {
    render(<RouterProvider router={router} />);
    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent("Graph layout failed");
    expect(alert).toHaveTextContent("worker unavailable");
    expect(
      screen.getByRole("button", { name: "Retry layout" }),
    ).toBeInTheDocument();
  });
});
