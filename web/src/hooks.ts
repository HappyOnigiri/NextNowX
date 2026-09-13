import {
  useMutation,
  useQuery,
  useQueryClient,
  type QueryClient,
} from "@tanstack/react-query";
import { useCallback, useContext, useEffect, useRef, useState } from "react";
import {
  getConfig,
  getDebugReport,
  getPromptTemplates,
  getSnapshot,
  getSyncStatus,
  syncIfDue,
} from "./api";
import type { QueryDiagnostic } from "./debug-text";
import { setDisplayLanguage } from "./i18n";
import { isSupportedLanguage } from "./i18n/settings";
import { runWhenIdle } from "./refresh-gate";
import {
  disconnectedNoticeMs,
  localWriteRevisionWindowMs,
  RevisionStreamContext,
  type RevisionStreamStatus,
} from "./revision-status";
import { startRevisionStream } from "./revision-stream";

const snapshotKey = ["snapshot"] as const;
const configKey = ["github-config"] as const;
const promptTemplatesKey = ["prompt-templates"] as const;
const syncStatusKey = ["github-sync-status"] as const;
const debugReportKey = ["debug-report"] as const;

// query オブジェクトをそのまま返して React Query のプロパティ追跡を保つ。分解す
// ると全 getter を読むため、ポーリングのたびに変わる isFetching のような無関係な
// 変化でも利用側が再描画される。
export function useSnapshot() {
  return useQuery({ queryKey: snapshotKey, queryFn: getSnapshot });
}

export function useConfig() {
  return useQuery({ queryKey: configKey, queryFn: getConfig });
}

// 表示言語はサーバーが解決した実効言語に従う。設定が auto でも、画面の言語と
// サーバーが描くプロンプトの言語がずれないようにするためである。
export function useDisplayLanguage() {
  const config = useConfig();
  const effective = config.data?.effectiveLanguage;
  useEffect(() => {
    if (!isSupportedLanguage(effective)) return;
    void setDisplayLanguage(effective);
  }, [effective]);
}

// 言語を変えると組み込みテンプレートも変わるので、設定と一緒にテンプレートの
// キャッシュも捨てる。その場で描画するプロンプトはキャッシュを持たない。
export function useLanguageMutation<TVariables, TData>(
  mutationFn: (input: TVariables) => Promise<TData>,
) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn,
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: configKey }),
        queryClient.invalidateQueries({ queryKey: promptTemplatesKey }),
      ]);
    },
  });
}

// テンプレートは設定パネルでしか読まないので、shell が保持し続ける query には
// 含めず、パネルが初めてマウントされたときに読み込む。
export function usePromptTemplates() {
  return useQuery({
    queryKey: promptTemplatesKey,
    queryFn: getPromptTemplates,
  });
}

export function usePromptTemplatesMutation<TVariables, TData>(
  mutationFn: (input: TVariables) => Promise<TData>,
) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn,
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: promptTemplatesKey }),
  });
}

// レポートは取得時点のスナップショットなので、自動で再取得はしない。収集時に
// データベースと設定ファイルを読むため、`enabled` で debug タブが開かれるまで
// リクエストを発行しない。
export function useDebugReport(enabled: boolean) {
  return useQuery({
    queryKey: debugReportKey,
    queryFn: getDebugReport,
    enabled,
    staleTime: Infinity,
    gcTime: 0,
  });
}

export function useAutoSync(enabled = true) {
  const queryClient = useQueryClient();
  const checking = useRef(false);
  const status = useQuery({
    queryKey: syncStatusKey,
    queryFn: getSyncStatus,
    enabled,
  });
  const check = useMutation({
    mutationFn: syncIfDue,
    onMutate: () => {
      checking.current = true;
    },
    onSuccess: async (response) => {
      if (response.status)
        queryClient.setQueryData(syncStatusKey, response.status);
      if (response.ran)
        await queryClient.invalidateQueries({ queryKey: snapshotKey });
    },
    onSettled: () => {
      checking.current = false;
    },
  });
  const { mutate } = check;

  useEffect(() => {
    if (!enabled) return;
    const run = () => {
      if (document.visibilityState === "visible" && !checking.current) mutate();
    };
    run();
    const interval = window.setInterval(run, 60_000);
    document.addEventListener("visibilitychange", run);
    window.addEventListener("focus", run);
    return () => {
      window.clearInterval(interval);
      document.removeEventListener("visibilitychange", run);
      window.removeEventListener("focus", run);
    };
  }, [enabled, mutate]);

  return { status, checking: check.isPending, error: check.error };
}

