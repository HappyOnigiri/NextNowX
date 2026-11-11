import { expect, test } from "@playwright/test";
import {
  addTask,
  createFeature,
  e2eBaseURL,
  guardBrowserErrors,
} from "./helpers";

test.use({ baseURL: e2eBaseURL });
guardBrowserErrors();

test("warns about a linked PR and saves Work with manual statuses after detaching", async ({
  page,
}) => {
  await createFeature(page, `Task kind ${crypto.randomUUID()}`);
  await addTask(page, "Kind lifecycle");
  const node = page.locator(".task-node").filter({ hasText: "Kind lifecycle" });
  await node.getByRole("button", { name: "Edit Kind lifecycle" }).click();
  const inspector = page.getByRole("complementary", { name: "Task inspector" });
  await expect(inspector.getByLabel("Kind", { exact: true })).toHaveValue("1");
  await inspector
    .getByPlaceholder("https://github.com/org/repo/pull/42")
    .fill(`https://github.com/acme/nnx/pull/${Date.now()}`);
  await inspector.getByRole("button", { name: "Attach" }).click();
  await expect(inspector.getByRole("button", { name: "Detach" })).toBeVisible();
  await inspector
    .getByLabel("Kind", { exact: true })
    .selectOption({ label: "Work" });
  const warning = page.getByRole("dialog", {
    name: "Detach the pull request first",
  });
  await expect(warning).toBeVisible();
  await warning.getByRole("button", { name: "Close", exact: true }).click();
  await expect(inspector.getByLabel("Kind", { exact: true })).toHaveValue("1");
  await inspector.getByRole("button", { name: "Detach" }).click();
  await expect(inspector.getByRole("button", { name: "Attach" })).toBeVisible();
  await inspector
    .getByLabel("Kind", { exact: true })
    .selectOption({ label: "Work" });
  await expect(inspector.getByLabel("Status").locator("option")).toHaveCount(4);
  await inspector.getByLabel("Status").selectOption({ label: "In progress" });
  await inspector.getByRole("button", { name: "Save task" }).click();
  await expect(inspector.getByRole("button", { name: "Attach" })).toHaveCount(
    0,
  );
  await expect(
    inspector.getByRole("button", { name: "Save task" }),
  ).toBeDisabled();
  await page.reload();
  await node.getByRole("button", { name: "Edit Kind lifecycle" }).click();
  await expect(inspector.getByLabel("Kind", { exact: true })).toHaveValue("2");
  await expect(inspector.getByLabel("Status")).toHaveValue("2");
  await inspector.getByLabel("Status").selectOption({ label: "Completed" });
  await inspector.getByRole("button", { name: "Save task" }).click();
  await expect(
    inspector.getByRole("button", { name: "Save task" }),
  ).toBeDisabled();
});
