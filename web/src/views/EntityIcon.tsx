import {
  CircleDot,
  Folder,
  FolderOpen,
  GitPullRequest,
  Layers,
  User,
} from "lucide-react";

// レコードの種類ごとに 1 つの図形を、全画面で共有する。サイドバーで覚えた
// フォルダはキューの行でも同じものとして読めるので、アイコンは飾りではなく
// 項目名の代わりになる。
const entityIcons = {
  project: Folder,
  // feature はブランチそのものではなく task と PR の束なので、git 系ではなく
  // 重なりのグリフを充てる。PR の丸と線に紛れないという利点もある。
  feature: Layers,
  task: CircleDot,
  pullRequest: GitPullRequest,
  assignee: User,
} as const;

export type EntityKind = keyof typeof entityIcons;

// 開いたフォルダは種類ではなく表示状態なので、EntityKind ではなく project
// 専用のプロパティで選ぶ。展開した行を持つサイドバーだけが要求できる。
type EntityIconProps = { size: number } & (
  | { kind: "project"; open?: boolean }
  | { kind: Exclude<EntityKind, "project">; open?: never }
);

// アイコン単体では意味を持たせない。支援技術は図形を読めないので、呼び出し側は
// 必ず名前・見出し・視覚的に隠したラベルのいずれかを隣に置く。
export function EntityIcon({ kind, size, open }: EntityIconProps) {
  const Icon = kind === "project" && open ? FolderOpen : entityIcons[kind];
  return (
    <Icon aria-hidden="true" focusable="false" size={size} strokeWidth={1.75} />
  );
}
