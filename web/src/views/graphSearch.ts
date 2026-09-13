import type { PullRequest, Task } from "../gen/prx/v1/prx_pb";
import { normalizeText } from "../task-search";

// グラフの検索は空白区切りの語をすべて含むタスクだけを残す。/tasks の修飾子
// 文法や引用符は持ち込まず、正規化だけを task-search と共有する。
// docs/design/webui.md を参照。

// 単一 feature のグラフでは feature の id と title が全件に一致してしまうため、
// 照合はタスクとその PR に限る。
export function matchesGraphSearch(
  task: Task,
  pullRequest: PullRequest | undefined,
  query: string,
): boolean {
  // 区切りは正規化の後で見る。NFKC が全角空白を半角に畳むので、全角で区切って
  // も同じ結果になる。
  const needles = normalizeText(query).split(/\s+/u).filter(Boolean);
  if (needles.length === 0) return true;
  const values = [
    task.id,
    task.title,
    task.scope,
    task.assignee,
    pullRequest?.host,
    pullRequest?.owner,
    pullRequest?.repository,
    pullRequest?.author,
  ].flatMap((value) => (value === undefined ? [] : [normalizeText(value)]));
  return needles.every((needle) =>
    values.some((value) => value.includes(needle)),
  );
}
