import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getPromptTemplates, type PromptTemplateSettings } from "../api";
import type { PromptTemplateOverrides } from "../gen/prx/v1/prx_pb";

type PromptKind = "design" | "implementation" | "batch" | "batchDesign";
export type PromptValues = Record<PromptKind, string>;
type PromptModes = Record<PromptKind, boolean>;

const promptKinds: readonly PromptKind[] = [
  "design",
  "implementation",
  "batch",
  "batchDesign",
];

// batch 系は task 用とは別の語彙で検証する。どちらの batch テンプレートにも
// 展開元になる単一の task がないため。
function isBatchKind(kind: PromptKind): boolean {
  return kind === "batch" || kind === "batchDesign";
}

export interface PromptOverridesPanelProps {
  scope: "project" | "feature";
  overrides?: PromptTemplateOverrides | undefined;
  parentOverrides?: PromptTemplateOverrides | undefined;
  editable: boolean;
  onStateChange: (
    values: PromptValues,
    ready: boolean,
    invalid: boolean,
  ) => void;
}

// project と feature の編集面で同じ解決順と編集操作を見せる。上書きが空なら
// 親の実効値を read-only で表示し、切り替えた瞬間にその文面を下書きへコピーする。
export function PromptOverridesPanel({
  scope,
  overrides,
  parentOverrides,
  editable,
  onStateChange,
}: PromptOverridesPanelProps) {
  const { t } = useTranslation();
  const initialValues = useMemo(() => valuesOf(overrides), [overrides]);
  const [values, setValues] = useState<PromptValues>(initialValues);
  const [modes, setModes] = useState<PromptModes>(() => modesOf(overrides));
  const [remembered, setRemembered] = useState<PromptValues>(initialValues);
  const [rememberedKinds, setRememberedKinds] = useState<PromptModes>(() =>
    modesOf(overrides),
  );
  const { settings, loading, error } = usePromptMetadata(
    values,
    modes,
    onStateChange,
  );

  function changeValue(kind: PromptKind, value: string) {
    const next = { ...values, [kind]: value };
    setValues(next);
    setRemembered((current) => ({ ...current, [kind]: value }));
    setRememberedKinds((current) => ({ ...current, [kind]: true }));
    onStateChange(
      next,
      settings !== undefined,
      settings ? invalidValues(next, modes, settings) : false,
    );
  }

  function changeMode(kind: PromptKind, enabled: boolean) {
    const inherited = inheritedValue(kind, settings, parentOverrides);
    const nextValue = enabled
      ? rememberedKinds[kind]
        ? remembered[kind]
        : inherited.value
      : "";
    const nextValues = { ...values, [kind]: nextValue };
    setModes((current) => ({ ...current, [kind]: enabled }));
    setValues(nextValues);
    if (enabled) {
      setRememberedKinds((current) => ({ ...current, [kind]: true }));
      if (!rememberedKinds[kind]) {
        setRemembered((current) => ({ ...current, [kind]: nextValue }));
      }
    }
    const nextModes = { ...modes, [kind]: enabled };
    onStateChange(
      nextValues,
      settings !== undefined,
      settings ? invalidValues(nextValues, nextModes, settings) : false,
    );
  }

  if (loading) {
    return (
      <p className="settings-panel-state">{t("promptOverrides.loading")}</p>
    );
  }
  if (!settings) {
    return (
      <p className="form-error">
        {error?.message ?? t("promptOverrides.unavailable")}
      </p>
    );
  }

  return (
    <div className="prompt-overrides-panel">
      <p className="dialog-lead">{t(`promptOverrides.${scope}Description`)}</p>
      <div className="prompt-overrides-list">
        {promptKinds.map((kind) => (
          <PromptOverrideField
            key={kind}
            kind={kind}
            scope={scope}
            values={values}
            modes={modes}
            settings={settings}
            parentOverrides={parentOverrides}
            editable={editable}
            onChangeValue={changeValue}
            onChangeMode={changeMode}
          />
        ))}
      </div>
    </div>
  );
}

