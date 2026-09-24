import { createContext, useContext } from "react";
import type { GitHubSyncStatus } from "./gen/nnx/v1/nnx_pb";

export interface AutoSyncStatus {
  status: {
    data: GitHubSyncStatus | undefined;
    isError: boolean;
  };
  checking: boolean;
  error: Error | null;
}

export const AutoSyncStatusContext = createContext<AutoSyncStatus | undefined>(
  undefined,
);

export function useAutoSyncStatus() {
  const status = useContext(AutoSyncStatusContext);
  if (!status)
    throw new Error("useAutoSyncStatus requires AutoSyncStatusContext");
  return status;
}
