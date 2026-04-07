import { useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  BarChart,
  Bar,
  Cell,
  PieChart,
  Pie,
  ResponsiveContainer,
  Legend,
} from "recharts";
import { api } from "../api/client";
import type { BuildTrend, Environment, OverviewStats, Project } from "../types";
import { formatDate } from "../utils/format";

// ── Helpers ───────────────────────────────────────────────────────────────────

function passRateColor(rate: number) {
  if (rate >= 80) return "#10b981";
  if (rate >= 50) return "#f59e0b";
  return "#ef4444";
}

function passRateTextCls(rate: number) {
  if (rate >= 80) return "text-emerald-500";
  if (rate >= 50) return "text-amber-500";
  return "text-red-500";
}

function isAbortLikeError(e: unknown): boolean {
  if (typeof DOMException !== "undefined" && e instanceof DOMException) {
    return e.name === "AbortError";
  }
  if (!(e instanceof Error)) return false;
  const msg = e.message.toLowerCase();
  return (
    e.name === "AbortError" ||
    msg.includes("aborted") ||
    msg.includes("canceled") ||
    msg.includes("cancelled")
  );
}

function errorMessage(e: unknown): string {
  if (e instanceof Error) return e.message;
  return "Failed to load overview";
}

// ── Themed tooltip wrapper ────────────────────────────────────────────────────

function ChartTooltip({
  active,
  payload,
  label,
  formatter,
}: {
  active?: boolean;
  payload?: { name: string; value: number; color: string }[];
  label?: string;
  formatter?: (name: string, value: number) => string;
}) {
  if (!active || !payload?.length) return null;
  return (
    <div
      className="rounded-xl px-3 py-2 text-xs shadow-lg"
      style={{
        background: "rgb(var(--color-surface-container-high))",
        border: "1px solid rgb(var(--color-outline-variant) / 0.4)",
      }}
    >
      {label && <p className="font-bold text-on-surface mb-1">{label}</p>}
      {payload.map((p) => (
        <p key={p.name} style={{ color: p.color }} className="font-medium">
          {formatter
            ? formatter(p.name, p.value)
            : `${p.name}: ${p.value.toLocaleString()}`}
        </p>
      ))}
    </div>
  );
}

// ── Card shell ────────────────────────────────────────────────────────────────

function Card({
  className = "",
  title,
  style,
  children,
}: {
  className?: string;
  title?: string;
  style?: React.CSSProperties;
  children: React.ReactNode;
}) {
  return (
    <div
      className={`rounded-2xl flex flex-col overflow-hidden ${className}`}
      style={{
        background: "rgb(var(--color-surface-container))",
        border: "1px solid rgb(var(--color-outline-variant) / 0.3)",
        ...style,
      }}
    >
      {title && (
        <div
          className="px-4 py-2.5 border-b flex-shrink-0"
          style={{ borderColor: "rgb(var(--color-outline-variant) / 0.3)" }}
        >
          <h2 className="text-xs font-bold font-headline text-on-surface">
            {title}
          </h2>
        </div>
      )}
      {children}
    </div>
  );
}

// ── Filter dropdown ───────────────────────────────────────────────────────────

interface FilterOption {
  value: string;
  label: string;
}

