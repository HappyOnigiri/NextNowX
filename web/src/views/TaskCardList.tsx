import type { TaskGroup } from "../gen/nnx/v1/nnx_pb";
import { TaskCard, type TaskCardProps } from "./TaskCard";
import { TaskGroupLink } from "./TaskGroupLink";
import { TaskGroupRow } from "./TaskGroupRow";

type TaskListEntry =
  { card: TaskCardProps } | { group: TaskGroup; cards: TaskCardProps[] };

export function TaskCardList({
  cards,
  groups,
  className = "",
  label,
}: {
  cards: TaskCardProps[];
  groups: TaskGroup[];
  className?: string;
  label?: string;
}) {
  const groupsById = new Map(groups.map((group) => [group.id, group]));
  const grouped = new Map<string, TaskCardProps[]>();
  const entries: TaskListEntry[] = [];
  for (const card of cards) {
    const group = groupsById.get(card.task.groupId);
    if (group?.featureId !== card.task.featureId) {
      entries.push({ card });
      continue;
    }
    const members = grouped.get(group.id);
    if (members) {
      members.push(card);
      continue;
    }
    const first = [card];
    grouped.set(group.id, first);
    entries.push({ group, cards: first });
  }
  return (
    <ol className={`task-card-list ${className}`} aria-label={label}>
      {entries.map((entry) =>
        "group" in entry ? (
          <li key={entry.group.id} className="task-card-group">
            <h2>
              <TaskGroupLink group={entry.group} />
            </h2>
            <ol className="task-group-rows" aria-label={entry.group.name}>
              {entry.cards.map(({ task, feature }) => (
                <li key={task.id}>
                  <TaskGroupRow
                    state={task.displayState}
                    kind={task.kind}
                    appearances={feature?.taskLabelAppearances}
                    title={task.title}
                  />
                </li>
              ))}
            </ol>
          </li>
        ) : (
          <TaskCard key={entry.card.task.id} {...entry.card} />
        ),
      )}
    </ol>
  );
}
