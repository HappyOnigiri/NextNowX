// document は 4 種類の親のいずれか 1 つにだけ属する。
export type DocumentParent =
  | { projectId: string; featureId?: never; taskId?: never; groupId?: never }
  | { projectId?: never; featureId: string; taskId?: never; groupId?: never }
  | { projectId?: never; featureId?: never; taskId: string; groupId?: never }
  | { projectId?: never; featureId?: never; taskId?: never; groupId: string };
