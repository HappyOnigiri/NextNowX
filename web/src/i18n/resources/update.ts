export const update = {
  en: {
    update: {
      available: "Update to {{version}}",
      dialogTitle: "Next Now X {{version}} is available",
      dialogDescription:
        "You are running {{current}}. Installing {{latest}} replaces the nnx binary and keeps your data.",
      install: "Update now",
      skip: "Skip this version",
      emptyNotes: "This release has no notes.",
      installed:
        "Installed {{version}}. The background server restarts itself.",
      restartRequired:
        "Installed {{version}}. Restart the running nnx serve to use it.",
    },
  },
  ja: {
    update: {
      available: "{{version}} に更新",
      dialogTitle: "Next Now X {{version}} が公開されています",
      dialogDescription:
        "現在は {{current}} です。{{latest}} を入れると nnx のバイナリだけが置き換わり、データはそのまま残ります。",
      install: "今すぐ更新",
      skip: "このバージョンをスキップ",
      emptyNotes: "このリリースには本文がありません。",
      installed: "{{version}} を入れました。常駐サーバーは自分で再起動します。",
      restartRequired:
        "{{version}} を入れました。稼働中の nnx serve を再起動してください。",
    },
  },
} as const;
