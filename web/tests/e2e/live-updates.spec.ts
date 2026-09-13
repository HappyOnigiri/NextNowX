import { expect, test } from "@playwright/test";
import { createFeature, e2eBaseURL, guardBrowserErrors } from "./helpers";

test.use({ baseURL: e2eBaseURL });

guardBrowserErrors();

// 別のページでの変更が、読み込み直さずに届くことを見る。CLI からの変更も同じ
// 経路（サーバーがデータベースを観測してリビジョンを配る）で反映される。
test("shows a feature another page created without a reload", async ({
  browser,
}) => {
  const context = await browser.newContext();
  const watcher = await context.newPage();
  const editor = await context.newPage();
  const title = `Live update ${Date.now()}`;
  try {
    await watcher.goto("/projects/P-1?features=active");
    await expect(
      watcher.getByRole("heading", { name: "Delivery platform" }),
    ).toBeVisible();
    await createFeature(editor, title);
    // rail のツリーと feature 一覧の両方に出るので、一覧の行で確かめる。
    await expect(
      watcher.locator(".feature-list-row").filter({ hasText: title }),
    ).toBeVisible();
  } finally {
    await context.close();
  }
});
