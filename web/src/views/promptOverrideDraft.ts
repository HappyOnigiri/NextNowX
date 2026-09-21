import type { PromptTemplateOverrides } from "../gen/prx/v1/prx_pb";
import type { PromptValues } from "./PromptOverridesPanel";

export function promptValuesOf(value?: PromptTemplateOverrides): PromptValues {
  return {
    design: value?.design ?? "",
    implementation: value?.implementation ?? "",
    batch: value?.batch ?? "",
    batchDesign: value?.batchDesign ?? "",
  };
}

export function promptValuesChanged(
  left: PromptValues,
  right: PromptValues,
): boolean {
  return (
    left.design !== right.design ||
    left.implementation !== right.implementation ||
    left.batch !== right.batch ||
    left.batchDesign !== right.batchDesign
  );
}

export function changedPromptValues(
  values: PromptValues,
  original: PromptValues,
):
  | {
      design?: string;
      implementation?: string;
      batch?: string;
      batchDesign?: string;
    }
  | undefined {
  const result: {
    design?: string;
    implementation?: string;
    batch?: string;
    batchDesign?: string;
  } = {};
  if (values.design !== original.design) result.design = values.design;
  if (values.implementation !== original.implementation)
    result.implementation = values.implementation;
  if (values.batch !== original.batch) result.batch = values.batch;
  if (values.batchDesign !== original.batchDesign)
    result.batchDesign = values.batchDesign;
  return Object.keys(result).length > 0 ? result : undefined;
}
