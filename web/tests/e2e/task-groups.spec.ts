import { expect, test, type Locator, type Page } from "@playwright/test";
import { createFeature, e2eBaseURL, guardBrowserErrors } from "./helpers";

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
  let inspector = await openInspector(page, hidden);
  await inspector.getByLabel("Status").selectOption({ label: "Completed" });
  await inspector.getByRole("button", { name: "Save task" }).click();
  await expect(
    inspector.getByRole("button", { name: "Save task" }),
  ).toBeDisabled();

  await addTask(page, looseFirst);
  await addTask(page, groupedFirst, name);
  await addTask(page, looseSecond);
  await addTask(page, groupedSecond);
  inspector = await openInspector(page, groupedSecond);
  const save = inspector.getByRole("button", { name: "Save task" });
  const select = inspector.getByLabel("Task group");
  for (const value of [name, empty, "Ungrouped", name]) {
    await select.selectOption({ label: value });
    await save.click();
    await expect(save).toBeDisabled();
  }

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
    renamedGroup.getByRole("link", { name: groupedFirst, exact: true }),
  ).toBeVisible();
  await expect(
    renamedGroup.getByRole("link", { name: groupedSecond, exact: true }),
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
      await expect(
        entry.getByRole("link", { name: title, exact: true }),
      ).toHaveCount(1);
    if (titles.length === 2) {
      await expect(entry.locator(".task-card-title a")).toHaveText(titles);
      await expect(
        entry.getByRole("heading", { name, exact: true }),
      ).toBeVisible();
    }
  }
}