// useRevisionStream はローカルデータベースの変更を購読し、届くたびにサーバー側の
// 状態を写した query を捨てる。CLI や別タブの書き込みもこれで反映される。
// AppShell から 1 回だけ呼ぶ。1 タブにつきストリームは 1 本だけにする。
export function useRevisionStream(): RevisionStreamStatus {
  const queryClient = useQueryClient();
  const [connected, setConnected] = useState(false);
  const [stale, setStale] = useState(false);

  useEffect(() => {
    const writes = watchLocalWrites(queryClient);
    // GitHub 同期の実行状態も同じデータベースにあるので、mutation と同じ 2 つを捨てる。
    // 進行中のポインタジェスチャを壊さないよう、取り直しは gate 越しに行う。
    const refresh = () => {
      runWhenIdle(() => {
        void queryClient.invalidateQueries({ queryKey: snapshotKey });
        void queryClient.invalidateQueries({ queryKey: syncStatusKey });
      });
    };
    let notice: ReturnType<typeof setTimeout> | undefined;
    const stop = startRevisionStream({
      // 接続のたびに無条件で捨てる。切断中の変更は、これで 1 回の再取得に畳まれる。
      onConnect: () => {
        if (notice !== undefined) clearTimeout(notice);
        notice = undefined;
        setConnected(true);
        setStale(false);
        refresh();
      },
      // このタブ自身の書き込みが起こしたリビジョンでは取り直さない。useDomainMutation
      // がすでに同じ 2 つを捨てており、二重の取り直しは画面を触っている最中に届く。
      onRevision: () => {
        if (writes.recent()) return;
        refresh();
      },
      onDisconnect: () => {
        setConnected(false);
        if (notice !== undefined) return;
        notice = setTimeout(() => {
          setStale(true);
        }, disconnectedNoticeMs);
      },
    });
    // StrictMode の二重マウントで 2 本張らないよう、後片付けで必ず止める。
    return () => {
      if (notice !== undefined) clearTimeout(notice);
      writes.stop();
      stop();
    };
  }, [queryClient]);

  return { connected, stale };
}

// watchLocalWrites はこのタブの mutation が最後に走った時刻を追う。サーバーは自分の
// 書き込みも他人のものと同じく検知するので、これがないと 1 回の変更で 2 回取り直す。
function watchLocalWrites(queryClient: QueryClient) {
  let lastAt = 0;
  // 同じ mutation について複数のイベントが届くので、数ではなく id の集合で数える。
  const pending = new Set<number>();
  const unsubscribe = queryClient.getMutationCache().subscribe((event) => {
    const mutation = event.mutation;
    if (!mutation) return;
    if (mutation.state.status === "pending") {
      pending.add(mutation.mutationId);
      return;
    }
    if (!pending.delete(mutation.mutationId)) return;
    lastAt = Date.now();
  });
  return {
    recent: () =>
      pending.size > 0 || Date.now() - lastAt < localWriteRevisionWindowMs,
    stop: unsubscribe,
  };
}

// useQueryDiagnostics は shell が保持する query のキャッシュ状態を返す。購読では
// なくキャッシュを読むので、debug タブを開いても新たな取得は始まらず、すでに失敗
// している query も隠れない。
export function useQueryDiagnostics(): QueryDiagnostic[] {
  const queryClient = useQueryClient();
  const stream = useContext(RevisionStreamContext);
  const queries = [snapshotKey, configKey, syncStatusKey].map((key) => {
    const state = queryClient.getQueryState(key);
    if (!state) return { name: key[0], state: "not requested" };
    if (state.error)
      return { name: key[0], state: `error: ${state.error.message}` };
    return { name: key[0], state: `${state.status}, ${state.fetchStatus}` };
  });
  return [...queries, { name: "revision-stream", state: streamState(stream) }];
}

function streamState(stream: RevisionStreamStatus | undefined): string {
  if (!stream) return "not started";
  if (stream.connected) return "connected";
  return stream.stale ? "disconnected, stale" : "disconnected";
}

export function useDomainMutation<TVariables, TData>(
  mutationFn: (input: TVariables) => Promise<TData>,
) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn,
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: snapshotKey }),
        queryClient.invalidateQueries({ queryKey: syncStatusKey }),
      ]);
    },
  });
}

// 部分的に書き込んだところで失敗した mutation は onSuccess を通らないので、
// 書き込めた分をグラフへ出すために呼び出し側からスナップショットを捨てる。
export function useSnapshotRefresh() {
  const queryClient = useQueryClient();
  return useCallback(
    () => queryClient.invalidateQueries({ queryKey: snapshotKey }),
    [queryClient],
  );
}

export function useConfigMutation<TVariables, TData>(
  mutationFn: (input: TVariables) => Promise<TData>,
) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: configKey }),
  });
}
