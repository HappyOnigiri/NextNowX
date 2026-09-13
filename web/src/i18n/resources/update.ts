export const update = {
  en: {
    update: {
      available: "Update to {{version}}",
      dialogTitle: "PRX {{version}} is available",
      dialogDescription:
        "You are running {{current}}. Installing {{latest}} replaces the prx binary and keeps your data.",
      install: "Update now",
      skip: "Skip this version",
      emptyNotes: "This release has no notes.",
      installed:
        "Installed {{version}}. The background server restarts itself.",
      restartRequired:
        "Installed {{version}}. Restart the running prx serve to use it.",
    },
  },
  ja: {
    update: {
      available: "{{version}} に更新",
      dialogTitle: "PRX {{version}} が公開されています",
      dialogDescription:
        "現在は {{current}} です。{{latest}} を入れると prx のバイナリだけが置き換わり、データはそのまま残ります。",
      install: "今すぐ更新",
      skip: "このバージョンをスキップ",
      emptyNotes: "このリリースには本文がありません。",
      installed: "{{version}} を入れました。常駐サーバーは自分で再起動します。",
      restartRequired:
        "{{version}} を入れました。稼働中の prx serve を再起動してください。",
    },
  },
} as const;
