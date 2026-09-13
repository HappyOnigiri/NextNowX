import type { PullRequest, Task } from "../gen/prx/v1/prx_pb";
import { normalizeText } from "../task-search";

// グラフの検索は入力全体を 1 つの語として扱う。/tasks の修飾子文法は持ち込まず、
// 正規化だけを task-search と共有する。docs/design/webui.md を参照。

// 単一 feature のグラフでは feature の id と title が全件に一致してしまうため、
// 照合はタスクとその PR に限る。
export function matchesGraphSearch(
  task: Task,
  pullRequest: PullRequest | undefined,
  query: string,
): boolean {
  const needle = normalizeText(query).trim();
  if (needle === "") return true;
  const values = [
    task.id,
    task.title,
    task.scope,
    task.assignee,
    pullRequest?.host,
    pullRequest?.owner,
    pullRequest?.repository,
    pullRequest?.author,
  ];
  return values.some(
    (value) => value !== undefined && normalizeText(value).includes(needle),
  );
}
