import { expect, test } from "@playwright/test";
import { e2eBaseURL, guardBrowserErrors } from "./helpers";

test.use({ baseURL: e2eBaseURL });

guardBrowserErrors();

// 検索は共有のデモグラフを読むだけなので、この spec は他と同じサーバーに対して
// 並列に実行する。
test("filters the graph from the search box and restores it on Escape", async ({
  page,
}) => {
  await page.goto("/");
  await page
    .getByRole("link", { name: /Delivery control showcase/ })
    .first()
    .click();
  const nodes = page.locator(".task-node");
  await expect(nodes).toHaveCount(16, { timeout: 25_000 });

  // ショートカットはツールバーのボタンと同じ検索窓を開く。
  await page.keyboard.press("ControlOrMeta+f");
  const field = page.getByRole("textbox", {
    name: "Search tasks on this graph",
  });
  await expect(field).toBeFocused();

  await field.fill("storage");
  await expect(nodes).toHaveCount(1, { timeout: 25_000 });
  await expect(nodes.first()).toContainText("Verify storage boundary");
  await expect(page.getByText("1 of 16 shown")).toBeVisible();

  // 一致が無くなると、完了非表示ではなく検索の空状態を出す。
  await field.fill("no such task");
  await expect(nodes).toHaveCount(0, { timeout: 25_000 });
  await expect(
    page.getByRole("heading", { name: "No task matches the search" }),
  ).toBeVisible();

  await page.keyboard.press("Escape");
  await expect(field).toBeHidden();
  await expect(nodes).toHaveCount(16, { timeout: 25_000 });
  await expect(
    page.getByRole("button", { name: "Search tasks" }),
  ).toBeFocused();
});
