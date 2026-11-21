import type { TaskGroup } from "../gen/nnx/v1/nnx_pb";
import { TaskCard, type TaskCardProps } from "./TaskCard";

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
            <h2>{entry.group.name}</h2>
            <ol className="task-card-list" aria-label={entry.group.name}>
              {entry.cards.map((card) => (
                <TaskCard key={card.task.id} {...card} />
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
