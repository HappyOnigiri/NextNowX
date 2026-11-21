import { useNavigate, useSearch } from "@tanstack/react-router";
import { Search } from "lucide-react";
import { useState, type SyntheticEvent } from "react";
import { useTranslation } from "react-i18next";
import type { Project, TaskGroup } from "../gen/nnx/v1/nnx_pb";
import { useSnapshot } from "../hooks";
import { formatError } from "../i18n/domain";
import {
  filterTaskSearchResults,
  parseTaskSearch,
  type TaskSearchParseError,
  type TaskSearchParseResult,
  type TaskSearchResult,
} from "../task-search";
import { StateMessage } from "./Dashboard";
import { IconButton } from "./IconButton";
import { TaskCardList } from "./TaskCardList";

export function TaskSearch() {
  const { t } = useTranslation();
  const { q } = useSearch({ from: "/tasks" });
  const { data, isPending, error, refetch } = useSnapshot();

  if (isPending)
    return (
      <StateMessage
        title={t("tasks.loadingTitle")}
        detail={t("tasks.loadingDetail")}
      />
    );
  if (error)
    return (
      <StateMessage
        title={t("tasks.errorTitle")}
        detail={formatError(error, t)}
        action={() => void refetch()}
      />
    );

  const parsed = parseTaskSearch(q);
  const results = parsed.error ? [] : filterTaskSearchResults(data, parsed);
  return (
    <TaskSearchContent
      key={q}
      query={q}
      parsed={parsed}
      results={results}
      projects={data.projects}
      groups={data.taskGroups}
    />
  );
}

function TaskSearchContent({
  query,
  parsed,
  results,
  projects,
  groups,
}: {
  query: string;
  parsed: TaskSearchParseResult;
  results: TaskSearchResult[];
  projects: Project[];
  groups: TaskGroup[];
}) {
  const { t } = useTranslation();
  return (
    <div className="dashboard task-search-page">
      <header className="page-head">
        <div>
          <h1>{t("tasks.title")}</h1>
        </div>
        <div className="task-search-count" aria-live="polite">
          <span>{parsed.error ? "—" : results.length}</span>
          <small>{t("tasks.resultCountLabel")}</small>
        </div>
      </header>
      <TaskSearchForm query={query} />
      <TaskSearchResults
        parsed={parsed}
        results={results}
        projects={projects}
        groups={groups}
      />
    </div>
  );
}

function TaskSearchForm({ query }: { query: string }) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [input, setInput] = useState(query);
  const submit = (event: SyntheticEvent<HTMLFormElement>) => {
    event.preventDefault();
    void navigate({
      to: "/tasks",
      search: { q: input.trim() },
    });
  };
  return (
    <form className="task-search-form" onSubmit={submit} role="search">
      <label htmlFor="task-search-input">{t("tasks.searchLabel")}</label>
      <div>
        <input
          id="task-search-input"
          name="q"
          value={input}
          onChange={(event) => {
            setInput(event.target.value);
          }}
          placeholder={t("tasks.searchPlaceholder")}
          autoComplete="off"
        />
        <IconButton
          icon={Search}
          label={t("tasks.searchSubmit")}
          variant="primary"
          type="submit"
        />
      </div>
      <p>{t("tasks.searchHint")}</p>
    </form>
  );
}

function TaskSearchResults({
  parsed,
  results,
  projects,
  groups,
}: {
  parsed: TaskSearchParseResult;
  results: TaskSearchResult[];
  projects: Project[];
  groups: TaskGroup[];
}) {
  const { t } = useTranslation();
  if (parsed.error) return <SearchError error={parsed.error} />;
  if (results.length === 0)
    return (
      <div className="empty compact task-search-empty">
        <h2>{t("tasks.emptyTitle")}</h2>
        <p>{t("tasks.emptyDetail")}</p>
      </div>
    );
  return (
    <TaskCardList
      className="task-results"
      label={t("tasks.listLabel")}
      groups={groups}
      cards={results.map(({ task, feature, pullRequest }) => ({
        task,
        feature,
        pullRequest,
        project: projects.find((item) => item.id === feature.projectId),
      }))}
    />
  );
}

function SearchError({ error }: { error: TaskSearchParseError }) {
  const { t } = useTranslation();
  const detail =
    error.type === "unterminated-quote"
      ? t("tasks.unterminatedQuote")
      : t("tasks.invalidQualifier", { key: error.key, value: error.value });
  return (
    <div className="task-search-error" role="alert">
      <strong>{t("tasks.invalidSearchTitle")}</strong>
      <p>{detail}</p>
    </div>
  );
}
