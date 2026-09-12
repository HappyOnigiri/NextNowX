import type { PromptTemplateOverrides } from "../gen/prx/v1/prx_pb";
import type { PromptValues } from "./PromptOverridesPanel";

export function promptValuesOf(value?: PromptTemplateOverrides): PromptValues {
  return {
    design: value?.design ?? "",
    implementation: value?.implementation ?? "",
    batch: value?.batch ?? "",
  };
}

export function promptValuesChanged(
  left: PromptValues,
  right: PromptValues,
): boolean {
  return (
    left.design !== right.design ||
    left.implementation !== right.implementation ||
    left.batch !== right.batch
  );
}

export function changedPromptValues(
  values: PromptValues,
  original: PromptValues,
): { design?: string; implementation?: string; batch?: string } | undefined {
  const result: {
    design?: string;
    implementation?: string;
    batch?: string;
  } = {};
  if (values.design !== original.design) result.design = values.design;
  if (values.implementation !== original.implementation)
    result.implementation = values.implementation;
  if (values.batch !== original.batch) result.batch = values.batch;
  return Object.keys(result).length > 0 ? result : undefined;
}
