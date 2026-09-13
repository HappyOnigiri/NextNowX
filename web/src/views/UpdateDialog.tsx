import { useMutation } from "@tanstack/react-query";
import { BellOff, Download, X } from "lucide-react";
import { useEffect, useId, useRef } from "react";
import { useTranslation } from "react-i18next";
import ReactMarkdown from "react-markdown";
import { applyUpdate, skipUpdateVersion } from "../api";
import type { UpdateRelease, UpdateStatus } from "../gen/prx/v1/prx_pb";
import { useUpdateStatusInvalidation } from "../hooks";
import { formatError } from "../i18n/domain";
import { IconButton } from "./IconButton";
import { useCloseOnEscape } from "./useCloseOnEscape";

interface UpdateDialogProps {
  status: UpdateStatus;
  onClose: () => void;
}

export function UpdateDialog({ status, onClose }: UpdateDialogProps) {
  const { t } = useTranslation();
  const titleId = useId();
  const descriptionId = useId();
  const dialogRef = useRef<HTMLElement>(null);
  const confirmRef = useRef<HTMLButtonElement>(null);
  const invalidate = useUpdateStatusInvalidation();

  const install = useMutation({
    mutationFn: () => applyUpdate(status.latestVersion),
    onSuccess: () => invalidate(),
  });
  const skip = useMutation({
    mutationFn: () => skipUpdateVersion(status.latestVersion),
    onSuccess: async () => {
      await invalidate();
      onClose();
    },
  });
  // 書き込みの完了を待っている間は閉じさせず、背後のモーダルにも Escape を渡さない。
  const pending = install.isPending || skip.isPending;

  useCloseOnEscape(onClose, !pending);

  useEffect(() => {
    const origin = document.activeElement;
    confirmRef.current?.focus();
    return () => {
      window.setTimeout(() => {
        if (origin instanceof HTMLElement && origin.isConnected) origin.focus();
      });
    };
  }, []);

  const error = install.error ?? skip.error;
  const applied = install.data;

  return (
    <div
      className="scrim update-scrim"
      role="presentation"
      onKeyDown={(event) => {
        trapTab(event, dialogRef.current);
      }}
    >
      <section
        ref={dialogRef}
        className="dialog update-dialog"
        role="dialog"
        aria-modal="true"
        aria-busy={pending || undefined}
        aria-labelledby={titleId}
        aria-describedby={descriptionId}
      >
        <header>
          <h2 id={titleId}>
            {t("update.dialogTitle", { version: status.latestVersion })}
          </h2>
          <p id={descriptionId}>
            {t("update.dialogDescription", {
              current: status.currentVersion,
              latest: status.latestVersion,
            })}
          </p>
        </header>
        <div className="update-notes">
          {status.releases.map((release) => (
            <ReleaseNotes key={release.version} release={release} />
          ))}
        </div>
        {applied && (
          <p className="update-result" role="status">
            {applied.restartRequired
              ? t("update.restartRequired", { version: applied.version })
              : t("update.installed", { version: applied.version })}
          </p>
        )}
        {error && (
          <p className="form-error" role="alert">
            {formatError(error, t)}
          </p>
        )}
        <footer>
          <IconButton
            icon={X}
            label={t("common.cancel")}
            variant="secondary"
            disabled={pending}
            onClick={onClose}
          />
          <IconButton
            icon={BellOff}
            label={t("update.skip")}
            variant="secondary"
            busy={skip.isPending}
            disabled={pending}
            onClick={() => {
              skip.mutate();
            }}
          />
          <IconButton
            ref={confirmRef}
            icon={Download}
            label={t("update.install")}
            variant="primary"
            busy={install.isPending}
            disabled={pending || applied !== undefined}
            onClick={() => {
              install.mutate();
            }}
          />
        </footer>
      </section>
    </div>
  );
}

function ReleaseNotes({ release }: { release: UpdateRelease }) {
  const { t } = useTranslation();
  return (
    <article className="update-release">
      <h3>
        {release.url ? (
          <a href={release.url} target="_blank" rel="noreferrer">
            {release.version}
          </a>
        ) : (
          release.version
        )}
        {release.publishedAt && <span>{formatDate(release.publishedAt)}</span>}
      </h3>
      <div className="markdown-content">
        {release.body ? (
          <ReactMarkdown
            components={{
              a: ({ children, ...props }) => (
                <a {...props} target="_blank" rel="noreferrer">
                  {children}
                </a>
              ),
            }}
          >
            {release.body}
          </ReactMarkdown>
        ) : (
          <p>{t("update.emptyNotes")}</p>
        )}
      </div>
    </article>
  );
}

// 公開日はリリースの並びを読むためだけに出すので、時刻までは見せない。
function formatDate(value: string) {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return value;
  return parsed.toISOString().slice(0, 10);
}

function trapTab(
  event: React.KeyboardEvent<HTMLDivElement>,
  dialog: HTMLElement | null,
) {
  if (event.key !== "Tab") return;
  const focusable = Array.from(
    dialog?.querySelectorAll<HTMLElement>("button:not(:disabled), a[href]") ??
      [],
  );
  if (focusable.length === 0) {
    event.preventDefault();
    return;
  }
  const first = focusable[0];
  const last = focusable.at(-1);
  if (!first || !last) return;
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault();
    last.focus();
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault();
    first.focus();
  }
}
