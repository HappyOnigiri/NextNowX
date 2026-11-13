import { ClipboardList, CodeXml } from "lucide-react";
import { useTranslation } from "react-i18next";
import { TaskKind } from "../gen/nnx/v1/nnx_pb";

export function TaskKindLabel({ kind }: { kind: TaskKind | undefined }) {
  const { t } = useTranslation();
  const work = kind === TaskKind.WORK;
  const Icon = work ? ClipboardList : CodeXml;
  return (
    <span className="task-kind-label">
      <Icon aria-hidden="true" focusable="false" size={13} strokeWidth={1.75} />
      {t(work ? "taskKind.work" : "taskKind.development")}
    </span>
  );
}
