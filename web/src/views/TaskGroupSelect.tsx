import { useTranslation } from "react-i18next";
import type { TaskGroup } from "../gen/nnx/v1/nnx_pb";

export function TaskGroupSelect({
  groups,
  value,
  defaultValue,
  onChange,
}: {
  groups: TaskGroup[];
  value?: string;
  defaultValue?: string | undefined;
  onChange?: (value: string) => void;
}) {
  const { t } = useTranslation();
  const unavailable = value && !groups.some((group) => group.id === value);
  return (
    <label>
      {t("taskGroups.field")}
      <select
        name="group"
        value={value}
        defaultValue={defaultValue}
        onChange={(event) => onChange?.(event.target.value)}
      >
        <option value="">{t("taskGroups.ungrouped")}</option>
        {unavailable && (
          <option value={value} disabled>
            {t("taskGroups.unavailable")}
          </option>
        )}
        {groups.map((group) => (
          <option key={group.id} value={group.id}>
            {group.name}
          </option>
        ))}
      </select>
    </label>
  );
}
