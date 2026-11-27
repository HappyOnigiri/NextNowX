import { expect, test, type Locator, type Page } from "@playwright/test";
import {
  createFeature,
  e2eBaseURL,
  guardBrowserErrors,
  settleGraph,
} from "./helpers";

test.use({ baseURL: e2eBaseURL });
guardBrowserErrors();

async function createGroupFromInspector(
  page: Page,
  inspector: Locator,
  name: string,
) {
  const select = inspector.getByLabel("Task group");
  await select.selectOption({ label: "Add new group…" });
  const dialog = page.getByRole("dialog", { name: "Create group" });
  await dialog.getByLabel("Group name").fill(name);
  await dialog.getByRole("button", { name: "Create group" }).click();
  await expect(select.locator("option:checked")).toHaveText(name);
}

async function groupTask(page: Page, title: string, name: string) {
  const inspector = await openInspector(page, title);
  await createGroupFromInspector(page, inspector, name);
  await inspector.getByRole("button", { name: "Save task" }).click();
  await expect(
    inspector.getByRole("button", { name: "Save task" }),
  ).toBeDisabled();
  await inspector.getByRole("button", { name: "Close inspector" }).click();
  await settleGraph(page);
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
  context,
}) => {
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
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
  await expect(
    page.getByRole("button", { name: "Task groups", exact: true }),
  ).toHaveCount(0);
  const hidden = `Filtered member ${prefix}`;
  await addTask(page, hidden);
  await groupTask(page, hidden, name);
  const cardTitle = page.getByRole("link", { name, exact: true });
  const cardHeader = page
    .locator(".graph-task-group")
    .locator(".graph-task-group-header");
  await expect(cardHeader).toHaveCount(1);
  await assertWrappedGroupTitle(
    cardTitle.locator(".task-group-link-name"),
    name,
  );
  const headerBox = await cardHeader.boundingBox();
  const rowBox = await page.locator(".graph-group-row").boundingBox();
  expect(
    headerBox && rowBox && rowBox.y >= headerBox.y + headerBox.height - 1,
  ).toBe(true);

  await cardHeader.getByRole("button", { name: "Copy group name" }).click();
  await expect
    .poll(() => page.evaluate(() => navigator.clipboard.readText()))
    .toBe(name);
  await expect(page).toHaveURL(featureURL);

  await cardTitle.click();
  const wideViewport = page.viewportSize();
  await page.setViewportSize({ width: 390, height: 844 });
  const detailTitle = page.getByRole("heading", { name, exact: true });
  await assertWrappedGroupTitle(detailTitle, name);
  if (wideViewport) await page.setViewportSize(wideViewport);
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
  await createGroupFromInspector(page, inspector, empty);
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
  await addTask(page, other);
  await groupTask(page, other, name);

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
  await page.setViewportSize({ width: 1280, height: 844 });
  await settleGraph(page);
  await page.locator(".react-flow__controls-fitview").click();
  await page.getByRole("button", { name: `Edit ${name}`, exact: true }).click();
  const edit = page.getByRole("dialog", {
    name: `Edit ${name}`,
    exact: true,
  });
  await edit.getByLabel("Group name", { exact: true }).fill(renamed);
  const palette = edit.getByRole("group", { name: "Card color" });
  const defaultColor = palette.getByRole("radio", { name: "Default" });
  await expect(defaultColor).toBeChecked();
  await page.keyboard.press("Tab");
  await expect(defaultColor).toBeFocused();
  await page.keyboard.press("ArrowRight");
  await page.keyboard.press("ArrowRight");
  await expect(palette.getByRole("radio", { name: "Blue" })).toBeChecked();
  await edit.getByRole("button", { name: "Save group", exact: true }).click();
  await expect(
    page.getByRole("link", { name: renamed, exact: true }),
  ).toBeVisible();
  const coloredFrame = page
    .locator(".graph-task-group")
    .filter({ hasText: renamed });
  await expect(coloredFrame).toHaveAttribute("style", /#3b82f6/);
  await page.goto(search);
  await expect(
    results.locator(".task-card-group").filter({ hasText: renamed }),
  ).toHaveAttribute("style", /#3b82f6/);
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
  await settleGraph(page);
  await page.locator(".react-flow__controls-fitview").click();
  await expect(
    page.getByRole("button", { name: `Delete ${renamed}` }),
  ).toHaveCount(0);
  const editRenamed = page.getByRole("dialog", { name: `Edit ${renamed}` });
  await page.getByRole("button", { name: `Edit ${renamed}` }).click();
  await expect(editRenamed.getByRole("radio", { name: "Blue" })).toBeChecked();
  const deleteGroup = editRenamed.getByRole("button", { name: "Delete group" });
  await deleteGroup.click();
  const deletion = page.getByRole("dialog", { name: `Delete ${renamed}` });
  await expect(deletion).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(deletion).toHaveCount(0);
  await expect(deleteGroup).toBeFocused();
  await deleteGroup.click();
  await deletion.getByRole("button", { name: "Delete group" }).click();
  await expect(editRenamed).toHaveCount(0);
  await expect(
    page.getByRole("link", { name: renamed, exact: true }),
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

async function assertWrappedGroupTitle(title: Locator, name: string) {
  await expect(title).toBeVisible();
  await expect(title).toHaveText(name);
  const layout = await title.evaluate((element) => {
    const range = document.createRange();
    range.selectNodeContents(element);
    const bounds = element.getBoundingClientRect();
    const lines = [...range.getClientRects()];
    return {
      lineCount: lines.length,
      fits: lines.every(
        (line) =>
          line.left >= bounds.left - 1 &&
          line.right <= bounds.right + 1 &&
          line.top >= bounds.top - 1 &&
          line.bottom <= bounds.bottom + 1,
      ),
    };
  });
  expect(layout.lineCount).toBeGreaterThan(1);
  expect(layout.fits).toBe(true);
}

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

test("switches between aggregated group edges and individual task edges", async ({
  page,
}) => {
  await createFeature(page, `Grouped graph ${crypto.randomUUID()}`);
  await addTask(page, "Collect inputs");
  await groupTask(page, "Collect inputs", "Checkout flow");
  await addTask(page, "Deliver inputs");
  await groupTask(page, "Deliver inputs", "Delivery flow");
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
  const taskDependencies = [
    [collect, deliver],
    [deliver, publish],
    [collect, publish],
    [collect, ungrouped],
    [publish, ungrouped],
    [prepare, collect],
    [prepare, publish],
  ];
  for (const [blockerTaskId, blockedTaskId] of taskDependencies) {
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
  const groupToggle = page.getByRole("switch", { name: "Show groups" });
  await expect(groupToggle).toBeChecked();
  await groupToggle.click();
  await settleGraph(page);
  await expect(cards).toHaveCount(5);
  await expect(rows).toHaveCount(0);
  await expect(stage.locator(".graph-task-group")).toHaveCount(0);
  await expect(stage.locator(".react-flow__edge")).toHaveCount(
    taskDependencies.length,
  );
  for (const [source, target] of taskDependencies) {
    await expect(
      stage.locator(
        `.react-flow__edge[data-id="${source}-${target}"] .react-flow__edge-path`,
      ),
    ).toHaveAttribute("d", /^M[\d.]+ [\d.]+ L/);
  }
  await page.reload();
  await expect(groupToggle).not.toBeChecked();
  await expect(cards).toHaveCount(5);
  await groupToggle.click();
  await settleGraph(page);
  await expect(stage.locator(".react-flow__edge")).toHaveCount(4);
  await checkout.click();
  await expect(groupToggle).toHaveCount(0);
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
  await expect(
    rows.getByRole("img", { name: "Hidden dependents: Ungrouped task" }),
  ).toBeVisible();

  await page.keyboard.press("Escape");
  await settleGraph(page);
  const inspector = await openInspector(page, "Prepare inputs");
  await inspector.getByLabel("Status").selectOption({ label: "Completed" });
  await inspector.getByRole("button", { name: "Save task" }).click();
  await expect(
    inspector.getByRole("button", { name: "Save task" }),
  ).toBeDisabled();
  await inspector.getByRole("button", { name: "Close inspector" }).click();
  await page.getByRole("switch", { name: "Hide completed" }).check();
  await expect(cards).toHaveCount(1);
  await expect(
    rows
      .filter({ hasText: "Collect inputs" })
      .getByRole("img", { name: "Hidden blockers: Prepare inputs" }),
  ).toBeVisible();
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

test("keeps the detail graph visible beside a scrollable long group name", async ({
  page,
  context,
}) => {
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
  await createFeature(page, `Long group ${crypto.randomUUID()}`);
  await addTask(page, "Verify detail layout");
  const name = "LongGroupHeading".repeat(120);
  await groupTask(page, "Verify detail layout", name);
  const groupURL = await page
    .getByRole("link", { name, exact: true })
    .getAttribute("href");
  if (!groupURL) throw new Error("Group detail URL missing");
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto(groupURL);

  const groupId = new URL(page.url()).searchParams.get("group");
  if (!groupId) throw new Error("Group ID missing from detail URL");
  const copyGroupId = page.getByRole("button", { name: "Copy Group ID" });
  await expect(copyGroupId).toHaveText(groupId);
  await expect(copyGroupId).toBeInViewport();
  await copyGroupId.click();
  await expect
    .poll(() => page.evaluate(() => navigator.clipboard.readText()))
    .toBe(groupId);

  const heading = page.getByRole("heading", { name, exact: true });
  await expect(heading).toBeVisible();
  expect(
    await heading.evaluate((element) => element.scrollHeight),
  ).toBeGreaterThan(await heading.evaluate((element) => element.clientHeight));
  await heading.focus();
  await expect(heading).toBeFocused();
  await page.keyboard.press("End");
  await expect
    .poll(() =>
      heading.evaluate(
        (element) =>
          element.scrollHeight - element.clientHeight - element.scrollTop,
      ),
    )
    .toBeLessThanOrEqual(1);
  await expect(page.locator(".graph-stage .task-node")).toBeInViewport();
});

test("opens a group task from the overview and uses its references and prompt", async ({
  page,
  context,
}) => {
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
  await createFeature(page, `Grouped task details ${crypto.randomUUID()}`);
  const title = "Collect and validate the inputs for the checkout flow";
  await addTask(page, title);
  await groupTask(page, title, "Checkout flow");
  const row = page.locator(".graph-group-row").getByRole("button", {
    name: title,
  });
  const overview = page.url();
  await row.click();
  await expect(page).toHaveURL(overview);
  const inspector = page.getByRole("complementary", { name: "Task inspector" });
  const heading = inspector.getByRole("heading", { name: title });
  await expect(heading).toBeVisible();
  const viewport = page.viewportSize();
  for (const width of [1280, 390]) {
    await page.setViewportSize({ width, height: 844 });
    const layout = await heading.evaluate((element) => {
      const header = element.closest("header");
      if (!header) throw new Error("Task title header missing");
      const style = getComputedStyle(header);
      const range = document.createRange();
      range.selectNodeContents(element);
      return {
        available:
          header.clientWidth -
          parseFloat(style.paddingLeft) -
          parseFloat(style.paddingRight),
        width: element.getBoundingClientRect().width,
        lines: range.getClientRects().length,
      };
    });
    expect(layout.width).toBeCloseTo(layout.available, 0);
    expect(layout.lines).toBeLessThanOrEqual(2);
  }
  if (viewport) await page.setViewportSize(viewport);
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

test("attaches a group reference and opens it from the group card", async ({
  page,
}) => {
  await createFeature(page, `Group references ${crypto.randomUUID()}`);
  const overview = page.url();
  await addTask(page, "Plan the rollout");
  const inspector = await openInspector(page, "Plan the rollout");
  await createGroupFromInspector(page, inspector, "Rollout");
  await inspector.getByRole("button", { name: "Close inspector" }).click();
  await page.getByRole("button", { name: "Discard", exact: true }).click();
  await settleGraph(page);
  const group = page.locator(".graph-task-group");
  await expect(group.getByRole("link", { name: "Rollout" })).toBeVisible();
  await group.getByRole("link", { name: "Rollout" }).click();

  const viewport = page.viewportSize();
  await page.setViewportSize({ width: 390, height: 844 });
  const references = page.getByRole("button", {
    name: "Group references",
    exact: true,
  });
  const panel = page.getByRole("region", { name: "Group references" });
  await references.click();
  await panel
    .getByRole("button", { name: "Add reference", exact: true })
    .click();
  const add = page.getByRole("dialog", { name: "Add group reference" });
  await add.getByRole("tab", { name: "Markdown" }).click();
  await add.getByLabel("Reference title (optional)").fill("Rollout notes");
  await add.getByLabel("Markdown content").fill("# Rollout checklist");
  await add.getByRole("button", { name: "Add reference" }).click();
  await expect(add).toBeHidden();
  await page.reload();
  await references.click();
  await panel
    .getByRole("button", { name: /Rollout notes/ })
    .first()
    .click();
  const preview = page.getByRole("dialog", { name: "Rollout notes" });
  await expect(
    preview.getByRole("heading", { name: "Rollout checklist" }),
  ).toBeVisible();
  await preview.getByRole("button", { name: "Close" }).click();
  await references.click();
  await panel.getByRole("button", { name: /Edit.*Rollout notes/ }).click();
  const editReference = page.getByRole("dialog", {
    name: "Edit group reference",
  });
  await editReference
    .getByLabel("Reference title (optional)")
    .fill("Reviewed notes");
  await editReference.getByRole("button", { name: "Save" }).click();
  await expect(editReference).toBeHidden();
  await references.click();
  await expect(
    panel.getByText("Reviewed notes", { exact: true }),
  ).toBeVisible();

  if (viewport) await page.setViewportSize(viewport);
  await page.getByRole("link", { name: "Back to feature" }).click();
  await settleGraph(page);
  const showReferences = group.getByRole("button", {
    name: "Group references for Rollout",
  });
  await showReferences.click();
  const groupReferences = page.getByRole("dialog", {
    name: "Group references for Rollout",
  });
  await expect(page).toHaveURL(overview);
  await groupReferences.getByRole("button", { name: /Reviewed notes/ }).click();
  await expect(
    page
      .getByRole("dialog", { name: "Reviewed notes" })
      .getByRole("heading", { name: "Rollout checklist" }),
  ).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(groupReferences).toBeVisible();
  await page.keyboard.press("Escape");
  await expect(groupReferences).toBeHidden();
  await expect(showReferences).toBeFocused();

  await group.getByRole("button", { name: "Edit Rollout" }).click();
  await page
    .getByRole("dialog", { name: "Edit Rollout" })
    .getByRole("button", { name: "Delete group" })
    .click();
  const deletion = page.getByRole("dialog", { name: "Delete Rollout" });
  await expect(deletion).toContainText("Group references will be deleted");
  await deletion.getByRole("button", { name: "Delete group" }).click();
  await expect(group).toHaveCount(0);
  await expect(
    page.getByRole("button", { name: "Edit Plan the rollout" }),
  ).toBeVisible();
});
