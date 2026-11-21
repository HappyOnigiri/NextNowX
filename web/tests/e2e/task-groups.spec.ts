import { expect, test, type Locator, type Page } from "@playwright/test";
import {
  createFeature,
  e2eBaseURL,
  guardBrowserErrors,
  settleGraph,
} from "./helpers";

test.use({ baseURL: e2eBaseURL });
guardBrowserErrors();

async function openGroups(page: Page) {
  await page.getByRole("button", { name: "Task groups" }).click();
  return page.getByRole("dialog", { name: "Task groups", exact: true });
}

async function addGroup(manager: Locator, name: string) {
  await manager.getByLabel("New group name").fill(name);
  await manager.getByRole("button", { name: "Create group" }).click();
  await expect(
    manager.getByRole("button", { name: `Rename ${name}` }),
  ).toBeVisible();
}

async function addTask(page: Page, title: string, group?: string) {
  await page
    .getByRole("button", { name: "Add task", exact: true })
    .first()
    .click();
  const form = page.getByRole("form", { name: "Create task" });
  await form.getByLabel("Title").fill(title);
  if (group) await form.getByLabel("Task group").selectOption({ label: group });
  await form.getByRole("button", { name: "Add task" }).click();
  await expect(form).toBeHidden();
}

async function openInspector(page: Page, title: string) {
  await page
    .getByRole("button", { name: `Edit ${title}`, exact: true })
    .click();
  return page.getByRole("complementary", { name: "Task inspector" });
}

test("manages membership and shows the same ordered group frames in search and ready tasks", async ({
  page,
}) => {
  const prefix = crypto.randomUUID().slice(0, 8);
  const name = `Checkout${prefix}${"LongGroupHeading".repeat(5)}`;
  const renamed = `Delivery ${prefix}`;
  const empty = `Empty ${prefix}`;
  const looseFirst = `${prefix} Loose first`;
  const groupedFirst = `${prefix} Group first`;
  const looseSecond = `${prefix} Loose second`;
  const groupedSecond = `${prefix} Group second`;
  const other = `${prefix} Other`;
  const search = `/tasks?${new URLSearchParams({ q: `${prefix} task-status:ready` })}`;

  await createFeature(page, `Group manager ${prefix}`);
  const featureURL = page.url();
  let manager = await openGroups(page);
  await addGroup(manager, name);
  await addGroup(manager, empty);
  await manager.getByRole("button", { name: "Close", exact: true }).click();

  const hidden = `Filtered member ${prefix}`;
  await addTask(page, hidden, name);
  await page.getByRole("link", { name, exact: true }).click();
  let inspector = await openInspector(page, hidden);
  await inspector.getByLabel("Status").selectOption({ label: "Completed" });
  await inspector.getByRole("button", { name: "Save task" }).click();
  await expect(
    inspector.getByRole("button", { name: "Save task" }),
  ).toBeDisabled();

  await page.getByRole("link", { name: "Back to feature" }).click();
  await addTask(page, looseFirst);
  await addTask(page, groupedFirst, name);
  await addTask(page, looseSecond);
  await addTask(page, groupedSecond);
  inspector = await openInspector(page, groupedSecond);
  await inspector.getByLabel("Task group").selectOption({ label: empty });
  await inspector.getByRole("button", { name: "Save task" }).click();
  await expect(
    inspector.getByRole("button", { name: "Save task" }),
  ).toBeDisabled();
  await inspector.getByRole("button", { name: "Close inspector" }).click();
  await settleGraph(page);
  await page.locator(".react-flow__controls-fitview").click();
  await settleGraph(page);
  await page.getByRole("link", { name: empty, exact: true }).click();
  inspector = await openInspector(page, groupedSecond);
  await inspector.getByLabel("Task group").selectOption({ label: "Ungrouped" });
  await inspector.getByRole("button", { name: "Save task" }).click();
  await expect(inspector).toHaveCount(0);
  await page.getByRole("link", { name: "Back to feature" }).click();
  inspector = await openInspector(page, groupedSecond);
  await inspector.getByLabel("Task group").selectOption({ label: name });
  await inspector.getByRole("button", { name: "Save task" }).click();
  await expect(
    page.locator(".graph-group-row").getByText(groupedSecond, { exact: true }),
  ).toBeVisible();

  await createFeature(page, `Second feature ${prefix}`);
  manager = await openGroups(page);
  await addGroup(manager, name);
  await manager.getByRole("button", { name: "Close", exact: true }).click();
  await addTask(page, other, name);

  await page.goto(search);
  const results = page.getByRole("list", { name: "Matching active tasks" });
  await assertGroupedList(results, prefix, name, empty, [
    [looseFirst],
    [groupedFirst, groupedSecond],
    [looseSecond],
    [other],
  ]);
  await results.getByRole("link", { name: groupedFirst, exact: true }).click();
  inspector = page.getByRole("complementary", { name: "Task inspector" });
  await expect(
    inspector.getByRole("heading", { name: groupedFirst, exact: true }),
  ).toBeVisible();
  await page.reload();
  await expect(inspector).toBeVisible();
  await inspector.getByRole("button", { name: "Close inspector" }).click();
  await expect(page).not.toHaveURL(/task=/);
  await page.goBack();
  await expect(results).toBeVisible();
  await page.setViewportSize({ width: 390, height: 844 });
  const groupHeading = results
    .getByRole("heading", { name, exact: true })
    .first();
  await expect(groupHeading).toBeVisible();
  const fits = await groupHeading.evaluate((heading) => {
    const frame = heading.parentElement;
    return (
      frame &&
      getComputedStyle(frame).borderInlineStartWidth === "1px" &&
      heading.scrollWidth === heading.clientWidth &&
      frame.getBoundingClientRect().right <= window.innerWidth
    );
  });
  expect(fits).toBe(true);

  await page.goto("/");
  const ready = page
    .getByRole("region", { name: "Ready to start" })
    .locator("ol")
    .first();
  await assertGroupedList(ready, prefix, name, empty, [
    [looseFirst],
    [groupedFirst, groupedSecond],
    [looseSecond],
    [other],
  ]);

  await page.goto(featureURL);
  manager = await openGroups(page);
  await manager.getByRole("button", { name: `Rename ${name}` }).click();
  await manager.getByLabel("Group name", { exact: true }).fill(renamed);
  await manager
    .getByRole("button", { name: "Rename group", exact: true })
    .click();
  await expect(
    manager.getByRole("button", { name: `Rename ${renamed}` }),
  ).toBeVisible();
  await page.goto(search);
  const renamedGroup = results.getByRole("list", {
    name: renamed,
    exact: true,
  });
  await expect(
    renamedGroup.getByText(groupedFirst, { exact: true }),
  ).toBeVisible();
  await expect(
    renamedGroup.getByText(groupedSecond, { exact: true }),
  ).toBeVisible();

  await page.goto(featureURL);
  manager = await openGroups(page);
  await manager.getByRole("button", { name: `Delete ${renamed}` }).click();
  await page
    .getByRole("dialog", { name: `Delete ${renamed}` })
    .getByRole("button", { name: "Delete group" })
    .click();
  await expect(
    manager.getByRole("button", { name: `Delete ${renamed}` }),
  ).toHaveCount(0);
  await page.goto(search);
  await expect(renamedGroup).toHaveCount(0);
  await expect(
    results.getByRole("link", { name: groupedFirst, exact: true }),
  ).toBeVisible();
  await expect(
    results.getByRole("link", { name: groupedSecond, exact: true }),
  ).toBeVisible();
  await expect(results.getByRole("list", { name, exact: true })).toHaveCount(1);
});

