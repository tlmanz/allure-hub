import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from "react";
import { api } from "../api/client";
import type { UploadSession } from "../types";

interface UploadContextValue {
  sessions: UploadSession[];
  drawerOpen: boolean;
  openDrawer: () => void;
  closeDrawer: () => void;
  startUpload: (
    envId: string,
    projectId: string,
    buildId: string,
    file: File,
    reportConfig?: Record<string, unknown>,
  ) => void;
  cancelUpload: (buildId: string) => void;
  retryUpload: (session: UploadSession) => void;
  deleteSession: (id: string) => Promise<void>;
}

const UploadContext = createContext<UploadContextValue | null>(null);

export function UploadProvider({ children }: { children: React.ReactNode }) {
  const [sessions, setSessions] = useState<UploadSession[]>([]);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const esRef = useRef<EventSource | null>(null);
  // Track AbortControllers keyed by buildId so uploads can be cancelled (M-27).
  const abortControllersRef = useRef<Map<string, AbortController>>(new Map());

  // Merge an incoming session update into the list (upsert by id).
  const upsert = useCallback((updated: UploadSession) => {
    setSessions((prev) => {
      const idx = prev.findIndex((s) => s.id === updated.id);
      if (idx === -1) return [updated, ...prev];
      const next = [...prev];
      next[idx] = updated;
      return next;
    });
  }, []);

  // Fetch snapshot on mount, then subscribe to SSE with exponential backoff
  // reconnect so repeated failures don't hammer the server (M-16).
  useEffect(() => {
    api
      .listUploadSessions()
      .then((list) => setSessions(list))
      .catch(() => {});

    let retryDelay = 1000; // ms - doubles on each failure, capped at 30 s
    let retryTimer: ReturnType<typeof setTimeout> | null = null;
    let stopped = false;

    function connect() {
      const es = new EventSource("/api/uploads/stream");
      esRef.current = es;

      es.addEventListener("session_updated", (e: MessageEvent) => {
        try {
          const session: UploadSession = JSON.parse(e.data);
          upsert(session);
        } catch {}
      });

      es.addEventListener("open", () => {
        retryDelay = 1000; // reset backoff on successful connection
      });

      es.onerror = () => {
        es.close();
        esRef.current = null;
        if (!stopped) {
          retryTimer = setTimeout(() => {
            retryDelay = Math.min(retryDelay * 2, 30_000);
            connect();
          }, retryDelay);
        }
      };
    }

    connect();

    return () => {
      stopped = true;
      if (retryTimer !== null) clearTimeout(retryTimer);
      esRef.current?.close();
      esRef.current = null;
    };
  }, [upsert]);

  const startUpload = useCallback(
    (
      envId: string,
      projectId: string,
      buildId: string,
      file: File,
      reportConfig?: Record<string, unknown>,
    ) => {
      setDrawerOpen(true);
      const controller = new AbortController();
      abortControllersRef.current.set(buildId, controller);
      // Fire-and-forget - the SSE stream keeps the UI updated for server-side
      // phase changes. On client-side failure we surface a synthetic failed
      // session so the drawer shows the error instead of silently swallowing it.
      api
        .uploadResults(
          envId,
          projectId,
          buildId,
          file,
          () => {},
          controller.signal,
          reportConfig,
        )
        .catch((err: unknown) => {
          abortControllersRef.current.delete(buildId);
          // Suppress AbortError - user cancelled intentionally.
          if (err instanceof Error && err.name === "AbortError") return;
          const message = err instanceof Error ? err.message : "Upload failed";
          const now = new Date().toISOString();
          upsert({
            id: `client-err-${Date.now()}`,
            uploadId: "",
            buildId,
            projectId,
            envId,
            fileName: file.name,
            totalSize: file.size,
            totalChunks: 0,
            receivedChunks: 0,
            phase: "failed",
            error: message,
            startedAt: now,
            passed: 0,
            failed: 0,
            skipped: 0,
            total: 0,
          });
        })
        .then(() => {
          abortControllersRef.current.delete(buildId);
        });
    },
    [upsert],
  );

  // retryUpload resumes a failed session from the point of failure:
  // - failedAtPhase === 'assembling': re-triggers assembly then generation
  // - failedAtPhase === 'generating': re-triggers generation only
  const retryUpload = useCallback(
    (session: UploadSession) => {
      const { envId, projectId, buildId, uploadId, failedAtPhase } = session;
      if (!envId || !projectId || !buildId) return;

      const runGenerate = () =>
        api
          .generateReport(envId, projectId, buildId, true)
          .catch((err: unknown) => {
            const message =
              err instanceof Error ? err.message : "Generation failed";
            upsert({ ...session, phase: "failed", error: message });
          });

      if (failedAtPhase === "assembling") {
        api
          .retryAssembly(envId, projectId, uploadId)
          .then(runGenerate)
          .catch((err: unknown) => {
            const message =
              err instanceof Error ? err.message : "Assembly failed";
            upsert({ ...session, phase: "failed", error: message });
          });
      } else if (failedAtPhase === "generating") {
        runGenerate();
      }
    },
    [upsert],
  );

  const cancelUpload = useCallback((buildId: string) => {
    const controller = abortControllersRef.current.get(buildId);
    if (controller) {
      controller.abort();
      abortControllersRef.current.delete(buildId);
    }
  }, []);

  const deleteSession = useCallback(async (id: string) => {
    await api.deleteUploadSession(id);
    setSessions((prev) => prev.filter((s) => s.id !== id));
  }, []);

  const openDrawer = useCallback(() => setDrawerOpen(true), []);
  const closeDrawer = useCallback(() => setDrawerOpen(false), []);

  return (
    <UploadContext.Provider
      value={{
        sessions,
        drawerOpen,
        openDrawer,
        closeDrawer,
        startUpload,
        cancelUpload,
        retryUpload,
        deleteSession,
      }}
    >
      {children}
    </UploadContext.Provider>
  );
}

export function useUpload(): UploadContextValue {
  const ctx = useContext(UploadContext);
  if (!ctx) throw new Error("useUpload must be used inside <UploadProvider>");
  return ctx;
}

export function useActiveUploadCount(): number {
  const { sessions } = useUpload();
  return sessions.filter(
    (s) =>
      s.phase === "uploading" ||
      s.phase === "assembling" ||
      s.phase === "generating",
  ).length;
}
