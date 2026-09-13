import { describe, expect, it } from "vitest";
import { matchesGraphSearch } from "../src/views/graphSearch";
import { makePullRequest, makeTask } from "./factories";

const task = makeTask({
  id: "T-7",
  title: "Ship API",
  scope: "Acceptance boundary",
  assignee: "Bob",
});
const pullRequest = makePullRequest({
  host: "github.com",
  owner: "acme",
  repository: "prx",
  author: "carol",
});

describe("matchesGraphSearch", () => {
  it.each([
    ["t-7", "id"],
    ["ship", "title"],
    ["boundary", "scope"],
    ["bob", "assignee"],
  ])("matches %s from the task %s", (query) => {
    expect(matchesGraphSearch(task, undefined, query)).toBe(true);
  });

  it.each([
    ["github.com", "host"],
    ["acme", "owner"],
    ["prx", "repository"],
    ["carol", "author"],
  ])("matches %s from the pull request %s", (query) => {
    expect(matchesGraphSearch(task, pullRequest, query)).toBe(true);
  });

  it("keeps every task while the query is empty or only spaces", () => {
    expect(matchesGraphSearch(task, undefined, "")).toBe(true);
    expect(matchesGraphSearch(task, undefined, "   ")).toBe(true);
  });

  it("normalizes case and full-width characters on both sides", () => {
    expect(matchesGraphSearch(task, undefined, "SHIP")).toBe(true);
    expect(matchesGraphSearch(task, undefined, "Ｓｈｉｐ")).toBe(true);
    expect(
      matchesGraphSearch(makeTask({ title: "ＡＰＩ 設計" }), undefined, "api"),
    ).toBe(true);
  });

  it("requires every space separated term, each from any field", () => {
    expect(matchesGraphSearch(task, undefined, "ship boundary")).toBe(true);
    expect(matchesGraphSearch(task, undefined, "  ship   bob  ")).toBe(true);
    expect(matchesGraphSearch(task, pullRequest, "ship carol")).toBe(true);
    expect(matchesGraphSearch(task, undefined, "ship storage")).toBe(false);
  });

  it("splits on full-width spaces too", () => {
    expect(matchesGraphSearch(task, undefined, "ship　boundary")).toBe(true);
    expect(matchesGraphSearch(task, undefined, "ship　storage")).toBe(false);
  });

  it("reports no match when nothing on the task carries the query", () => {
    expect(matchesGraphSearch(task, pullRequest, "storage")).toBe(false);
  });

  it("ignores the pull request fields when the task has no pull request", () => {
    expect(matchesGraphSearch(task, undefined, "acme")).toBe(false);
  });
});
