import { createContext, useContext } from "react";

// disconnectedNoticeMs は切断表示を出すまでの継続時間。単発の切断は環境によって
// 頻繁に起こり得るので、そのたびに警告を出すと表示自体が信用されなくなる。
export const disconnectedNoticeMs = 30_000;

// localWriteRevisionWindowMs は、自分の書き込みが起こしたリビジョンとみなす猶予。
// サーバーの検査周期より長く取る。取り違えても失うのは重複した取り直しだけである。
export const localWriteRevisionWindowMs = 3_000;

export interface RevisionStreamStatus {
  connected: boolean;
  // stale は切断が disconnectedNoticeMs 以上続いていることを表す。
  stale: boolean;
}

export const RevisionStreamContext = createContext<
  RevisionStreamStatus | undefined
>(undefined);

export function useRevisionStreamStatus(): RevisionStreamStatus {
  const status = useContext(RevisionStreamContext);
  if (!status)
    throw new Error("useRevisionStreamStatus requires RevisionStreamContext");
  return status;
}
