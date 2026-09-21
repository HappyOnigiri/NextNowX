import { useMemo, useState } from "react";
import { type Task } from "../gen/prx/v1/prx_pb";
import {
  batchCandidates,
  prunedSelection,
  type BatchCandidate,
  type BatchPromptKind,
} from "./batchPromptTasks";

export interface BatchSelection {
  kind: BatchPromptKind;
  includeBlocked: boolean;
  includeDesigned: boolean;
  includeUndesigned: boolean;
  candidates: BatchCandidate[];
  selected: ReadonlySet<string>;
  allSelected: boolean;
  // targets は候補の並び順に従う選択。リストの並びが読み手に見える順序なので、
  // プロンプトはチェックした順ではなくその並びで task を挙げる。
  targets: string[];
  changeKind: (kind: BatchPromptKind) => void;
  changeIncludeBlocked: (include: boolean) => void;
  changeIncludeDesigned: (include: boolean) => void;
  changeIncludeUndesigned: (include: boolean) => void;
  toggle: (taskId: string) => void;
  toggleAll: () => void;
}

// useBatchSelection は一括ダイアログの候補集合と選択を持つ。表示範囲が狭まる
// たび、渡せなくなった task を選択から刈り込む。
export function useBatchSelection(tasks: Task[]): BatchSelection {
  // 既定は従来どおりの一括実装。このボタンで運ばれてきた作業がそれである。
  const [kind, setKind] = useState<BatchPromptKind>("implementation");
  // どのトグルもこの受け渡し限りの判断で保持する設定ではないため、
  // 開くたび、そしてタブを移るたびにオフから始める。
  const [includeBlocked, setIncludeBlocked] = useState(false);
  const [includeDesigned, setIncludeDesigned] = useState(false);
  const [includeUndesigned, setIncludeUndesigned] = useState(false);
  const [selected, setSelected] = useState<ReadonlySet<string>>(new Set());

  const candidates = useMemo(
    () =>
      batchCandidates(tasks, {
        kind,
        includeBlocked,
        includeDesigned,
        includeUndesigned,
      }),
    [tasks, kind, includeBlocked, includeDesigned, includeUndesigned],
  );
  const targets = useMemo(
    () =>
      candidates
        .filter((candidate) => selected.has(candidate.task.id))
        .map((candidate) => candidate.task.id),
    [candidates, selected],
  );

  function prune(options: {
    includeBlocked: boolean;
    includeDesigned: boolean;
    includeUndesigned: boolean;
  }) {
    setSelected((current) =>
      prunedSelection(batchCandidates(tasks, { kind, ...options }), current),
    );
  }

  return {
    kind,
    includeBlocked,
    includeDesigned,
    includeUndesigned,
    candidates,
    selected,
    allSelected: candidates.length > 0 && selected.size === candidates.length,
    targets,
    changeKind: (next) => {
      if (next === kind) return;
      // 候補集合が入れ替わるので、前のタブの選択もトグルも持ち越さない。
      setKind(next);
      setIncludeBlocked(false);
      setIncludeDesigned(false);
      setIncludeUndesigned(false);
      setSelected(new Set());
    },
    changeIncludeBlocked: (include) => {
      setIncludeBlocked(include);
      prune({ includeBlocked: include, includeDesigned, includeUndesigned });
    },
    changeIncludeDesigned: (include) => {
      setIncludeDesigned(include);
      prune({ includeBlocked, includeDesigned: include, includeUndesigned });
    },
    changeIncludeUndesigned: (include) => {
      setIncludeUndesigned(include);
      prune({ includeBlocked, includeDesigned, includeUndesigned: include });
    },
    toggle: (taskId) => {
      setSelected((current) => {
        const next = new Set(current);
        // task を外すとその上に積まれた作業が土台を失うので、
        // 選択を残さず刈り込む。
        if (next.delete(taskId)) return prunedSelection(candidates, next);
        next.add(taskId);
        return next;
      });
    },
    toggleAll: () => {
      // 候補はすべて同時に選べる。バッチで運べないブロッカーを持つものは
      // 候補の時点で除外済み。
      setSelected((current) =>
        current.size === candidates.length
          ? new Set()
          : new Set(candidates.map((candidate) => candidate.task.id)),
      );
    },
  };
}