async function assertGroupedList(
  list: Locator,
  prefix: string,
  name: string,
  empty: string,
  expected: string[][],
) {
  const entries = list.locator(":scope > li").filter({ hasText: prefix });
  await expect(entries).toHaveCount(expected.length);
  await expect(list.getByRole("heading", { name, exact: true })).toHaveCount(2);
  await expect(
    list.getByRole("heading", { name: empty, exact: true }),
  ).toHaveCount(0);
  for (const [index, titles] of expected.entries()) {
    const entry = entries.nth(index);
    for (const title of titles)
      await expect(entry.getByText(title, { exact: true })).toHaveCount(1);
    if (titles.length === 2) {
      await expect(entry.locator(".task-group-row-title")).toHaveText(titles);
      await expect(
        entry.getByRole("heading", { name, exact: true }),
      ).toBeVisible();
    }
  }
}

test("combines external dependencies at each group card and shows individual edges inside", async ({
  page,
}) => {
  await createFeature(page, `Grouped graph ${crypto.randomUUID()}`);
  const manager = await openGroups(page);
  await addGroup(manager, "Checkout flow");
  await addGroup(manager, "Delivery flow");
  await addGroup(manager, "Empty group");
  await manager.getByRole("button", { name: "Close", exact: true }).click();
  await addTask(page, "Collect inputs", "Checkout flow");
  await addTask(page, "Deliver inputs", "Delivery flow");
  await addTask(page, "Publish inputs", "Checkout flow");
  await addTask(page, "Ungrouped task");
  await addTask(page, "Prepare inputs");

  const stage = page.locator(".graph-stage");
  const cards = stage.locator(".task-node");
  const rows = stage.locator(".graph-group-row");
  await expect(cards).toHaveCount(2);
  await expect(rows).toHaveCount(3);
  await expect(stage.locator(".graph-task-group")).toHaveCount(2);
  const collect = await graphTaskId(stage, "Collect inputs");
  const deliver = await graphTaskId(stage, "Deliver inputs");
  const publish = await graphTaskId(stage, "Publish inputs");
  const ungrouped = await graphTaskId(stage, "Ungrouped task");
  const prepare = await graphTaskId(stage, "Prepare inputs");

  // タスク単位では DAG でも、グループ単位では往復になる依存を作る。
  for (const [blockerTaskId, blockedTaskId] of [
    [collect, deliver],
    [deliver, publish],
    [collect, publish],
    [collect, ungrouped],
    [publish, ungrouped],
    [prepare, collect],
    [prepare, publish],
  ]) {
    const response = await page.request.post(
      "/nnx.v1.NNXService/AddDependency",
      {
        headers: { "Connect-Protocol-Version": "1" },
        data: { blockerTaskId, blockedTaskId },
      },
    );
    expect(response.ok()).toBe(true);
  }
  await page.reload();
  await settleGraph(page);
  const checkout = stage.getByRole("link", { name: "Checkout flow" });
  const checkoutGroup = new URL(
    (await checkout.getAttribute("href")) ?? "",
    page.url(),
  ).searchParams.get("group");
  const deliveryGroup = new URL(
    (await stage
      .getByRole("link", { name: "Delivery flow" })
      .getAttribute("href")) ?? "",
    page.url(),
  ).searchParams.get("group");
  expect(checkoutGroup).toBeTruthy();
  expect(deliveryGroup).toBeTruthy();
  await expect(stage.locator(".react-flow__edge")).toHaveCount(4);
  for (const [source, target] of [
    [checkoutGroup, deliveryGroup],
    [deliveryGroup, checkoutGroup],
    [checkoutGroup, ungrouped],
    [prepare, checkoutGroup],
  ]) {
    await expect(
      stage.locator(
        `.react-flow__edge[data-id="${source}-${target}"] .react-flow__edge-path`,
      ),
    ).toHaveAttribute("d", /^M[\d.]+ [\d.]+ L/);
  }
  await checkout.click();
  await expect(
    page.getByRole("heading", { name: "Checkout flow" }),
  ).toBeVisible();
  await expect(cards).toHaveCount(2);
  await expect(rows).toHaveCount(0);
  await expect(
    stage.locator(`.react-flow__edge[data-id="${collect}-${publish}"]`),
  ).toBeVisible();
  await expect(stage.getByText("Deliver inputs", { exact: true })).toHaveCount(
    0,
  );
  await page.reload();
  await expect(cards).toHaveCount(2);

  await page.getByRole("button", { name: "Add task", exact: true }).click();
  await expect(
    page
      .getByRole("combobox", { name: "Task group" })
      .locator("option:checked"),
  ).toHaveText("Checkout flow");
  await page.getByRole("button", { name: "Cancel", exact: true }).click();
  await page.getByRole("link", { name: "Back to feature" }).click();
  await expect(rows).toHaveCount(3);

  await page.keyboard.press("ControlOrMeta+f");
  await page
    .getByRole("textbox", { name: "Search tasks on this graph" })
    .fill("Publish inputs");
  await expect(rows).toHaveCount(1);
  await expect(stage.locator(".graph-task-group")).toHaveCount(1);
});

