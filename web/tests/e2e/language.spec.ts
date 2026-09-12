import { expect, test } from "@playwright/test";
import {
  guardBrowserErrors,
  languageBaseURL,
  openDisplaySettings,
  saveSettings,
} from "./helpers";

// 表示言語はサーバーの共有設定なので、切り替えると同じサーバーを見ている画面が
// すべて変わる。言語に触れるテストはこの spec に集め、専用のサーバーへ向ける。
test.use({ baseURL: languageBaseURL });

guardBrowserErrors();

test.describe.configure({ mode: "serial" });

test("switches the display language and restores it from the shared configuration", async ({
  page,
}) => {
  await page.goto("/");
  await openDisplaySettings(page);
  await page.getByLabel("Display language").selectOption("ja");
  await expect(page.locator("html")).toHaveAttribute("lang", "en");
  await saveSettings(page);
  await expect(
    page.getByRole("heading", { name: /いま動かせるタスク/ }),
  ).toBeVisible();
  await expect(page.locator("html")).toHaveAttribute("lang", "ja");
  // 言語はサーバーに残るので、読み込み直しても Local Storage ではなく設定から
  // 復元される。
  await page.reload();
  await openDisplaySettings(page, "ja");
  await expect(page.getByLabel("表示言語")).toHaveValue("ja");
  await expect(
    page.getByRole("heading", { name: /いま動かせるタスク/ }),
  ).toBeVisible();

  // 同じサーバーを使う後続のテストのために、設定を自動判定へ戻す。
  await page.getByLabel("表示言語").selectOption("auto");
  await saveSettings(page, "ja");
  await expect(page.locator("html")).toHaveAttribute("lang", "en");
});

test("keeps controls usable at a narrow viewport", async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 720 });
  await page.goto("/");
  await expect(
    page.getByRole("heading", { name: /What can move/ }),
  ).toBeVisible();
  await expect(page.locator(".page-head .dashboard-sync")).toBeVisible();
  await expect(page.locator(".rail .dashboard-sync")).toHaveCount(0);
  const dashboardSyncButton = page.getByRole("button", {
    name: "Sync GitHub",
  });
  await expect(dashboardSyncButton).toBeVisible();
  const dashboardSyncBounds = await dashboardSyncButton.boundingBox();
  expect(dashboardSyncBounds).not.toBeNull();
  if (dashboardSyncBounds)
    expect(
      dashboardSyncBounds.x + dashboardSyncBounds.width,
    ).toBeLessThanOrEqual(320);
  // この幅では 1 行の rail にツリーを置けないが、Projects リンクは残り、その
  // ページがツリーの役割を担う。
  await expect(page.getByRole("link", { name: /Projects/ })).toBeVisible();
  await expect(page.locator(".rail .nav-tree")).toBeHidden();
  await expect(page.locator("body")).toHaveJSProperty("scrollWidth", 320);
  await openDisplaySettings(page);
  await page.getByLabel("Display language").selectOption("ja");
  await saveSettings(page);
  const settingsDialog = page.getByRole("dialog", { name: "設定" });
  await expect(settingsDialog).toBeVisible();
  await expect(page.getByRole("tab", { name: "表示" })).toHaveAttribute(
    "aria-selected",
    "true",
  );
  for (const control of await settingsDialog.getByRole("combobox").all()) {
    const bounds = await control.boundingBox();
    expect(bounds).not.toBeNull();
    if (bounds) {
      expect(bounds.x).toBeGreaterThanOrEqual(0);
      expect(bounds.x + bounds.width).toBeLessThanOrEqual(320);
    }
  }
  await expect(page.locator("body")).toHaveJSProperty("scrollWidth", 320);
  await page.getByLabel("表示言語").selectOption("en");
  await saveSettings(page, "ja");
  await page
    .getByRole("dialog", { name: "Settings" })
    .getByRole("button", { name: "Close" })
    .click();
  // この幅ではツリーが隠れるので、feature へはサイドバーではなく Projects
  // ページから辿る。
  await page.getByRole("link", { name: /Projects/ }).click();
  await page
    .getByRole("region", { name: "Project list" })
    .getByText("Delivery platform")
    .click();
  await page
    .getByRole("link", { name: /Delivery control showcase/ })
    .first()
    .click();
  const addTaskButton = page.getByRole("button", { name: "Add task" });
  await expect(addTaskButton).toBeVisible();
  await expect(addTaskButton.locator("svg")).toHaveAttribute(
    "aria-hidden",
    "true",
  );
  await expect(page.getByRole("button", { name: "Sync GitHub" })).toBeHidden();
  await expect(page.getByRole("button", { name: "Edit feature" })).toBeHidden();
  const referencesButton = page.getByRole("button", { name: "References" });
  await expect(referencesButton).toBeVisible();
  await referencesButton.click();
  const referencesPanel = page.getByRole("region", { name: "References" });
  await expect(referencesPanel).toBeVisible();
  await expect(
    referencesPanel.getByRole("button", { name: "Add reference", exact: true }),
  ).toBeVisible();
  await expect(
    page.locator(".workspace-actions > .icon-button-danger"),
  ).toHaveCount(0);
  const addTaskBounds = await addTaskButton.boundingBox();
  expect(addTaskBounds).not.toBeNull();
  if (addTaskBounds) {
    expect(addTaskBounds.x + addTaskBounds.width).toBeLessThanOrEqual(320);
  }
});

// デモの警告は言語を切り替えても二言語のまま残る。言語を書き換えるので、
// demo.spec ではなくこの spec に置く。
test("keeps the bilingual demo reset warning visible", async ({ page }) => {
  await page.goto("/");
  const banner = page.getByRole("status");
  await expect(banner).toContainText("DEMO");
  await expect(banner).toContainText("Changes reset on restart");
  await expect(banner).toContainText("変更は再起動時にリセットされます");

  await openDisplaySettings(page);
  await page.getByLabel("Display theme").selectOption("dark");
  await page.getByLabel("Display language").selectOption("ja");
  // 表示の設定はフッタの保存 1 つで適用する。
  await page
    .getByRole("dialog", { name: "Settings" })
    .getByRole("button", { name: "Save" })
    .click();
  await expect(banner).toBeVisible();
  await page.getByRole("button", { name: "閉じる" }).first().click();

  await page.setViewportSize({ width: 320, height: 720 });
  await page.evaluate(() => {
    document.body.style.zoom = "2";
  });
  // toContainText は textContent を見るため、スクリーンリーダーに読む内容が
  // 残っていなくても、隠れた広幅用の文言だけで条件を満たしてしまう。
  const compact = banner.locator(".demo-banner-compact");
  await expect(compact).toBeVisible();
  await expect(banner.locator(".demo-banner-full")).toBeHidden();
  await expect(compact).toHaveText("DEMO · Reset on restart再起動でリセット");
  // 閉じるボタンのアイコンは装飾なので svg は数えない。
  expect(
    await banner.evaluate(
      (element) =>
        Array.from(element.querySelectorAll("[aria-hidden='true']:not(svg)"))
          .length,
    ),
  ).toBe(0);

  // 同じサーバーを使う後続のテストのために、設定を自動判定へ戻す。拡大と
  // 幅を戻すだけでは body の zoom が残るので、読み込み直してから操作する。
  await page.setViewportSize({ width: 1280, height: 720 });
  await page.goto("/");
  await openDisplaySettings(page, "ja");
  await page.getByLabel("表示言語").selectOption("auto");
  await saveSettings(page, "ja");
  await expect(page.locator("html")).toHaveAttribute("lang", "en");
});
