import type { PromptTemplateOverrides } from "../gen/prx/v1/prx_pb";
import {
  PromptOverridesPanel,
  type PromptValues,
} from "./PromptOverridesPanel";
import { TabPanel } from "./TabList";

// project と feature の編集ダイアログが共有する prompt タブの結線。
export function PromptOverridesTabPanel({
  active,
  idPrefix,
  promptsMounted,
  scope,
  overrides,
  parentOverrides,
  editable,
  onStateChange,
}: {
  active: boolean;
  idPrefix: string;
  promptsMounted: boolean;
  scope: "project" | "feature";
  overrides?: PromptTemplateOverrides | undefined;
  parentOverrides?: PromptTemplateOverrides | undefined;
  editable: boolean;
  onStateChange: (
    values: PromptValues,
    ready: boolean,
    invalid: boolean,
  ) => void;
}) {
  return (
    <TabPanel
      active={active}
      className="entity-edit-tab-panel"
      idPrefix={idPrefix}
      tab="prompts"
    >
      {promptsMounted && (
        <PromptOverridesPanel
          scope={scope}
          overrides={overrides}
          parentOverrides={parentOverrides}
          editable={editable}
          onStateChange={onStateChange}
        />
      )}
    </TabPanel>
  );
}