async function graphTaskId(stage: Locator, title: string) {
  const id = await stage
    .locator(".react-flow__node")
    .filter({
      has: stage.page().getByText(title, { exact: true }),
    })
    .getAttribute("data-id");
  if (!id) throw new Error(`Task ID missing for ${title}`);
  return id;
}

test("opens a group task from the overview and uses its references and prompt", async ({
  page,
  context,
}) => {
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
  await createFeature(page, `Grouped task details ${crypto.randomUUID()}`);
  const manager = await openGroups(page);
  await addGroup(manager, "Checkout flow");
  await manager.getByRole("button", { name: "Close", exact: true }).click();
  await addTask(page, "Collect inputs", "Checkout flow");
  const row = page.locator(".graph-group-row").getByRole("button", {
    name: "Collect inputs",
  });
  const overview = page.url();
  await row.click();
  await expect(page).toHaveURL(overview);
  const inspector = page.getByRole("complementary", { name: "Task inspector" });
  await expect(
    inspector.getByRole("heading", { name: "Collect inputs" }),
  ).toBeVisible();
  await inspector
    .getByRole("button", { name: "Add reference", exact: true })
    .click();
  const attachment = page.getByRole("dialog", { name: "Add task reference" });
  await attachment.getByRole("tab", { name: "Local file" }).click();
  await attachment
    .getByLabel("Reference title (optional)")
    .fill("Checkout notes");
  await attachment.getByLabel("File path").fill("README.md");
  await attachment
    .getByRole("button", { name: "Add reference", exact: true })
    .click();
  await inspector
    .getByRole("button", { name: /Checkout notes/ })
    .first()
    .click();
  const preview = page.getByRole("dialog", { name: "Checkout notes" });
  await expect(preview).toBeVisible();
  await preview.getByRole("button", { name: "Copy file path" }).click();
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(
    "README.md",
  );
  await preview.getByRole("button", { name: "Close" }).click();

  await inspector.getByRole("button", { name: "Copy task prompt" }).click();
  const prompt = page.getByRole("dialog", { name: "Copy task prompt" });
  const text = prompt.getByLabel("Prompt preview");
  await expect(text).not.toHaveValue("");
  const value = await text.inputValue();
  await prompt
    .getByRole("button", { name: "Copy prompt", exact: true })
    .click();
  await expect(prompt.getByText("Design prompt copied.")).toBeVisible();
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(value);
});
