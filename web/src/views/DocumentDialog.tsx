import { useMutation } from "@tanstack/react-query";
import { FileUp, Link, Plus, Text, X } from "lucide-react";
import {
  useEffect,
  useId,
  useRef,
  useState,
  type KeyboardEvent,
  type ReactNode,
  type SyntheticEvent,
} from "react";
import { useTranslation } from "react-i18next";
import { mutations, selectLocalFile } from "../api";
import type { DocumentParent } from "../document-parent";
import { DocumentKind, type Document } from "../gen/prx/v1/prx_pb";
import { useDomainMutation } from "../hooks";
import { IconButton } from "./IconButton";
import { MutationError } from "./MutationError";
import { TabPanel as SharedTabPanel, TabList } from "./TabList";
import { DiscardChangesDialog, SaveButton } from "./UnsavedChanges";
import { useCloseOnEscape } from "./useCloseOnEscape";

const documentTabs = [
  { kind: DocumentKind.URL, key: "url", icon: Link },
  { kind: DocumentKind.LOCAL_FILE, key: "localFile", icon: FileUp },
  { kind: DocumentKind.MARKDOWN, key: "markdown", icon: Text },
] as const;

function sourceCase(kind: DocumentKind) {
  if (kind === DocumentKind.URL) return "url" as const;
  if (kind === DocumentKind.LOCAL_FILE) return "localFile" as const;
  return "markdown" as const;
}

// 追加と編集は同じフォームを使う。入口ごとにダイアログを分けると見た目が
// ずれるので、差分は mode と初期値だけに閉じ込める。
export type DocumentDialogProps = {
  trigger: HTMLElement | null;
  onClose: () => void;
} & (
  | ({ mode: "add" } & DocumentParent)
  | { mode: "edit"; document: Document; content: string }
);

type SourceValues = Partial<Record<DocumentKind, string>>;

interface DialogInitial {
  kind: DocumentKind;
  title: string;
  values: SourceValues;
  implementationPlan: boolean;
  taskDocument: boolean;
}

function initialState(props: DocumentDialogProps): DialogInitial {
  const values: SourceValues = {
    [DocumentKind.URL]: "",
    [DocumentKind.LOCAL_FILE]: "",
    [DocumentKind.MARKDOWN]: "",
  };
  if (props.mode === "add")
    return {
      kind: DocumentKind.URL,
      title: "",
      values,
      implementationPlan: false,
      taskDocument: props.taskId !== undefined,
    };
  const { document } = props;
  values[document.kind] =
    document.kind === DocumentKind.MARKDOWN ? props.content : document.locator;
  return {
    kind: document.kind,
    title: document.title,
    values,
    implementationPlan: document.isImplementationPlan,
    taskDocument: document.taskId !== "",
  };
}

function useDialogState(props: DocumentDialogProps) {
  const [initial] = useState(() => initialState(props));
  const addDocument = useDomainMutation(mutations.addDocument);
  const updateDocument = useDomainMutation(mutations.updateDocument);
  const filePicker = useMutation({ mutationFn: selectLocalFile });
  const [kind, setKind] = useState<DocumentKind>(initial.kind);
  const [title, setTitle] = useState(initial.title);
  const [values, setValues] = useState<SourceValues>(initial.values);
  const [implementationPlan, setImplementationPlan] = useState(
    initial.implementationPlan,
  );
  const [pickerCanceled, setPickerCanceled] = useState(false);
  const titleId = useId();
  const idPrefix = useId();
  const saving =
    props.mode === "add" ? addDocument.isPending : updateDocument.isPending;
  const busy = saving || filePicker.isPending;
  // 別の種別のタブへ打ち込んだ内容は送信されないので、dirty は送る値だけで見る。
  const dirty =
    title !== initial.title ||
    kind !== initial.kind ||
    (values[kind] ?? "") !== (initial.values[kind] ?? "") ||
    implementationPlan !== initial.implementationPlan;

  async function chooseFile() {
    setPickerCanceled(false);
    try {
      const result = await filePicker.mutateAsync();
      if (result.canceled) {
        setPickerCanceled(true);
        return;
      }
      setValues((current) => ({
        ...current,
        [DocumentKind.LOCAL_FILE]: result.path,
      }));
    } catch {
      return;
    }
  }

  async function submit(event: SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    try {
      if (props.mode === "add") await addDocument.mutateAsync(addInput(props));
      else await updateDocument.mutateAsync(updateInput(props));
    } catch {
      return;
    }
    props.onClose();
  }

  function addInput(added: Extract<DocumentDialogProps, { mode: "add" }>) {
    const input: Parameters<typeof mutations.addDocument>[0] = {
      title,
      kind,
      value: values[kind] ?? "",
    };
    if (added.projectId !== undefined) input.projectId = added.projectId;
    if (added.featureId !== undefined) input.featureId = added.featureId;
    if (added.taskId !== undefined) {
      input.taskId = added.taskId;
      input.isImplementationPlan = implementationPlan;
    }
    return input;
  }

  function updateInput(edited: Extract<DocumentDialogProps, { mode: "edit" }>) {
    const input: Parameters<typeof mutations.updateDocument>[0] = {
      id: edited.document.id,
      title,
      source: { case: sourceCase(kind), value: values[kind] ?? "" },
    };
    // task 以外の資料に is_implementation_plan を送るとサーバーが拒否する。
    if (initial.taskDocument) input.isImplementationPlan = implementationPlan;
    return input;
  }

  return {
    ...props,
    busy,
    chooseFile,
    dirty,
    error: props.mode === "add" ? addDocument.error : updateDocument.error,
    filePicker,
    idPrefix,
    implementationPlan,
    kind,
    pickerCanceled,
    saving,
    setImplementationPlan,
    setKind,
    setPickerCanceled,
    setTitle,
    setValues,
    submit,
    taskDocument: initial.taskDocument,
    title,
    titleId,
    values,
  };
}

