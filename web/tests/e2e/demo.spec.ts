import { expect, test } from "@playwright/test";

const browserErrors: string[] = [];
const e2ePort = process.env["PRX_E2E_PORT"];
if (!e2ePort) throw new Error("Playwright did not capture the E2E server port");

test.use({
  baseURL: `http://127.0.0.1:${e2ePort}`,
});

test.beforeEach(({ page }) => {
  browserErrors.length = 0;
  page.on("console", (message) => {
    if (message.type() === "error" || message.type() === "warning")
      browserErrors.push(`console ${message.type()}: ${message.text()}`);
  });
  page.on("pageerror", (error) =>
    browserErrors.push(`pageerror: ${error.message}`),
  );
  page.on("requestfailed", (request) =>
    browserErrors.push(
      `requestfailed: ${request.method()} ${request.url()} ${request.failure()?.errorText}`,
    ),
  );
});

test.afterEach(() => {
  expect(browserErrors, browserErrors.join("\n")).toEqual([]);
});

test("keeps the dismissed demo warning hidden until the server restarts", async ({
  page,
}) => {
  await page.goto("/");
  await page
    .getByRole("button", {
      name: "Hide the demo notice until the demo server restarts",
    })
    .click();
  await expect(page.getByRole("status")).toBeHidden();
  await expect(page.locator(".app-shell")).not.toHaveAttribute("data-demo");

  await page.reload();
  await expect(page.locator(".app-shell")).toBeVisible();
  await expect(page.getByRole("status")).toBeHidden();

  // サーバを起動し直すと HTML の ID が変わる。保存済みの ID を別の値に書き換えて
  // 同じ状況を作る。
  await page.evaluate(() => {
    localStorage.setItem(
      "prx.webui.demoNoticeDismissedSession",
      "restarted-server",
    );
  });
  await page.reload();
  await expect(page.getByRole("status")).toBeVisible();
});
