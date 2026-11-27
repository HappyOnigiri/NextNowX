import { Check, Copy } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { IconButton } from "./IconButton";

type CopyStatus = "copied" | "failed";

function useCopyText(value: string) {
  const [status, setStatus] = useState<CopyStatus>();

  async function copy() {
    try {
      await navigator.clipboard.writeText(value);
      setStatus("copied");
    } catch {
      setStatus("failed");
    }
    window.setTimeout(() => {
      setStatus(undefined);
    }, 1600);
  }

  return { status, copy };
}

export function CopyTextButton({
  label,
  value,
}: {
  label: string;
  value: string;
}) {
  const { t } = useTranslation();
  const { status, copy } = useCopyText(value);
  const isCopied = status === "copied";

  return (
    <>
      <IconButton
        icon={isCopied ? Check : Copy}
        label={label}
        size="compact"
        iconOnly
        aria-label={isCopied ? t("common.copied") : label}
        onClick={() => void copy()}
      />
      <span
        className="copyable-identifier-status is-visually-hidden"
        aria-live="polite"
      >
        {status === "failed" && t("common.copyFailed")}
      </span>
    </>
  );
}

export function CopyableIdentifier({
  label,
  value,
  valueOnly = false,
}: {
  label: string;
  value: string;
  valueOnly?: boolean;
}) {
  const { t } = useTranslation();
  const { status, copy } = useCopyText(value);

  const copyLabel = t("common.copyIdentifier", { label });
  const isCopied = status === "copied";
  const Icon = isCopied ? Check : Copy;

  if (valueOnly) {
    return (
      <span className="copyable-identifier is-value-only">
        <button
          type="button"
          className={`copyable-identifier-value-button${isCopied ? " is-copied" : ""}`}
          aria-label={isCopied ? t("common.copied") : copyLabel}
          title={copyLabel}
          onClick={() => void copy()}
        >
          <code className="copyable-identifier-value" title={value}>
            {value}
          </code>
        </button>
        <span
          className="copyable-identifier-status is-visually-hidden"
          aria-live="polite"
        >
          {status === "failed" && t("common.copyFailed")}
        </span>
      </span>
    );
  }

  return (
    <span className="copyable-identifier">
      <span className="copyable-identifier-label">{label}</span>
      <code className="copyable-identifier-value" title={value}>
        {value}
      </code>
      <IconButton
        icon={Icon}
        label={copyLabel}
        variant="quiet"
        size="compact"
        iconOnly
        className={`copyable-identifier-button${isCopied ? " is-copied" : ""}`}
        aria-label={isCopied ? t("common.copied") : copyLabel}
        title={copyLabel}
        onClick={() => void copy()}
        iconProps={{ "data-icon": isCopied ? "check" : "copy" }}
      />
      <span className="copyable-identifier-status" aria-live="polite">
        {status === "failed" && t("common.copyFailed")}
      </span>
    </span>
  );
}