type DialogState = ReturnType<typeof useDialogState>;

export function DocumentDialog(props: DocumentDialogProps) {
  const { t } = useTranslation();
  const state = useDialogState(props);
  const dialogRef = useRef<HTMLFormElement>(null);
  const [confirmDiscard, setConfirmDiscard] = useState(false);

  // 編集は下書きを持つので、閉じる経路をすべて requestClose に通す。
  function requestClose() {
    if (props.mode === "edit" && state.dirty) setConfirmDiscard(true);
    else props.onClose();
  }

  useCloseOnEscape(requestClose, !state.busy);

  useEffect(() => {
    return () => {
      window.setTimeout(() => props.trigger?.focus());
    };
  }, [props.trigger]);

  return (
    <>
      <div
        className="scrim document-dialog-scrim"
        role="presentation"
        aria-hidden={confirmDiscard ? true : undefined}
        inert={confirmDiscard}
        onKeyDown={(event) => {
          trapFocus(event, dialogRef.current);
        }}
      >
        <form
          ref={dialogRef}
          className="dialog document-dialog"
          role="dialog"
          aria-modal="true"
          aria-labelledby={state.titleId}
          onSubmit={state.submit}
        >
          <DialogHeader state={state} onClose={requestClose} />
          <label className="document-dialog-title-field">
            {t("documentDialog.titleLabel")}
            <input
              name="title"
              placeholder={t("documentDialog.titlePlaceholder")}
              value={state.title}
              onChange={(event) => {
                state.setTitle(event.currentTarget.value);
              }}
            />
          </label>
          <DocumentSourceTabs state={state} />
          {state.taskDocument && (
            <label className="document-plan-toggle">
              <input
                checked={state.implementationPlan}
                type="checkbox"
                onChange={(event) => {
                  state.setImplementationPlan(event.currentTarget.checked);
                }}
              />
              {t("documentDialog.implementationPlan")}
            </label>
          )}
          <MutationError error={state.error} />
          <DialogFooter state={state} onClose={requestClose} />
        </form>
      </div>
      {confirmDiscard && (
        <DiscardChangesDialog
          onCancel={() => {
            setConfirmDiscard(false);
          }}
          onConfirm={props.onClose}
        />
      )}
    </>
  );
}

function trapFocus(
  event: KeyboardEvent<HTMLDivElement>,
  dialog: HTMLFormElement | null,
) {
  if (event.key !== "Tab") return;
  const focusable = Array.from(
    dialog?.querySelectorAll<HTMLElement>(
      "button:not(:disabled), input:not(:disabled), textarea:not(:disabled), a[href]",
    ) ?? [],
  ).filter((element) => !element.closest("[hidden]"));
  const first = focusable[0];
  const last = focusable.at(-1);
  if (!first || !last) {
    event.preventDefault();
    return;
  }
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault();
    last.focus();
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault();
    first.focus();
  }
}

// 見出しにドキュメントの所属先を示すことで、追加と編集それぞれの入口に
// 専用ダイアログを用意せずに済ませる。
function documentDialogTitleKey(state: DialogState) {
  if (state.mode === "edit") {
    if (state.document.taskId !== "") return "documentDialog.editTaskTitle";
    if (state.document.featureId !== "")
      return "documentDialog.editFeatureTitle";
    return "documentDialog.editProjectTitle";
  }
  if (state.projectId !== undefined) return "documentDialog.projectTitle";
  if (state.featureId !== undefined) return "documentDialog.featureTitle";
  return "documentDialog.taskTitle";
}