function FilterDropdown({
  value,
  onChange,
  disabled,
  placeholder,
  options,
  icon,
}: {
  value: string;
  onChange: (v: string) => void;
  disabled?: boolean;
  placeholder: string;
  options: FilterOption[];
  icon: string;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    function handleOutside(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handleOutside);
    return () => document.removeEventListener("mousedown", handleOutside);
  }, [open]);

  const selected = options.find((o) => o.value === value);
  const label = selected?.label ?? placeholder;
  const isActive = !!value;

  return (
    <div className="relative" ref={ref}>
      <button
        type="button"
        disabled={disabled}
        onClick={() => setOpen((o) => !o)}
        className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-semibold font-headline transition-colors
          disabled:opacity-40 disabled:cursor-not-allowed
          ${
            isActive
              ? "bg-primary/10 text-primary border border-primary/20"
              : "text-on-surface-variant border border-outline-variant/30 hover:text-on-surface hover:bg-black/5 dark:hover:bg-white/5"
          }`}
      >
        <span className="material-symbols-outlined text-[14px]">{icon}</span>
        <span className="max-w-[120px] truncate">{label}</span>
        <span className="material-symbols-outlined text-[14px] text-on-surface-variant">
          expand_more
        </span>
      </button>

      {open && (
        <div
          className="absolute left-0 top-full mt-1 min-w-[160px] rounded-xl border shadow-lg z-50 py-1 overflow-hidden"
          style={{
            background: "rgb(var(--color-surface-container))",
            borderColor: "rgb(var(--color-outline-variant) / 0.4)",
          }}
        >
          {/* "All" option */}
          <button
            type="button"
            onClick={() => {
              onChange("");
              setOpen(false);
            }}
            className={`w-full text-left px-3 py-2 text-xs font-medium transition-colors flex items-center gap-2
              ${
                !value
                  ? "text-primary font-semibold"
                  : "text-on-surface-variant hover:text-on-surface hover:bg-black/5 dark:hover:bg-white/5"
              }`}
          >
            {!value && (
              <span className="material-symbols-outlined text-[12px] text-primary">
                check
              </span>
            )}
            <span className={!value ? "" : "ml-[20px]"}>{placeholder}</span>
          </button>

          {options.length > 0 && (
            <div
              className="my-1 border-t"
              style={{ borderColor: "rgb(var(--color-outline-variant) / 0.2)" }}
            />
          )}

          {options.map((opt) => (
            <button
              key={opt.value}
              type="button"
              onClick={() => {
                onChange(opt.value);
                setOpen(false);
              }}
              className={`w-full text-left px-3 py-2 text-xs font-medium transition-colors flex items-center gap-2
                ${
                  value === opt.value
                    ? "text-primary font-semibold"
                    : "text-on-surface-variant hover:text-on-surface hover:bg-black/5 dark:hover:bg-white/5"
                }`}
            >
              {value === opt.value && (
                <span className="material-symbols-outlined text-[12px] text-primary">
                  check
                </span>
              )}
              <span
                className={value === opt.value ? "" : "ml-[20px]"}
                title={opt.label}
              >
                {opt.label}
              </span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

// ── Page ──────────────────────────────────────────────────────────────────────

export default function OverviewPage() {
  const [stats, setStats] = useState<OverviewStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Filter state - kept in URL so the page is shareable and survives refresh
  const [params, setParams] = useSearchParams();
  const selectedEnvId = params.get("env") ?? "";
  const selectedProjectId = params.get("project") ?? "";

  const [environments, setEnvironments] = useState<Environment[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);

  function setSelectedEnvId(v: string) {
    setParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        if (v) next.set("env", v);
        else next.delete("env");
        next.delete("project"); // reset project when env changes
        return next;
      },
      { replace: true },
    );
  }

  function setSelectedProjectId(v: string) {
    setParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        if (v) next.set("project", v);
        else next.delete("project");
        return next;
      },
      { replace: true },
    );
  }

  // Load environments once on mount
  useEffect(() => {
    api
      .listEnvironments()
      .then(setEnvironments)
      .catch(() => {});
  }, []);

  // Load projects when environment changes
  useEffect(() => {
    setProjects([]);
    if (!selectedEnvId) return;
    api
      .listProjects(selectedEnvId)
      .then(setProjects)
      .catch(() => {});
  }, [selectedEnvId]);

  // Load stats when filters change
  useEffect(() => {
    const ac = new AbortController();
    let stopped = false;

    async function load() {
      setLoading(true);
      setError(null);

      const params = {
        envId: selectedEnvId || undefined,
        projectId: selectedProjectId || undefined,
      };

      for (let attempt = 0; attempt < 2; attempt += 1) {
        try {
          const data = await api.getOverviewStats(params, ac.signal);
          if (!stopped) {
            setStats(data);
            setError(null);
          }
          return;
        } catch (e) {
          if (isAbortLikeError(e)) return;
          if (attempt === 0) {
            await new Promise((resolve) => setTimeout(resolve, 250));
            continue;
          }
          if (!stopped) {
            setStats(null);
            setError(errorMessage(e));
          }
        }
      }
    }

    load().finally(() => {
      if (!stopped) setLoading(false);
    });

    return () => {
      stopped = true;
      ac.abort();
    };
  }, [selectedEnvId, selectedProjectId]);

  const scopeLabel = selectedProjectId
    ? `· ${projects.find((p) => p.id === selectedProjectId)?.name ?? selectedProjectId}`
    : selectedEnvId
      ? `· ${environments.find((e) => e.id === selectedEnvId)?.name ?? selectedEnvId}`
      : "· analytics across all environments";

  if (loading)
    return (
      <div className="flex items-center justify-center h-full text-on-surface-variant text-sm">
        Loading overview…
      </div>
    );
  if (error || !stats) {
    return (
      <div className="h-full">
        <div className="rounded-2xl border border-error/15 bg-error/8 px-4 py-3 flex items-center gap-2 text-xs text-error">
          <span className="material-symbols-outlined text-[14px]">error</span>
          {error ?? "Failed to load overview"}
        </div>
      </div>
    );
  }

  const {
    summary,
    dailyTrends,
    topFailingProjects,
    recentBuilds,
    projectBuildTrend,
  } = stats;

  // Pie chart data - overall test distribution
  const pieData = [
    { name: "Passed", value: summary.totalPassed, color: "#10b981" },
    { name: "Failed", value: summary.totalFailed, color: "#ef4444" },
    {
      name: "Skipped",
      value: Math.max(
        0,
        summary.totalPassed + summary.totalFailed === 0
          ? 0
          : dailyTrends.reduce((s, t) => s + t.skipped, 0),
      ),
      color: "#6b7280",
    },
  ].filter((d) => d.value > 0);

  // Horizontal bar data for top failing
  const barData = topFailingProjects.map((p) => ({
    name:
      p.projectName.length > 14
        ? p.projectName.slice(0, 13) + "…"
        : p.projectName,
    fullName: p.projectName,
    failures: p.totalFailed,
    passRate: p.passRate,
    to: `/environments/${encodeURIComponent(p.envId)}/projects/${encodeURIComponent(p.projectId)}`,
  }));

  return (
    <div className="h-full flex flex-col gap-3">
      {/* ── Header ── */}
      <div className="flex items-center justify-between gap-2 flex-shrink-0">
        <div className="flex items-baseline gap-2">
          <h1 className="text-xl font-black font-headline text-on-surface tracking-tight">
            Overview
          </h1>
          <span className="text-sm text-on-surface-variant">{scopeLabel}</span>
        </div>

        {/* ── Filters ── */}
        <div className="flex items-center gap-2">
          <FilterDropdown
            value={selectedEnvId}
            onChange={(v) => setSelectedEnvId(v)}
            placeholder="All environments"
            icon="folder_open"
            options={environments.map((env) => ({
              value: env.id,
              label: env.name,
            }))}
          />
          <FilterDropdown
            value={selectedProjectId}
            onChange={(v) => setSelectedProjectId(v)}
            disabled={!selectedEnvId}
            placeholder="All projects"
            icon="inventory_2"
            options={projects.map((p) => ({ value: p.id, label: p.name }))}
          />
          {(selectedEnvId || selectedProjectId) && (
            <button
              onClick={() => setParams({}, { replace: true })}
              className="p-1.5 rounded-lg text-on-surface-variant hover:text-on-surface hover:bg-black/5 dark:hover:bg-white/5 transition-colors"
              title="Clear filters"
            >
              <span className="material-symbols-outlined text-[16px]">
                close
              </span>
            </button>
          )}
        </div>
      </div>

      {/* ── Summary stat cards ── */}
      <div className="grid grid-cols-6 gap-3 flex-shrink-0">
        {(
          [
            {
              label: "Environments",
              value: summary.totalEnvironments,
              icon: "folder_open",
              cls: undefined,
            },
            {
              label: "Projects",
              value: summary.totalProjects,
              icon: "inventory_2",
              cls: undefined,
            },
            {
              label: "Builds",
              value: summary.totalBuilds,
              icon: "deployed_code",
              cls: undefined,
            },
            {
              label: "Passed",
              value: summary.totalPassed.toLocaleString(),
              icon: "check_circle",
              cls: "text-emerald-500",
            },
            {
              label: "Failed",
              value: summary.totalFailed.toLocaleString(),
              icon: "cancel",
              cls: "text-red-500",
            },
            {
              label: "Pass Rate",
              value: `${summary.overallPassRate}%`,
              icon: "monitoring",
              cls: passRateTextCls(summary.overallPassRate),
            },
          ] as const
        ).map(({ label, value, icon, cls }) => (
          <div
            key={label}
            className="rounded-2xl px-4 py-3 flex items-center gap-3"
            style={{
              background: "rgb(var(--color-surface-container))",
              border: "1px solid rgb(var(--color-outline-variant) / 0.3)",
            }}
          >
            <span
              className={`material-symbols-outlined text-[20px] flex-shrink-0 ${cls ?? "text-on-surface-variant"}`}
            >
              {icon}
            </span>
            <div className="min-w-0">
              <div
                className={`text-lg font-black font-headline leading-none tabular-nums ${cls ?? "text-on-surface"}`}
              >
                {value}
              </div>
              <div className="text-[11px] text-on-surface-variant font-medium mt-0.5">
                {label}
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* ── Main dashboard grid ── */}
      <div className="flex-1 min-h-0 grid grid-cols-3 gap-3">
        {/* Left column (2/3) */}
        <div className="col-span-2 flex flex-col gap-3 min-h-0">
          {/* 30-day trend area chart */}
          <Card title="30-Day Test Trend" className="flex-1 min-h-0">
            <div className="flex-1 min-h-0 px-2 py-2">
              {dailyTrends.length === 0 ? (
                <div className="flex items-center justify-center h-full text-sm text-on-surface-variant">
                  No data in the last 30 days
                </div>
              ) : (
                <ResponsiveContainer width="100%" height="100%">
                  <AreaChart
                    data={dailyTrends}
                    margin={{ top: 6, right: 12, left: -16, bottom: 0 }}
                  >
                    <defs>
                      <linearGradient
                        id="gradPassed"
                        x1="0"
                        y1="0"
                        x2="0"
                        y2="1"
                      >
                        <stop
                          offset="5%"
                          stopColor="#10b981"
                          stopOpacity={0.25}
                        />
                        <stop
                          offset="95%"
                          stopColor="#10b981"
                          stopOpacity={0.02}
                        />
                      </linearGradient>
                      <linearGradient
                        id="gradFailed"
                        x1="0"
                        y1="0"
                        x2="0"
                        y2="1"
                      >
                        <stop
                          offset="5%"
                          stopColor="#ef4444"
                          stopOpacity={0.25}
                        />
                        <stop
                          offset="95%"
                          stopColor="#ef4444"
                          stopOpacity={0.02}
                        />
                      </linearGradient>
                    </defs>
                    <CartesianGrid
                      strokeDasharray="3 3"
                      stroke="rgb(var(--color-outline-variant) / 0.25)"
                    />
                    <XAxis
                      dataKey="date"
                      tickFormatter={(d) => d.slice(5)}
                      tick={{
                        fontSize: 10,
                        fill: "rgb(var(--color-on-surface-variant))",
                      }}
                      axisLine={false}
                      tickLine={false}
                      interval="preserveStartEnd"
                    />
                    <YAxis
                      tick={{
                        fontSize: 10,
                        fill: "rgb(var(--color-on-surface-variant))",
                      }}
                      axisLine={false}
                      tickLine={false}
                    />
                    <Tooltip content={<ChartTooltip />} />
                    <Legend
                      iconType="circle"
                      iconSize={8}
                      wrapperStyle={{ fontSize: 11, paddingTop: 4 }}
                    />
                    <Area
                      type="monotone"
                      dataKey="passed"
                      name="Passed"
                      stroke="#10b981"
                      strokeWidth={2}
                      fill="url(#gradPassed)"
                      dot={false}
                      activeDot={{ r: 4 }}
                    />
                    <Area
                      type="monotone"
                      dataKey="failed"
                      name="Failed"
                      stroke="#ef4444"
                      strokeWidth={2}
                      fill="url(#gradFailed)"
                      dot={false}
                      activeDot={{ r: 4 }}
                    />
                  </AreaChart>
                </ResponsiveContainer>
              )}
            </div>
          </Card>

          {/* Build Trend - shown when an environment or project is selected */}
          {(selectedProjectId || selectedEnvId) && (
            <Card
              title={
                selectedProjectId
                  ? "Project Build Trend (Last 30 Builds)"
                  : "Environment Build Trend (Last 30 Builds)"
              }
              className="flex-shrink-0"
              style={{ height: "38%" }}
            >
              <div className="flex-1 min-h-0 px-2 py-2 h-full">
                {projectBuildTrend.length === 0 ? (
                  <div className="flex items-center justify-center h-full text-sm text-on-surface-variant">
                    No builds yet
                  </div>
                ) : (
                  <ResponsiveContainer width="100%" height="100%">
                    <AreaChart
                      data={projectBuildTrend}
                      margin={{ top: 6, right: 12, left: -16, bottom: 0 }}
                    >
                      <defs>
                        <linearGradient
                          id="gradBuildPassed"
                          x1="0"
                          y1="0"
                          x2="0"
                          y2="1"
                        >
                          <stop
                            offset="5%"
                            stopColor="#10b981"
                            stopOpacity={0.25}
                          />
                          <stop
                            offset="95%"
                            stopColor="#10b981"
                            stopOpacity={0.02}
                          />
                        </linearGradient>
                        <linearGradient
                          id="gradBuildFailed"
                          x1="0"
                          y1="0"
                          x2="0"
                          y2="1"
                        >
                          <stop
                            offset="5%"
                            stopColor="#ef4444"
                            stopOpacity={0.25}
                          />
                          <stop
                            offset="95%"
                            stopColor="#ef4444"
                            stopOpacity={0.02}
                          />
                        </linearGradient>
                      </defs>
                      <CartesianGrid
                        strokeDasharray="3 3"
                        stroke="rgb(var(--color-outline-variant) / 0.25)"
                      />
                      <XAxis
                        dataKey="buildId"
                        tick={{
                          fontSize: 10,
                          fill: "rgb(var(--color-on-surface-variant))",
                        }}
                        axisLine={false}
                        tickLine={false}
                        interval="preserveStartEnd"
                        tickFormatter={(id: string) =>
                          id.length > 10 ? id.slice(0, 9) + "…" : id
                        }
                      />
                      <YAxis
                        tick={{
                          fontSize: 10,
                          fill: "rgb(var(--color-on-surface-variant))",
                        }}
                        axisLine={false}
                        tickLine={false}
                      />
                      <Tooltip
                        content={({ active, payload }) => {
                          if (!active || !payload?.length) return null;
                          const d = payload[0].payload as BuildTrend;
                          return (
                            <div
                              className="rounded-xl px-3 py-2 text-xs shadow-lg"
                              style={{
                                background:
                                  "rgb(var(--color-surface-container-high))",
                                border:
                                  "1px solid rgb(var(--color-outline-variant) / 0.4)",
                              }}
                            >
                              <p className="font-bold text-on-surface mb-1">
                                {d.buildId}
                              </p>
                              <p className="text-on-surface-variant mb-1">
                                {d.createdAt}
                              </p>
                              <p style={{ color: "#10b981" }}>
                                Passed: {d.passed.toLocaleString()}
                              </p>
                              <p style={{ color: "#ef4444" }}>
                                Failed: {d.failed.toLocaleString()}
                              </p>
                              {d.skipped > 0 && (
                                <p style={{ color: "#6b7280" }}>
                                  Skipped: {d.skipped.toLocaleString()}
                                </p>
                              )}
                            </div>
                          );
                        }}
                      />
                      <Legend
                        iconType="circle"
                        iconSize={8}
                        wrapperStyle={{ fontSize: 11, paddingTop: 4 }}
                      />
                      <Area
                        type="monotone"
                        dataKey="passed"
                        name="Passed"
                        stroke="#10b981"
                        strokeWidth={2}
                        fill="url(#gradBuildPassed)"
                        dot={{ r: 3, fill: "#10b981", strokeWidth: 0 }}
                        activeDot={{ r: 5 }}
                      />
                      <Area
                        type="monotone"
                        dataKey="failed"
                        name="Failed"
                        stroke="#ef4444"
                        strokeWidth={2}
                        fill="url(#gradBuildFailed)"
                        dot={{ r: 3, fill: "#ef4444", strokeWidth: 0 }}
                        activeDot={{ r: 5 }}
                      />
                    </AreaChart>
                  </ResponsiveContainer>
                )}
              </div>
            </Card>
          )}

          {/* Recent builds table */}
          <Card
            title="Recent Builds"
            className="flex-shrink-0"
            style={{ maxHeight: "38%" }}
          >
            {recentBuilds.length === 0 ? (
              <p className="px-4 py-4 text-sm text-on-surface-variant">
                No builds yet
              </p>
            ) : (
              <div className="overflow-y-auto flex-1">
                {recentBuilds.map((b) => {
                  const rate =
                    b.total > 0 ? Math.round((b.passed / b.total) * 100) : 0;
                  return (
                    <div
                      key={b.buildId}
                      className="flex items-center gap-3 px-4 py-2 hover:bg-black/[0.02] dark:hover:bg-white/[0.02] transition-colors"
                    >
                      <span
                        className={`w-1.5 h-1.5 rounded-full flex-shrink-0 ${b.failed > 0 ? "bg-red-500" : b.total > 0 ? "bg-emerald-500" : "bg-outline"}`}
                      />
                      <div className="flex-1 min-w-0">
                        <Link
                          to={`/environments/${encodeURIComponent(b.envId ?? "")}/projects/${encodeURIComponent(b.projectId ?? "")}`}
                          className="text-xs font-semibold text-on-surface hover:text-primary transition-colors truncate block"
                        >
                          {b.projectId}
                          <span className="font-normal text-on-surface-variant">
                            {" "}
                            · {b.buildId}
                          </span>
                        </Link>
                        <p className="text-[10px] text-on-surface-variant">
                          {formatDate(b.createdAt as unknown as string)}
                        </p>
                      </div>
                      <div className="hidden sm:flex items-center gap-2 text-[11px] tabular-nums">
                        <span className="text-emerald-500">{b.passed}✓</span>
                        {b.failed > 0 && (
                          <span className="text-red-500">{b.failed}✗</span>
                        )}
                        {b.skipped > 0 && (
                          <span className="text-on-surface-variant">
                            {b.skipped}–
                          </span>
                        )}
                      </div>
                      <span
                        className={`text-[11px] font-bold tabular-nums w-9 text-right ${passRateTextCls(rate)}`}
                      >
                        {b.total > 0 ? `${rate}%` : "-"}
                      </span>
                      {b.reportUrl && (
                        <a
                          href={b.reportUrl}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-on-surface-variant hover:text-primary transition-colors"
                          title="Open report"
                        >
                          <span className="material-symbols-outlined text-[16px]">
                            open_in_new
                          </span>
                        </a>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </Card>
        </div>

        {/* Right column (1/3) */}
        <div className="flex flex-col gap-3 min-h-0">
          {/* Pass / Fail / Skipped donut */}
          <Card
            title="Test Distribution"
            className="flex-shrink-0"
            style={{ height: "42%" }}
          >
            <div className="flex-1 min-h-0 py-1">
              {pieData.length === 0 ? (
                <div className="flex items-center justify-center h-full text-sm text-on-surface-variant">
                  No test data
                </div>
              ) : (
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie
                      data={pieData}
                      cx="50%"
                      cy="48%"
                      innerRadius="48%"
                      outerRadius="72%"
                      paddingAngle={2}
                      dataKey="value"
                    >
                      {pieData.map((entry) => (
                        <Cell
                          key={entry.name}
                          fill={entry.color}
                          stroke="transparent"
                        />
                      ))}
                    </Pie>
                    <Tooltip
                      content={({ active, payload }) => {
                        if (!active || !payload?.length) return null;
                        const d = payload[0];
                        const total = pieData.reduce((s, p) => s + p.value, 0);
                        const pct =
                          total > 0
                            ? Math.round(((d.value as number) / total) * 100)
                            : 0;
                        return (
                          <div
                            className="rounded-xl px-3 py-2 text-xs shadow-lg"
                            style={{
                              background:
                                "rgb(var(--color-surface-container-high))",
                              border:
                                "1px solid rgb(var(--color-outline-variant) / 0.4)",
                            }}
                          >
                            <p
                              style={{ color: d.payload?.color }}
                              className="font-bold"
                            >
                              {d.name}
                            </p>
                            <p className="text-on-surface">
                              {(d.value as number).toLocaleString()}{" "}
                              <span className="text-on-surface-variant">
                                ({pct}%)
                              </span>
                            </p>
                          </div>
                        );
                      }}
                    />
                    <Legend
                      iconType="circle"
                      iconSize={8}
                      wrapperStyle={{ fontSize: 11 }}
                    />
                  </PieChart>
                </ResponsiveContainer>
              )}
            </div>
          </Card>

          {/* Top failing projects - horizontal bar */}
          <Card title="Top Failing Projects" className="flex-1 min-h-0">
            <div className="flex-1 min-h-0 px-1 py-2">
              {barData.length === 0 ? (
                <div className="flex items-center justify-center h-full text-sm text-on-surface-variant">
                  No failures recorded
                </div>
              ) : (
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart
                    data={barData}
                    layout="vertical"
                    margin={{ top: 0, right: 40, left: 4, bottom: 0 }}
                    barCategoryGap="25%"
                  >
                    <CartesianGrid
                      strokeDasharray="3 3"
                      horizontal={false}
                      stroke="rgb(var(--color-outline-variant) / 0.25)"
                    />
                    <XAxis
                      type="number"
                      tick={{
                        fontSize: 10,
                        fill: "rgb(var(--color-on-surface-variant))",
                      }}
                      axisLine={false}
                      tickLine={false}
                    />
                    <YAxis
                      type="category"
                      dataKey="name"
                      width={80}
                      tick={{
                        fontSize: 10,
                        fill: "rgb(var(--color-on-surface-variant))",
                      }}
                      axisLine={false}
                      tickLine={false}
                    />
                    <Tooltip
                      content={({ active, payload }) => {
                        if (!active || !payload?.length) return null;
                        const d = payload[0].payload;
                        return (
                          <div
                            className="rounded-xl px-3 py-2 text-xs shadow-lg"
                            style={{
                              background:
                                "rgb(var(--color-surface-container-high))",
                              border:
                                "1px solid rgb(var(--color-outline-variant) / 0.4)",
                            }}
                          >
                            <p className="font-bold text-on-surface mb-1">
                              {d.fullName}
                            </p>
                            <p style={{ color: "#ef4444" }}>
                              {d.failures.toLocaleString()} failures
                            </p>
                            <p style={{ color: passRateColor(d.passRate) }}>
                              Pass rate: {d.passRate}%
                            </p>
                          </div>
                        );
                      }}
                    />
                    <Bar
                      dataKey="failures"
                      name="Failures"
                      radius={[0, 4, 4, 0]}
                    >
                      {barData.map((d) => (
                        <Cell
                          key={d.name}
                          fill={passRateColor(d.passRate)}
                          fillOpacity={0.85}
                        />
                      ))}
                    </Bar>
                  </BarChart>
                </ResponsiveContainer>
              )}
            </div>
          </Card>
        </div>
      </div>
    </div>
  );
}
