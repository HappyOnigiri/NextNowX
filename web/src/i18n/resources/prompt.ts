export const prompt = {
  en: {
    promptSettings: {
      description:
        "Templates used for the prompt copied from a task. The CLI prints the same text.",
      loading: "Loading prompt templates…",
      design: "Design prompt",
      designHint: "Used while the task has no implementation plan.",
      implementation: "Implementation prompt",
      implementationHint: "Used once the task has an implementation plan.",
      placeholders:
        "Available placeholders: {{list}}. {{required}} is required.",
      batch: "Batch implementation prompt",
      batchHint:
        "Used for the tasks selected on a feature and copied as one prompt.",
      batchDesign: "Batch design prompt",
      batchDesignHint:
        "Used for the tasks selected on a feature and copied as one design prompt.",
      batchPlaceholders:
        "Available placeholders: {{list}}. {{required}} is required.",
      restoreDefaults: "Restore built-in templates",
    },
    promptOverrides: {
      projectDescription:
        "Override the templates for every feature in this project. Leave a kind inherited to follow the global Settings › Prompts value.",
      featureDescription:
        "Override the templates for this feature only. Inherited values follow the project override, then the global Settings › Prompts value.",
      loading: "Loading prompt metadata…",
      unavailable:
        "Prompt metadata is unavailable; prompt changes will not be saved.",
      design: "Design prompt",
      implementation: "Implementation prompt",
      batch: "Batch implementation prompt",
      batchDesign: "Batch design prompt",
      mode: "Source",
      modeFor: "Source for {{label}}",
      inherit: "Inherit",
      override: "Override here",
      projectSource: "Project override",
      featureSource: "Feature override",
      globalSource: "Global settings",
      inheritedFrom: "Inherited from {{source}}",
      taskHint: "Available placeholders: {{list}}. {{required}} is required.",
      batchHint: "Available placeholders: {{list}}. {{required}} is required.",
      invalidEmpty: "An override cannot be empty.",
      invalidTooLong: "Overrides are limited to 8192 bytes.",
      invalidPlaceholder: "Unsupported placeholder {{placeholder}}.",
      invalidRequired: "The template must include {{required}}.",
    },
    batchPrompt: {
      open: "Copy batch prompt",
      title: "Copy batch prompt",
      tabsLabel: "Batch prompt kind",
      tab: { design: "Design", implementation: "Implementation" },
      designDescription:
        "One prompt covering the tasks you select. It names each task so the agent hands it to a SubAgent that fetches the design prompt from Next Now X and registers a plan. A dependent task is designed after the work it waits for. Edit global wording in Settings › Prompts, or override it from a project or feature edit dialog.",
      implementationDescription:
        "One prompt covering the tasks you select. It names each task so the agent fetches its instructions from Next Now X. A dependent task is implemented after the work it waits for, with its pull request stacked on that work. Edit global wording in Settings › Prompts, or override it from a project or feature edit dialog.",
      designEmpty: "No task in this feature is waiting to be designed.",
      implementationEmpty: "No task in this feature is ready to implement.",
      selectAll: "Select all",
      clearAll: "Clear selection",
      includeBlocked: "Include dependent tasks",
      includeDesigned: "Include designed tasks",
      includeUndesigned: "Include tasks with no plan",
      undesignedNotice:
        "Tasks with no implementation plan are offered. The built-in implementation prompt has the agent settle the approach itself, but a template you overrode may still expect a registered plan.",
      afterTasks: "after {{tasks}}",
      selectedCount: "{{selected}} of {{total}} selected",
      preview: "Prompt preview",
      previewEmpty: "Select a task to preview the prompt.",
      copy: "Copy prompt",
      copied: "Copied a prompt for {{selected}} tasks.",
      failed: "The prompt could not be copied.",
    },
  },
  ja: {
    promptSettings: {
      description:
        "タスクからコピーするプロンプトのテンプレートです。CLIも同じ文面を出力します。",
      loading: "プロンプトテンプレートを読み込んでいます…",
      design: "設計プロンプト",
      designHint: "実装プランがないタスクで使います。",
      implementation: "実装プロンプト",
      implementationHint: "実装プランがあるタスクで使います。",
      placeholders:
        "使用できるプレースホルダー: {{list}}（{{required}} は必須）",
      batch: "一括実装プロンプト",
      batchHint:
        "フィーチャーで選んだタスクを 1 つのプロンプトにまとめるときに使います。",
      batchDesign: "一括設計プロンプト",
      batchDesignHint:
        "フィーチャーで選んだタスクを 1 つの設計プロンプトにまとめるときに使います。",
      batchPlaceholders:
        "使用できるプレースホルダー: {{list}}（{{required}} は必須）",
      restoreDefaults: "既定のテンプレートに戻す",
    },
    promptOverrides: {
      projectDescription:
        "このプロジェクト配下のすべてのフィーチャーで使うテンプレートを上書きします。継承にした種類は設定 › プロンプトのグローバル値に従います。",
      featureDescription:
        "このフィーチャーだけで使うテンプレートを上書きします。継承した値はプロジェクト、次に設定 › プロンプトのグローバル値に従います。",
      loading: "プロンプトのメタデータを読み込んでいます…",
      unavailable:
        "プロンプトのメタデータを取得できないため、変更は保存されません。",
      design: "設計プロンプト",
      implementation: "実装プロンプト",
      batch: "一括実装プロンプト",
      batchDesign: "一括設計プロンプト",
      mode: "取得元",
      modeFor: "{{label}} の取得元",
      inherit: "継承",
      override: "ここで上書き",
      projectSource: "プロジェクトの上書き",
      featureSource: "フィーチャーの上書き",
      globalSource: "グローバル設定",
      inheritedFrom: "{{source}} から継承",
      taskHint: "使用できるプレースホルダー: {{list}}（{{required}} は必須）",
      batchHint: "使用できるプレースホルダー: {{list}}（{{required}} は必須）",
      invalidEmpty: "上書きは空にできません。",
      invalidTooLong: "上書きは 8192 バイト以内で指定してください。",
      invalidPlaceholder: "未対応のプレースホルダー {{placeholder}} です。",
      invalidRequired: "テンプレートには {{required}} が必要です。",
    },
    batchPrompt: {
      open: "一括プロンプトをコピー",
      title: "一括プロンプトをコピー",
      tabsLabel: "一括プロンプトの種類",
      tab: { design: "設計", implementation: "実装" },
      designDescription:
        "選んだタスクをまとめた 1 つのプロンプトです。エージェントは各タスクを SubAgent に渡し、SubAgent は Next Now X から設計プロンプトを取得して実装計画を登録します。依存関係があるタスクは依存元の設計後に設計されます。グローバルの文面は設定 › プロンプトで、プロジェクトやフィーチャー固有の文面は各編集ダイアログで変更できます。",
      implementationDescription:
        "選んだタスクをまとめた 1 つのプロンプトです。各タスクの指示はエージェントが Next Now X から取得します。依存関係があるタスクは依存元の完了後に実装され、その PR は依存元の PR に積まれます。グローバルの文面は設定 › プロンプトで、プロジェクトやフィーチャー固有の文面は各編集ダイアログで変更できます。",
      designEmpty: "このフィーチャーに設計を待っているタスクはありません。",
      implementationEmpty:
        "このフィーチャーに実装へ着手できるタスクはありません。",
      selectAll: "すべて選択",
      clearAll: "選択を解除",
      includeBlocked: "依存タスクも含める",
      includeDesigned: "設計済みのタスクも含める",
      includeUndesigned: "実装計画がないタスクも含める",
      undesignedNotice:
        "実装計画がないタスクも候補に出しています。組み込みの実装プロンプトはエージェントに自分で方針を決めさせますが、上書きしたテンプレートは計画がある前提のままかもしれません。",
      afterTasks: "{{tasks}} の後",
      selectedCount: "{{total}} 件中 {{selected}} 件を選択",
      preview: "プロンプトのプレビュー",
      previewEmpty: "タスクを選ぶとプロンプトを表示します。",
      copy: "プロンプトをコピー",
      copied: "{{selected}} 件のタスクのプロンプトをコピーしました。",
      failed: "プロンプトをコピーできませんでした。",
    },
  },
} as const;