function usePromptMetadata(
  values: PromptValues,
  modes: PromptModes,
  onStateChange: PromptOverridesPanelProps["onStateChange"],
): {
  settings: PromptTemplateSettings | undefined;
  loading: boolean;
  error: Error | undefined;
} {
  const [settings, setSettings] = useState<PromptTemplateSettings>();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error>();

  useEffect(() => {
    let cancelled = false;
    void getPromptTemplates()
      .then((result) => {
        if (cancelled) return;
        setSettings(result);
        setLoading(false);
        onStateChange(values, true, invalidValues(values, modes, result));
      })
      .catch((cause: unknown) => {
        if (cancelled) return;
        setError(cause instanceof Error ? cause : new Error(String(cause)));
        setLoading(false);
        onStateChange(values, false, false);
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps -- ダイアログの初期値を固定し、取得途中の保存競合を避ける。
  }, []);
  return { settings, loading, error };
}

function PromptOverrideField({
  kind,
  scope,
  values,
  modes,
  settings,
  parentOverrides,
  editable,
  onChangeValue,
  onChangeMode,
}: {
  kind: PromptKind;
  scope: "project" | "feature";
  values: PromptValues;
  modes: PromptModes;
  settings: PromptTemplateSettings;
  parentOverrides?: PromptTemplateOverrides | undefined;
  editable: boolean;
  onChangeValue: (kind: PromptKind, value: string) => void;
  onChangeMode: (kind: PromptKind, enabled: boolean) => void;
}) {
  const { t } = useTranslation();
  const inherited = inheritedValue(kind, settings, parentOverrides);
  const enabled = modes[kind];
  const label = t(`promptOverrides.${kind}`);
  const validation = enabled
    ? validationError(kind, values[kind], settings)
    : undefined;
  const translateValidation = t as unknown as (
    key: string,
    options?: { placeholder?: string; required?: string },
  ) => string;
  const validationText = validation
    ? translateValidation(validation.key, validation.options)
    : undefined;

  return (
    <section className="prompt-override-field">
      <div className="prompt-override-head">
        <div>
          <h3>{label}</h3>
          <p className="prompt-override-source">
            {enabled
              ? t(`promptOverrides.${scope}Source`)
              : t("promptOverrides.inheritedFrom", {
                  source: t(`promptOverrides.${inherited.source}`),
                })}
          </p>
        </div>
        <label className="prompt-override-mode">
          <span>{t("promptOverrides.mode")}</span>
          <select
            aria-label={t("promptOverrides.modeFor", { label })}
            disabled={!editable}
            value={enabled ? "override" : "inherit"}
            onChange={(event) => {
              onChangeMode(kind, event.target.value === "override");
            }}
          >
            <option value="inherit">{t("promptOverrides.inherit")}</option>
            <option value="override">{t("promptOverrides.override")}</option>
          </select>
        </label>
      </div>
      <textarea
        aria-label={label}
        className="prompt-override-textarea"
        disabled={!editable || !enabled}
        readOnly={!enabled}
        rows={8}
        value={enabled ? values[kind] : inherited.value}
        onChange={(event) => {
          onChangeValue(kind, event.target.value);
        }}
      />
      <small>
        {isBatchKind(kind)
          ? t("promptOverrides.batchHint", {
              list: placeholderList(settings.batchSupportedPlaceholders),
              required: `{{${settings.batchRequiredPlaceholder}}}`,
            })
          : t("promptOverrides.taskHint", {
              list: placeholderList(settings.supportedPlaceholders),
              required: `{{${settings.requiredPlaceholder}}}`,
            })}
      </small>
      {validationText && <p className="form-error">{validationText}</p>}
    </section>
  );
}

function valuesOf(overrides?: PromptTemplateOverrides): PromptValues {
  return {
    design: overrides?.design ?? "",
    implementation: overrides?.implementation ?? "",
    batch: overrides?.batch ?? "",
    batchDesign: overrides?.batchDesign ?? "",
  };
}

function modesOf(overrides?: PromptTemplateOverrides): PromptModes {
  const values = valuesOf(overrides);
  return {
    design: values.design !== "",
    implementation: values.implementation !== "",
    batch: values.batch !== "",
    batchDesign: values.batchDesign !== "",
  };
}

function inheritedValue(
  kind: PromptKind,
  settings: PromptTemplateSettings | undefined,
  parentOverrides?: PromptTemplateOverrides,
): { value: string; source: "projectSource" | "globalSource" } {
  const parent = parentOverrides?.[kind] ?? "";
  if (parent !== "") return { value: parent, source: "projectSource" };
  return { value: settings?.[kind] ?? "", source: "globalSource" };
}

function placeholderList(names: string[]): string {
  return names.map((name) => `{{${name}}}`).join(", ");
}

function invalidValues(
  values: PromptValues,
  modes: PromptModes,
  settings: PromptTemplateSettings,
): boolean {
  return promptKinds.some(
    (kind) =>
      modes[kind] &&
      validationError(kind, values[kind], settings) !== undefined,
  );
}

interface ValidationError {
  key:
    | "promptOverrides.invalidEmpty"
    | "promptOverrides.invalidTooLong"
    | "promptOverrides.invalidPlaceholder"
    | "promptOverrides.invalidRequired";
  options?: { placeholder?: string; required?: string };
}

function validationError(
  kind: PromptKind,
  value: string,
  settings: PromptTemplateSettings,
): ValidationError | undefined {
  if (value.trim() === "") return { key: "promptOverrides.invalidEmpty" };
  if (new TextEncoder().encode(value).length > 8192)
    return { key: "promptOverrides.invalidTooLong" };
  const supported = new Set(
    isBatchKind(kind)
      ? settings.batchSupportedPlaceholders
      : settings.supportedPlaceholders,
  );
  const placeholders = value.match(/\{\{([^{}]*)\}\}/g) ?? [];
  for (const placeholder of placeholders) {
    const name = placeholder.slice(2, -2).trim();
    if (!supported.has(name))
      return {
        key: "promptOverrides.invalidPlaceholder",
        options: { placeholder },
      };
  }
  const required = isBatchKind(kind)
    ? settings.batchRequiredPlaceholder
    : settings.requiredPlaceholder;
  if (
    !placeholders.some(
      (placeholder) => placeholder.slice(2, -2).trim() === required,
    )
  )
    return {
      key: "promptOverrides.invalidRequired",
      options: { required: `{{${required}}}` },
    };
  return undefined;
}
