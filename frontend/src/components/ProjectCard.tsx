import React, { useState } from "react";
import { Link } from "react-router-dom";
import { api } from "../api/client";
import DeleteConfirmModal from "./ui/DeleteConfirmModal";
import { useAuth } from "../context/AuthContext";

export interface ProjectCardProps {
  name: string;
  id: string;
  envId: string;
  builds: number;
  status: "passed" | "failed" | "inactive";
  lastRun: string;
  lastTotal: number;
  lastPassed: number;
  lastFailed: number;
  onDeleted?: () => void;
}

const CIRCUMFERENCE = 2 * Math.PI * 28; // r=28

const statusConfig = {
  passed:   { label: "Live Production", labelColor: "text-primary" },
  failed:   { label: "Degraded",        labelColor: "text-error" },
  inactive: { label: "No Runs Yet",     labelColor: "text-on-surface-variant" },
};

const MIN_SEG = 4; // minimum SVG units to render a segment

interface SegmentRingProps {
  passed: number;
  failed: number;
  total: number;
  label: string;
}

const SegmentRing: React.FC<SegmentRingProps> = ({ passed, failed, total, label }) => {
  const skipped = Math.max(0, total - passed - failed);
  const hasData = total > 0;

  const pLen = hasData ? (passed  / total) * CIRCUMFERENCE : 0;
  const fLen = hasData ? (failed  / total) * CIRCUMFERENCE : 0;
  const sLen = hasData ? (skipped / total) * CIRCUMFERENCE : 0;

  // Each circle: dasharray=[segLen, rest], dashoffset=C-startPos
  const arc = (len: number, start: number, color: string) => {
    if (len < MIN_SEG) return null;
    return (
      <circle
        cx="32" cy="32" r="28"
        fill="none"
        stroke={color}
        strokeWidth="6"
        strokeLinecap="round"
        strokeDasharray={`${len} ${CIRCUMFERENCE - len}`}
        strokeDashoffset={CIRCUMFERENCE - start}
      />
    );
  };

  return (
    <div className="relative w-16 h-16 flex items-center justify-center shrink-0">
      <svg className="w-full h-full -rotate-90" viewBox="0 0 64 64">
        {/* Track */}
        <circle cx="32" cy="32" r="28" fill="none" stroke="rgb(var(--color-surface-container-high))" strokeWidth="6" />
        {hasData && (
          <>
            {arc(pLen, 0,          "#10b981")}
            {arc(fLen, pLen,        "#ef4444")}
            {arc(sLen, pLen + fLen, "#f59e0b")}
          </>
        )}
      </svg>
      <span className="absolute text-[10px] font-black font-headline text-on-surface">
        {label}
      </span>
    </div>
  );
};

const ProjectCard: React.FC<ProjectCardProps> = ({
  name,
  id,
  envId,
  builds,
  status,
  lastRun,
  lastTotal,
  lastPassed,
  lastFailed,
  onDeleted,
}) => {
  const { can } = useAuth();
  const [modalOpen, setModalOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const cfg      = statusConfig[status];
  const ringPct   = lastTotal > 0 ? Math.round((lastPassed / lastTotal) * 100) : 0;
  const ringLabel = lastTotal === 0 ? "—" : `${ringPct}%`;

  async function handleDelete() {
    setDeleting(true);
    setDeleteError(null);
    try {
      await api.deleteProject(envId, id);
      setModalOpen(false);
      onDeleted?.();
    } catch (e) {
      setDeleteError(e instanceof Error ? e.message : 'Delete failed. Please try again.');
    } finally {
      setDeleting(false);
    }
  }

  function handleDeleteClick(e: React.MouseEvent) {
    e.preventDefault();
    e.stopPropagation();
    setModalOpen(true);
  }

  return (
    <>
      <DeleteConfirmModal
        isOpen={modalOpen}
        onClose={() => { setModalOpen(false); setDeleteError(null); }}
        onConfirm={handleDelete}
        title="Delete Project"
        description="This will permanently remove the project and all its data. This action cannot be undone."
        itemName={name}
        isDeleting={deleting}
        errorMessage={deleteError ?? undefined}
      />
      <Link
        to={`/environments/${envId}/projects/${id}`}
        className="block bg-surface-container-low rounded-xl p-6 flex flex-col justify-between
                   group hover:bg-surface-container transition-all duration-300
                   border border-transparent hover:border-outline-variant/10 cursor-pointer"
      >
        {/* Top row: status label + name, ring chart + delete */}
        <div className="flex justify-between items-start mb-8">
          <div className="min-w-0 flex-1 pr-4">
            <div className="flex items-center gap-2">
              <span className={`text-[10px] font-label font-bold uppercase tracking-[0.2em] ${cfg.labelColor}`}>
                {cfg.label}
              </span>
              {can('manage') && (
                <button
                  onClick={handleDeleteClick}
                  className="ml-auto p-1 rounded opacity-0 group-hover:opacity-100 focus-visible:opacity-100 hover:bg-error/10 hover:text-error text-on-surface-variant transition-all"
                  title="Delete project"
                  aria-label={`Delete project ${name}`}
                >
                  <span className="material-symbols-outlined text-[14px]" aria-hidden="true">delete</span>
                </button>
              )}
            </div>
            <h3 className="text-lg font-headline font-bold text-on-surface mt-1 truncate">
              {name}
            </h3>
            <p className="text-[11px] font-mono text-on-surface-variant mt-0.5 truncate opacity-60">
              {id}
            </p>
          </div>
          <SegmentRing passed={lastPassed} failed={lastFailed} total={lastTotal} label={ringLabel} />
        </div>

        {/* Bottom: bento stat grid */}
        <div className="grid grid-cols-3 gap-3 mt-auto">
          <div className="bg-surface-container-high rounded-lg p-3">
            <p className="text-[10px] font-label text-on-surface-variant uppercase mb-1 tracking-wide">
              Builds
            </p>
            <p className="text-lg font-headline font-bold text-on-surface">
              {builds.toLocaleString()}
            </p>
          </div>
          <div className="bg-surface-container-high rounded-lg p-3">
            <p className="text-[10px] font-label text-on-surface-variant uppercase mb-1 tracking-wide">
              Tests
            </p>
            <p className="text-lg font-headline font-bold text-on-surface">
              {lastTotal > 0 ? lastTotal.toLocaleString() : "—"}
            </p>
          </div>
          <div className="bg-surface-container-high rounded-lg p-3">
            <p className="text-[10px] font-label text-on-surface-variant uppercase mb-1 tracking-wide">
              Last Run
            </p>
            <p className="text-xs font-label font-semibold text-on-surface-variant leading-tight mt-1">
              {lastRun}
            </p>
          </div>
        </div>
      </Link>
    </>
  );
};

export default React.memo(ProjectCard);