function DialogHeader({
  state,
  onClose,
}: {
  state: DialogState;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  return (
    <header className="document-dialog-head">
      <div>
        <h2 id={state.titleId}>{t(documentDialogTitleKey(state))}</h2>
      </div>
      <IconButton
        icon={X}
        label={t("common.close")}
        variant="secondary"
        iconOnly
        disabled={state.busy}
        onClick={onClose}
      />
    </header>
  );
}

function DocumentSourceTabs({ state }: { state: DialogState }) {
  const { t } = useTranslation();
  const active = documentTabs.find((tab) => tab.kind === state.kind);
  return (
    <>
      <TabList
        tabs={documentTabs.map((tab) => ({
          id: tab.key,
          label: t(`documentDialog.tabs.${tab.key}`),
          icon: <tab.icon aria-hidden="true" focusable="false" size={16} />,
        }))}
        active={active?.key ?? documentTabs[0].key}
        onSelect={(key) => {
          const tab = documentTabs.find((entry) => entry.key === key);
          if (tab) state.setKind(tab.kind);
        }}
        idPrefix={state.idPrefix}
        className="document-tabs"
        tabClassName="document-tab"
        focusOnMount
      />
      <URLPanel state={state} />
      <LocalFilePanel state={state} />
      <MarkdownPanel state={state} />
    </>
  );
}

function TabPanel({
  children,
  state,
  tab,
}: {
  children: ReactNode;
  state: DialogState;
  tab: (typeof documentTabs)[number];
}) {
  return (
    <SharedTabPanel
      active={state.kind === tab.kind}
      className={`document-tab-panel document-tab-panel-${tab.key}`}
      idPrefix={state.idPrefix}
      tab={tab.key}
    >
      {children}
    </SharedTabPanel>
  );
}

function setSourceValue(state: DialogState, kind: DocumentKind, value: string) {
  state.setValues((current) => ({ ...current, [kind]: value }));
}

function URLPanel({ state }: { state: DialogState }) {
  const { t } = useTranslation();
  const tab = documentTabs[0];
  return (
    <TabPanel state={state} tab={tab}>
      <label>
        {t("documentDialog.urlLabel")}
        <input
          disabled={state.kind !== tab.kind}
          required
          type="url"
          placeholder={t("documentDialog.urlPlaceholder")}
          value={state.values[tab.kind] ?? ""}
          onChange={(event) => {
            setSourceValue(state, tab.kind, event.currentTarget.value);
          }}
        />
      </label>
    </TabPanel>
  );
}

function LocalFilePanel({ state }: { state: DialogState }) {
  const { t } = useTranslation();
  const tab = documentTabs[1];
  return (
    <TabPanel state={state} tab={tab}>
      <label>
        {t("documentDialog.pathLabel")}
        <input
          disabled={state.kind !== tab.kind}
          required
          type="text"
          placeholder={t("documentDialog.pathPlaceholder")}
          value={state.values[tab.kind] ?? ""}
          onChange={(event) => {
            state.setPickerCanceled(false);
            setSourceValue(state, tab.kind, event.currentTarget.value);
          }}
        />
      </label>
      <div className="document-file-actions">
        <IconButton
          icon={FileUp}
          label={t(
            state.filePicker.isPending
              ? "documentDialog.choosingFile"
              : "documentDialog.chooseFile",
          )}
          variant="secondary"
          disabled={state.busy}
          onClick={() => void state.chooseFile()}
        />
        {state.pickerCanceled && (
          <span className="document-picker-status" role="status">
            {t("documentDialog.chooseCanceled")}
          </span>
        )}
      </div>
      <MutationError error={state.filePicker.error} />
    </TabPanel>
  );
}

function MarkdownPanel({ state }: { state: DialogState }) {
  const { t } = useTranslation();
  const tab = documentTabs[2];
  return (
    <TabPanel state={state} tab={tab}>
      <label className="document-markdown-field">
        {t("documentDialog.markdownLabel")}
        <textarea
          disabled={state.kind !== tab.kind}
          required
          placeholder={t("documentDialog.markdownPlaceholder")}
          value={state.values[tab.kind] ?? ""}
          onChange={(event) => {
            setSourceValue(state, tab.kind, event.currentTarget.value);
          }}
        />
      </label>
    </TabPanel>
  );
}

function DialogFooter({
  state,
  onClose,
}: {
  state: DialogState;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  return (
    <footer>
      <IconButton
        icon={X}
        label={t("common.cancel")}
        variant="secondary"
        disabled={state.busy}
        onClick={onClose}
      />
      {state.mode === "edit" ? (
        <SaveButton type="submit" dirty={state.dirty} pending={state.busy} />
      ) : (
        <IconButton
          icon={Plus}
          label={t(
            state.saving
              ? "documentDialog.submitting"
              : "documentDialog.submit",
          )}
          variant="primary"
          type="submit"
          disabled={state.busy}
        />
      )}
    </footer>
  );
}
