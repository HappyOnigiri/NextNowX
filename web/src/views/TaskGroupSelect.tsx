import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { TaskGroup } from "../gen/nnx/v1/nnx_pb";
import { TaskGroupNameDialog } from "./TaskGroupDialogs";

const createOption = "new";

export function TaskGroupSelect({
  groups,
  value,
  defaultValue,
  onChange,
  createInFeature,
}: {
  groups: TaskGroup[];
  value?: string;
  createInFeature?: string;
  defaultValue?: string | undefined;
  onChange?: (value: string) => void;
}) {
  const { t } = useTranslation();
  const [creating, setCreating] = useState(false);
  const unavailable = value && !groups.some((group) => group.id === value);
  return (
    <>
      <label>
        {t("taskGroups.field")}
        <select
          name="group"
          value={value}
          defaultValue={defaultValue}
          onChange={(event) => {
            const selected = event.target.value;
            if (selected === createOption) setCreating(true);
            else onChange?.(selected);
          }}
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
          {createInFeature && (
            <option value={createOption}>{t("taskGroups.addNew")}</option>
          )}
        </select>
      </label>
      {createInFeature && creating && (
        <TaskGroupNameDialog
          featureId={createInFeature}
          groups={groups}
          onSaved={(group) => {
            onChange?.(group.id);
            setCreating(false);
          }}
          onClose={() => {
            setCreating(false);
          }}
        />
      )}
    </>
  );
}
