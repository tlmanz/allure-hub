import React, { useCallback, useEffect, useRef, useState } from "react";
import { useParams, Link } from "react-router-dom";
import { dump as yamlDump } from "js-yaml";
import { api } from "../api/client";
import type { Report, ReportStats } from "../types";
import { useUpload } from "../context/UploadContext";
import { useAuth } from "../context/AuthContext";
import UploadResultsModal from "../components/UploadResultsModal";
import DeleteConfirmModal from "../components/ui/DeleteConfirmModal";
import { formatDate } from '../utils/format'

const PAGE_SIZE = 10;

function passRate(passed: number, total: number): number {
  return total === 0 ? 0 : Math.round((passed / total) * 100);
}

function getStatusConfig(status: string, failed: number) {
  if (status === "failed" || (failed > 0 && status === "")) {
    return {
      label: "CRITICAL FAILURE",
      labelClass: "bg-red-500/10 text-red-500",
      barClass: "bg-red-500",
    };
  }
  if (status === "broken" || failed > 0) {
    return {
      label: "UNSTABLE",
      labelClass: "bg-amber-500/10 text-amber-500",
      barClass: "bg-amber-500",
    };
  }
  if (status === "unknown" || status === "") {
    return {
      label: "NO DATA",
      labelClass: "bg-surface-container-high text-on-surface-variant",
      barClass: "bg-outline",
    };
  }
  return {
    label: "STABLE",
    labelClass: "bg-emerald-500/10 text-emerald-500",
    barClass: "bg-emerald-500",
  };
}

type Filter = "ALL" | "PASSED" | "FAILED";
const filterParam: Record<Filter, string> = {
  ALL: "all",
  PASSED: "passed",
  FAILED: "failed",
};

