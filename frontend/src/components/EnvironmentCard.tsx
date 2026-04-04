import React, { useState } from "react";
import { Link } from "react-router-dom";
import { api } from "../api/client";
import DeleteConfirmModal from "./ui/DeleteConfirmModal";
import EditEnvironmentModal from "./EditEnvironmentModal";
import { useAuth } from "../context/AuthContext";

export interface EnvironmentCardProps {
  id: string;
  name: string;
  icon: string;
  projectCount: number;
  createdAt: string;
  onDeleted?: () => void;
  onUpdated?: () => void;
}

const EnvironmentCard: React.FC<EnvironmentCardProps> = ({
  id,
  name,
  icon,
  projectCount,
  createdAt,
  onDeleted,
  onUpdated,
}) => {
  const { can } = useAuth();
  const [deleteModalOpen, setDeleteModalOpen] = useState(false);
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const created = new Date(createdAt).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  });

  async function handleDelete() {
    setDeleting(true);
    setDeleteError(null);
    try {
      await api.deleteEnvironment(id);
      setDeleteModalOpen(false);
      onDeleted?.();
    } catch (e) {
      setDeleteError(
        e instanceof Error ? e.message : "Delete failed. Please try again.",
      );
    } finally {
      setDeleting(false);
    }
  }

  function handleDeleteClick(e: React.MouseEvent) {
    e.preventDefault();
    e.stopPropagation();
    setDeleteModalOpen(true);
  }

  function handleEditClick(e: React.MouseEvent) {
    e.preventDefault();
    e.stopPropagation();
    setEditModalOpen(true);
  }

  return (
    <>
      <DeleteConfirmModal
        isOpen={deleteModalOpen}
        onClose={() => {
          setDeleteModalOpen(false);
          setDeleteError(null);
        }}
        onConfirm={handleDelete}
        title="Delete Environment"
        description="This will permanently remove the environment and all its projects and data. This action cannot be undone."
        itemName={name}
        isDeleting={deleting}
        errorMessage={deleteError ?? undefined}
      />
      <EditEnvironmentModal
        isOpen={editModalOpen}
        onClose={() => setEditModalOpen(false)}
        onUpdated={() => {
          setEditModalOpen(false);
          onUpdated?.();
        }}
        envId={id}
        currentName={name}
        currentIcon={icon}
      />
      <Link
        to={`/environments/${id}`}
        state={{ envName: name }}
        className="group relative bg-surface-container-low border border-outline-variant/10 rounded-2xl p-6 flex flex-col gap-4 hover:border-primary/30 hover:shadow-glow-primary transition-all duration-300"
      >
        {/* Header */}
        <div className="flex items-start justify-between gap-3">
          <div className="flex items-center gap-3 min-w-0">
            <div className="w-10 h-10 rounded-xl bg-primary/10 flex items-center justify-center shrink-0">
              <span className="material-symbols-outlined text-[20px] text-primary">
                {icon || "deployed_code"}
              </span>
            </div>
            <div className="min-w-0">
              <h3 className="text-base font-headline font-bold text-on-surface truncate group-hover:text-primary transition-colors">
                {name}
              </h3>
              <p className="text-[11px] font-label text-on-surface-variant font-mono truncate">
                {id}
              </p>
            </div>
          </div>

          {/* Action buttons - appear on hover or keyboard focus (manage only) */}
          {can("manage") && (
            <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 focus-within:opacity-100 transition-all shrink-0">
              <button
                onClick={handleEditClick}
                className="p-1.5 rounded-lg hover:bg-surface-container-highest text-on-surface-variant hover:text-on-surface transition-colors"
                title="Edit environment"
                aria-label={`Edit environment ${name}`}
              >
                <span
                  className="material-symbols-outlined text-[16px]"
                  aria-hidden="true"
                >
                  edit
                </span>
              </button>
              {/* Delete hidden for the built-in default environment */}
              {id !== "default" && (
                <button
                  onClick={handleDeleteClick}
                  className="p-1.5 rounded-lg hover:bg-error/10 hover:text-error text-on-surface-variant transition-colors"
                  title="Delete environment"
                  aria-label={`Delete environment ${name}`}
                >
                  <span
                    className="material-symbols-outlined text-[16px]"
                    aria-hidden="true"
                  >
                    delete
                  </span>
                </button>
              )}
            </div>
          )}
        </div>

        {/* Stats row */}
        <div className="flex gap-3">
          <div className="flex-1 bg-surface-container-high rounded-lg p-3">
            <p className="text-[10px] font-label text-on-surface-variant uppercase mb-1 tracking-wide">
              Projects
            </p>
            <p className="text-lg font-headline font-bold text-on-surface">
              {projectCount}
            </p>
          </div>
          <div className="flex-1 bg-surface-container-high rounded-lg p-3">
            <p className="text-[10px] font-label text-on-surface-variant uppercase mb-1 tracking-wide">
              Created
            </p>
            <p className="text-xs font-label font-semibold text-on-surface-variant leading-tight mt-1">
              {created}
            </p>
          </div>
        </div>
      </Link>
    </>
  );
};

export default React.memo(EnvironmentCard);
