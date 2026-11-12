import { useState } from "react";
import { useTranslation } from "react-i18next";
import { TaskKind } from "../gen/nnx/v1/nnx_pb";
import { ConfirmationDialog } from "./ConfirmationDialog";

export function TaskKindSelect({
  value,
  defaultValue,
  onChange,
  hasPullRequest = false,
}: {
  value?: TaskKind;
  defaultValue?: TaskKind;
  onChange?: (kind: TaskKind) => void;
  hasPullRequest?: boolean;
}) {
  const { t } = useTranslation();
  const [showWarning, setShowWarning] = useState(false);
  function closeWarning() {
    setShowWarning(false);
  }
  return (
    <>
      <label>
        {t("common.kind")}
        <select
          name="kind"
          aria-label={t("common.kind")}
          value={value}
          defaultValue={defaultValue}
          onChange={(event) => {
            const kind: TaskKind = Number(event.target.value);
            if (kind === TaskKind.WORK && hasPullRequest) {
              setShowWarning(true);
              return;
            }
            onChange?.(kind);
          }}
        >
          <option value={TaskKind.DEVELOPMENT}>
            {t("taskKind.development")}
          </option>
          <option value={TaskKind.WORK}>{t("taskKind.work")}</option>
        </select>
      </label>
      {showWarning && (
        <ConfirmationDialog
          title={t("taskKind.warningTitle")}
          description={t("taskKind.warningDescription")}
          confirmLabel={t("common.close")}
          pending={false}
          error={null}
          onCancel={closeWarning}
          onConfirm={closeWarning}
        />
      )}
    </>
  );
}