export default function ProjectDetailPage() {
  const { envId, projectId } = useParams<{
    envId: string;
    projectId: string;
  }>();
  const [builds, setBuilds] = useState<Report[]>([]);
  const [total, setTotal] = useState(0);
  const [stats, setStats] = useState<ReportStats | null>(null);
  const [buildId, setBuildId] = useState("");
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [activeFilter, setActiveFilter] = useState<Filter>("ALL");
  const [uploadModalOpen, setUploadModalOpen] = useState(false);
  const [pendingDelete, setPendingDelete] = useState<Report | null>(null);
  const [deleting, setDeleting] = useState(false);
  const [expandedConfigs, setExpandedConfigs] = useState<Set<string>>(new Set());
  const [expandedWarnings, setExpandedWarnings] = useState<Set<string>>(new Set());

  function toggleConfig(buildId: string) {
    setExpandedConfigs(prev => {
      const next = new Set(prev);
      if (next.has(buildId)) {
        next.delete(buildId);
      } else {
        next.add(buildId);
      }
      return next;
    });
  }

  function toggleWarnings(buildId: string) {
    setExpandedWarnings(prev => {
      const next = new Set(prev);
      if (next.has(buildId)) {
        next.delete(buildId);
      } else {
        next.add(buildId);
      }
      return next;
    });
  }

  function yamlToString(snapshot: Record<string, unknown>): string {
    if (!snapshot || Object.keys(snapshot).length === 0) return "(no config — server defaults used)";
    try {
      return yamlDump(snapshot, { indent: 2, lineWidth: -1 }).trimEnd();
    } catch {
      return JSON.stringify(snapshot, null, 2);
    }
  }

  function renderConfigSnapshot(snapshot: Record<string, unknown>): React.ReactNode {
    const yaml = yamlToString(snapshot);
    const lines = yaml.split("\n");

    return lines.map((line, idx) => {
      const keyValueMatch = line.match(/^(\s*-?\s*)([^:#\n][^:]*)(\s*:\s*)(.*)$/);
      const commentOnlyMatch = line.match(/^(\s*)(#.*)$/);

      const lineNode = (() => {
        if (commentOnlyMatch) {
          return (
            <>
              <span>{commentOnlyMatch[1]}</span>
              <span className="text-emerald-600 dark:text-emerald-300">{commentOnlyMatch[2]}</span>
            </>
          );
        }

        if (keyValueMatch) {
          const [, indent, key, sep, rawValue] = keyValueMatch;
          const value = rawValue.trim();
          let valueClass = "text-on-surface";
          if (value === "true" || value === "false" || value === "null") {
            valueClass = "text-violet-700 dark:text-violet-300";
          } else if (/^-?\d+(\.\d+)?$/.test(value)) {
            valueClass = "text-fuchsia-700 dark:text-fuchsia-300";
          } else if (value.startsWith('"') || value.startsWith("'")) {
            valueClass = "text-amber-700 dark:text-amber-300";
          }

          return (
            <>
              <span>{indent}</span>
              <span className="text-sky-700 dark:text-sky-300">{key}</span>
              <span className="text-on-surface-variant">{sep}</span>
              <span className={valueClass}>{rawValue}</span>
            </>
          );
        }

        const listMatch = line.match(/^(\s*)(-\s+)(.*)$/);
        if (listMatch) {
          const [, indent, dash, value] = listMatch;
          return (
            <>
              <span>{indent}</span>
              <span className="text-violet-700 dark:text-violet-300">{dash}</span>
              <span className="text-on-surface">{value}</span>
            </>
          );
        }

        return <span className="text-on-surface">{line}</span>;
      })();

      return (
        <React.Fragment key={`cfg-line-${idx}`}>
          {lineNode}
          {idx < lines.length - 1 ? "\n" : null}
        </React.Fragment>
      );
    });
  }

  const { can } = useAuth()
  const { sessions } = useUpload();
  // Track session IDs we've already reacted to so we don't refetch repeatedly.
  const handledSessionsRef = useRef<Set<string>>(new Set());

  // Abort controller for the initial page load — aborted on unmount or when
  // envId/projectId change so stale responses never update state (M-19).
  const initAbortRef = useRef<AbortController | null>(null)

  const fetchStats = useCallback((signal?: AbortSignal) => {
    if (!envId || !projectId) return;
    api
      .getReportStats(envId, projectId, signal)
      .then(setStats)
      .catch((e: Error) => { if (e.name !== 'AbortError') {} });
  }, [envId, projectId]);

  const fetchPage = useCallback(
    (filter: Filter, offset: number, append: boolean, signal?: AbortSignal) => {
      if (!envId || !projectId) return;
      const setLoad = append ? setLoadingMore : setLoading;
      setLoad(true);
      api
        .listReportsPaged(
          envId,
          projectId,
          PAGE_SIZE,
          offset,
          filterParam[filter],
          signal,
        )
        .then(({ builds: newBuilds, total: newTotal }) => {
          setBuilds((prev) => (append ? [...prev, ...newBuilds] : newBuilds));
          setTotal(newTotal);
        })
        .catch((e: Error) => { if (e.name !== 'AbortError') setError(e.message) })
        .finally(() => setLoad(false));
    },
    [envId, projectId],
  );

  useEffect(() => {
    initAbortRef.current?.abort()
    const controller = new AbortController()
    initAbortRef.current = controller
    fetchStats(controller.signal);
    fetchPage("ALL", 0, false, controller.signal);
    return () => controller.abort()
  }, [fetchStats, fetchPage]);

  // When any upload session for this project completes, refresh builds + stats
  // without requiring a manual page reload. handledSessionsRef prevents acting
  // on the same session more than once across re-renders.
  useEffect(() => {
    if (!projectId) return;
    const newlyDone = sessions.filter(
      (s) => s.projectId === projectId && s.phase === 'done' && !handledSessionsRef.current.has(s.id)
    );
    if (newlyDone.length === 0) return;
    newlyDone.forEach((s) => handledSessionsRef.current.add(s.id));
    fetchStats();
    setActiveFilter('ALL');
    fetchPage('ALL', 0, false);
  }, [sessions, projectId, fetchStats, fetchPage]);

  function handleFilterChange(f: Filter) {
    setActiveFilter(f);
    setBuilds([]);
    fetchPage(f, 0, false);
  }

  function handleLoadMore() {
    fetchPage(activeFilter, builds.length, true);
  }


  async function handleGenerate(e: React.FormEvent) {
    e.preventDefault();
    if (!envId || !projectId || !buildId.trim()) return;
    setSubmitting(true);
    setError(null);
    try {
      await api.generateReport(envId, projectId, buildId.trim());
      setBuildId("");
      fetchStats();
      // Reset to first page to show the new build
      setActiveFilter("ALL");
      fetchPage("ALL", 0, false);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to generate report.",
      );
    } finally {
      setSubmitting(false);
    }
  }

  const hasMore = builds.length < total;

  async function confirmDelete() {
    if (!pendingDelete || !envId || !projectId) return;
    setDeleting(true);
    try {
      await api.deleteReport(envId, projectId, pendingDelete.buildId);
      setBuilds((prev) => prev.filter((b) => b.buildId !== pendingDelete.buildId));
      setTotal((t) => t - 1);
      fetchStats();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete report.");
    } finally {
      setDeleting(false);
      setPendingDelete(null);
    }
  }

  return (
    <div className="flex flex-col h-full -mx-8 -my-6">
      {/* ── Sticky header: title + stats + filters ── */}
      <div className="sticky top-0 z-10 bg-background px-8 pt-6 pb-4 border-b border-outline-variant/15">
        {/* Breadcrumb */}
        <div className="flex items-center gap-2 mb-4">
          <Link
            to="/"
            className="flex items-center gap-1.5 px-2 py-1.5 text-on-surface-variant hover:bg-surface-container hover:text-on-surface transition-colors rounded-lg"
            aria-label="Back to Environments"
          >
            <span className="material-symbols-outlined text-[18px]">arrow_back</span>
            <span className="text-xs font-label">Environments</span>
          </Link>
          <span className="text-xs font-label text-on-surface-variant opacity-40">/</span>
          <Link
            to={`/environments/${envId}`}
            className="text-xs font-label text-on-surface-variant hover:text-on-surface transition-colors px-1"
          >
            {envId}
          </Link>
          <span className="text-xs font-label text-on-surface-variant opacity-40">/</span>
          <span className="text-xs font-label text-on-surface">{projectId}</span>
        </div>

        {/* Page header */}
        <div className="flex flex-col md:flex-row justify-between items-start md:items-center gap-4 mb-4">
          <div>
            <h3 className="text-3xl font-headline font-bold text-on-surface tracking-tight">
              {projectId}
            </h3>
            <p className="text-sm text-on-surface-variant font-body mt-0.5">
              Test execution analytics &amp; report history
            </p>
          </div>

          <div className="flex items-center gap-2 shrink-0">
            {can('upload') && (
              <>
                <button
                  type="button"
                  onClick={() => setUploadModalOpen(true)}
                  className="flex items-center gap-2 border border-outline-variant/30 text-on-surface px-4 py-2 rounded-lg font-headline font-bold text-sm hover:bg-surface-container active:scale-95 transition-all shrink-0"
                >
                  <span className="material-symbols-outlined text-[16px]">
                    upload_file
                  </span>
                  Upload Results
                </button>
                <form
                  onSubmit={handleGenerate}
                  className="flex items-center gap-2 shrink-0"
                >
                  <input
                    type="text"
                    value={buildId}
                    onChange={(e) => setBuildId(e.target.value)}
                    placeholder="Build Number…"
                    required
                    className="bg-surface-container-low border border-outline-variant/30 rounded-lg px-3 py-2 text-sm font-mono text-on-surface outline-none focus:ring-1 focus:ring-primary/40 placeholder:text-on-surface-variant/40 w-40"
                  />
                  <button
                    type="submit"
                    disabled={submitting}
                    className="flex items-center gap-2 bg-primary text-on-primary px-5 py-2 rounded-lg font-headline font-bold text-sm hover:brightness-110 active:scale-95 transition-all disabled:opacity-50 shrink-0"
                  >
                    <span
                      className="material-symbols-outlined text-[16px]"
                      style={{ fontVariationSettings: "'FILL' 1" }}
                    >
                      add
                    </span>
                    {submitting ? "Running…" : "NEW RUN"}
                  </button>
                </form>
              </>
            )}
          </div>
        </div>

        {error && (
          <p className="text-xs text-error bg-error/10 rounded-lg px-4 py-3 mb-4">
            {error}
          </p>
        )}

        {/* Aggregate stats */}
        {stats && stats.totalRuns > 0 && (
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 mb-4">
            <div className="bg-surface-container-low rounded-xl p-4 border border-outline-variant/10">
              <div className="flex items-center justify-between mb-2">
                <span className="text-[10px] font-label font-bold uppercase tracking-widest text-on-surface-variant">
                  Total Runs
                </span>
                <span className="material-symbols-outlined text-[16px] text-on-surface-variant opacity-40">
                  history
                </span>
              </div>
              <p className="text-2xl font-headline font-bold text-on-surface">
                {stats.totalRuns}
              </p>
            </div>
            <div className="bg-surface-container-low rounded-xl p-4 border border-outline-variant/10">
              <div className="flex items-center justify-between mb-2">
                <span className="text-[10px] font-label font-bold uppercase tracking-widest text-on-surface-variant">
                  Latest Rate
                </span>
                <span className="material-symbols-outlined text-[16px] text-on-surface-variant opacity-40">
                  speed
                </span>
              </div>
              <p
                className={`text-2xl font-headline font-bold ${stats.latestRate >= 80 ? "text-emerald-500" : stats.latestRate >= 50 ? "text-amber-500" : "text-red-500"}`}
              >
                {stats.latestRate}%
              </p>
              <div className="mt-1.5 h-1 rounded-full bg-surface-container-highest overflow-hidden">
                <div
                  className={`h-full rounded-full ${stats.latestRate >= 80 ? "bg-emerald-500" : stats.latestRate >= 50 ? "bg-amber-500" : "bg-red-500"}`}
                  style={{ width: `${stats.latestRate}%` }}
                />
              </div>
            </div>
            <div className="bg-surface-container-low rounded-xl p-4 border border-outline-variant/10">
              <div className="flex items-center justify-between mb-2">
                <span className="text-[10px] font-label font-bold uppercase tracking-widest text-on-surface-variant">
                  Avg Pass Rate
                </span>
                <span className="material-symbols-outlined text-[16px] text-on-surface-variant opacity-40">
                  analytics
                </span>
              </div>
              <p
                className={`text-2xl font-headline font-bold ${stats.avgRate >= 80 ? "text-emerald-500" : stats.avgRate >= 50 ? "text-amber-500" : "text-red-500"}`}
              >
                {stats.avgRate}%
              </p>
              <div className="mt-1.5 h-1 rounded-full bg-surface-container-highest overflow-hidden">
                <div
                  className={`h-full rounded-full ${stats.avgRate >= 80 ? "bg-emerald-500" : stats.avgRate >= 50 ? "bg-amber-500" : "bg-red-500"}`}
                  style={{ width: `${stats.avgRate}%` }}
                />
              </div>
            </div>
            <div className="bg-surface-container-low rounded-xl p-4 border border-outline-variant/10">
              <div className="flex items-center justify-between mb-2">
                <span className="text-[10px] font-label font-bold uppercase tracking-widest text-on-surface-variant">
                  Total Failures
                </span>
                <span className="material-symbols-outlined text-[16px] text-on-surface-variant opacity-40">
                  error_outline
                </span>
              </div>
              <p
                className={`text-2xl font-headline font-bold ${stats.totalFailed > 0 ? "text-red-500" : "text-emerald-500"}`}
              >
                {stats.totalFailed}
              </p>
              <p className="text-xs font-label text-on-surface-variant mt-0.5">
                across all {stats.totalRuns} run
                {stats.totalRuns !== 1 ? "s" : ""}
              </p>
            </div>
          </div>
        )}

        {/* Filter bar */}
        <div className="flex items-center gap-3">
          <div className="flex bg-surface-container-low p-1 rounded-lg">
            {(["ALL", "PASSED", "FAILED"] as const).map((f) => (
              <button
                key={f}
                onClick={() => handleFilterChange(f)}
                className={`px-4 py-1.5 text-xs font-label font-bold rounded-md transition-colors ${
                  activeFilter === f
                    ? f === "PASSED"
                      ? "bg-emerald-500/10 text-emerald-500 shadow-sm"
                      : f === "FAILED"
                        ? "bg-red-500/10 text-red-500 shadow-sm"
                        : "bg-surface-container-highest text-on-surface shadow-sm"
                    : "text-on-surface-variant hover:text-on-surface"
                }`}
              >
                {f}
              </button>
            ))}
          </div>
          {!loading && total > 0 && (
            <span className="text-xs font-label text-on-surface-variant px-2 py-1 bg-surface-container-low rounded-md">
              {total} result{total !== 1 ? "s" : ""}
            </span>
          )}
        </div>
      </div>

      {/* ── Scrollable build list ── */}
      <div className="flex-1 overflow-y-auto px-8 py-6">
        {loading ? (
          <div className="flex items-center justify-center py-24">
            <span className="text-on-surface-variant font-body animate-pulse">
              Loading reports…
            </span>
          </div>
        ) : builds.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-24 text-on-surface-variant bg-surface-container-low/30 rounded-2xl border border-dashed border-outline-variant/20">
            <span className="material-symbols-outlined text-[48px] mb-4 opacity-20">
              history
            </span>
            <p className="font-headline font-bold text-on-surface text-base">
              No builds yet
            </p>
            <p className="text-sm font-body mt-1 text-on-surface-variant">
              Enter a Build Number above and click{" "}
              <span className="text-primary font-semibold">NEW RUN</span> to
              generate your first report.
            </p>
          </div>
        ) : (
          <div className="space-y-3">
            {builds.map((r) => {
              const { label, labelClass, barClass } = getStatusConfig(
                r.status,
                r.failed,
              );
              const rate = passRate(r.passed, r.total);
              return (
                <div
                  key={r.buildId}
                  className="group relative bg-surface-container-low hover:bg-surface-container transition-all duration-300 rounded-xl overflow-hidden"
                >
                  <div className={`status-bar-left ${barClass}`} />
                  <div className="pl-7 pr-6 py-5 flex items-center justify-between gap-8">
                    <div className="flex-1 space-y-1.5 min-w-0">
                      <div className="flex items-center gap-3 flex-wrap">
                        <span
                          className={`${labelClass} text-[10px] font-black font-label px-2 py-0.5 rounded tracking-widest uppercase`}
                        >
                          {label}
                        </span>
                        <h4 className="text-base font-headline font-semibold text-on-surface group-hover:text-primary transition-colors truncate">
                          Build Number #{r.buildId}
                        </h4>
                      </div>
                      <div className="flex items-center gap-5 text-xs font-label text-on-surface-variant flex-wrap">
                        <div className="flex items-center gap-1.5">
                          <span className="material-symbols-outlined text-[14px]">
                            calendar_today
                          </span>
                          <span>{formatDate(r.createdAt)}</span>
                        </div>
                        <div className="flex items-center gap-1.5">
                          <span className="material-symbols-outlined text-[14px]">
                            speed
                          </span>
                          <span>Pass rate: {rate}%</span>
                        </div>
                        {r.uploadedBy && (
                          <div className="flex items-center gap-1.5">
                            <span className="material-symbols-outlined text-[14px]">person</span>
                            <span>{r.uploadedBy}</span>
                          </div>
                        )}
                        <button
                          type="button"
                          onClick={(e) => { e.stopPropagation(); toggleConfig(r.buildId); }}
                          className="flex items-center gap-1 text-on-surface-variant/70 hover:text-primary transition-colors"
                          aria-expanded={expandedConfigs.has(r.buildId)}
                          aria-label={expandedConfigs.has(r.buildId) ? "Hide report config" : "Show report config"}
                        >
                          <span className="material-symbols-outlined text-[13px]" aria-hidden="true">settings</span>
                          <span className="text-[11px]">Config</span>
                        </button>
                        {(r.generationWarnings?.length ?? 0) > 0 && (
                          <span className="material-symbols-outlined text-[12px] text-on-surface-variant/40" aria-hidden="true">
                            chevron_right
                          </span>
                        )}
                        {(r.generationWarnings?.length ?? 0) > 0 && (
                          <button
                            type="button"
                            onClick={(e) => { e.stopPropagation(); toggleWarnings(r.buildId); }}
                            className="flex items-center gap-1 text-amber-600/90 hover:text-amber-500 transition-colors"
                            aria-expanded={expandedWarnings.has(r.buildId)}
                            aria-label={expandedWarnings.has(r.buildId) ? "Hide generation warnings" : "Show generation warnings"}
                          >
                            <span className="material-symbols-outlined text-[13px]" aria-hidden="true">warning</span>
                            <span className="text-[11px]">Warnings ({r.generationWarnings?.length ?? 0})</span>
                          </button>
                        )}
                      </div>
                    </div>
                    <div className="flex items-center gap-10 shrink-0">
                      <div className="flex items-center gap-8">
                        {r.total > 0 && (
                          <div className="text-center">
                            <div className="text-xl font-headline font-bold text-on-surface">
                              {r.total}
                            </div>
                            <div className="text-[10px] font-label text-on-surface-variant uppercase tracking-wider">
                              Total
                            </div>
                          </div>
                        )}
                        <div className="text-center">
                          <div className="text-xl font-headline font-bold text-emerald-500">
                            {r.passed}
                          </div>
                          <div className="text-[10px] font-label text-on-surface-variant uppercase tracking-wider">
                            Passed
                          </div>
                        </div>
                        <div className="text-center">
                          <div
                            className={`text-xl font-headline font-bold ${r.failed > 0 ? "text-red-500" : "text-on-surface-variant/30"}`}
                          >
                            {r.failed}
                          </div>
                          <div className="text-[10px] font-label text-on-surface-variant uppercase tracking-wider">
                            Failed
                          </div>
                        </div>
                        {r.skipped > 0 && (
                          <div className="text-center">
                            <div className="text-xl font-headline font-bold text-amber-500">
                              {r.skipped}
                            </div>
                            <div className="text-[10px] font-label text-on-surface-variant uppercase tracking-wider">
                              Skipped
                            </div>
                          </div>
                        )}
                      </div>
                      <a
                        href={r.reportUrl}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="p-3 rounded-full bg-surface-container-highest text-on-surface hover:text-primary transition-colors active:scale-90"
                        aria-label="Open report"
                      >
                        <span className="material-symbols-outlined text-[20px]">
                          chevron_right
                        </span>
                      </a>
                      {can('manage') && (
                        <button
                          onClick={() => setPendingDelete(r)}
                          className="shrink-0 p-1.5 rounded-lg text-on-surface-variant/40 hover:text-error hover:bg-error/10
                                     opacity-0 group-hover:opacity-100 focus-visible:opacity-100 transition-all"
                          title="Delete report"
                          aria-label={`Delete build #${r.buildId}`}
                        >
                          <span className="material-symbols-outlined text-[18px]" aria-hidden="true">delete</span>
                        </button>
                      )}
                    </div>
                  </div>
                  {expandedConfigs.has(r.buildId) && (
                    <div className="mx-7 mb-4 rounded-lg bg-surface-container border border-outline-variant/20 overflow-hidden">
                      <div className="flex items-center gap-2 px-3 py-2 border-b border-outline-variant/20 bg-surface-container-highest/40">
                        <span className="material-symbols-outlined text-[14px] text-on-surface-variant">settings</span>
                        <span className="text-[11px] font-label font-bold uppercase tracking-wider text-on-surface-variant">
                          Effective allurerc.yml config
                        </span>
                      </div>
                      <pre className="px-4 py-3 text-xs font-mono text-on-surface leading-relaxed overflow-x-auto whitespace-pre">
                        {renderConfigSnapshot(r.configSnapshot)}
                      </pre>
                    </div>
                  )}
                  {expandedWarnings.has(r.buildId) && (r.generationWarnings?.length ?? 0) > 0 && (
                    <div className="mx-7 mb-4 rounded-lg bg-amber-500/10 border border-amber-500/30 dark:border-amber-300/35 overflow-hidden">
                      <div className="flex items-center gap-2 px-3 py-2 border-b border-amber-500/25 dark:border-amber-300/35 bg-amber-500/15 dark:bg-amber-300/15">
                        <span className="material-symbols-outlined text-[14px] text-amber-800 dark:text-amber-200">warning</span>
                        <span className="text-[11px] font-label font-bold uppercase tracking-wider text-amber-900 dark:text-amber-100">
                          Generation warnings
                        </span>
                      </div>
                      <ul className="px-4 py-3 text-xs font-mono text-amber-950 dark:text-amber-50 leading-relaxed space-y-2">
                        {(r.generationWarnings ?? []).map((warning, idx) => (
                          <li key={`${r.buildId}-warning-${idx}`} className="break-words">
                            {warning}
                          </li>
                        ))}
                      </ul>
                    </div>
                  )}
                </div>
              );
            })}

            {hasMore && (
              <button
                onClick={handleLoadMore}
                disabled={loadingMore}
                className="w-full flex items-center justify-center gap-2 py-3 rounded-xl border border-outline-variant/20
                           text-sm font-label font-semibold text-on-surface-variant
                           hover:bg-surface-container hover:text-on-surface transition-colors disabled:opacity-50"
              >
                <span className="material-symbols-outlined text-[18px]">
                  expand_more
                </span>
                {loadingMore
                  ? "Loading…"
                  : `Load more (${total - builds.length} remaining)`}
              </button>
            )}
          </div>
        )}
      </div>

      {envId && projectId && (
        <UploadResultsModal
          isOpen={uploadModalOpen}
          onClose={() => setUploadModalOpen(false)}
          envId={envId}
          projectId={projectId}
        />
      )}

      <DeleteConfirmModal
        isOpen={!!pendingDelete}
        onClose={() => setPendingDelete(null)}
        onConfirm={confirmDelete}
        title="Delete Report"
        description="This will permanently remove the build record, results, and generated report files."
        itemName={pendingDelete ? `Build #${pendingDelete.buildId}` : ""}
        isDeleting={deleting}
      />
    </div>
  );
}
