import React, { useCallback, useEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { api } from "../api/client";
import type {
  APIKey,
  TrackedUser,
  RetentionSettings,
  CleanupRun,
} from "../types";
import { formatDate } from "../utils/format";
import { useAuth } from "../context/AuthContext";
import SearchInput from "../components/ui/SearchInput";
import { useSnackbar } from "../components/ui/Snackbar";

// ── Helpers ───────────────────────────────────────────────────────────────────

const ROLES = ["admin", "developer", "viewer"] as const;
type Role = (typeof ROLES)[number];

const PAGE_SIZE = 20;

function roleBadge(role: string) {
  switch (role) {
    case "admin":
      return "bg-primary/10 text-primary";
    case "developer":
      return "bg-secondary/10 text-secondary";
    default:
      return "bg-surface-container text-on-surface-variant";
  }
}

// ── Plaintext reveal modal ────────────────────────────────────────────────────

function PlaintextModal({
  plaintext,
  onClose,
}: {
  plaintext: string;
  onClose: () => void;
}) {
  const [copied, setCopied] = useState(false);

  function copy() {
    navigator.clipboard.writeText(plaintext).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/40 backdrop-blur-sm"
      onClick={onClose}
    >
      <div
        className="w-full max-w-lg rounded-2xl shadow-2xl border p-6 space-y-4"
        style={{
          background: "rgb(var(--color-surface-container))",
          borderColor: "rgb(var(--color-outline-variant) / 0.4)",
        }}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-start justify-between gap-4">
          <div>
            <h3 className="text-lg font-headline font-bold text-on-surface">
              API Key Created
            </h3>
            <p className="text-sm text-on-surface-variant mt-1">
              Copy this key now - it will{" "}
              <span className="text-error font-semibold">
                never be shown again
              </span>
              .
            </p>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 rounded-lg text-on-surface-variant hover:text-on-surface hover:bg-black/5 dark:hover:bg-white/5 transition-colors"
          >
            <span className="material-symbols-outlined text-[18px]">close</span>
          </button>
        </div>
        <div className="flex items-center gap-2 bg-surface-container-lowest rounded-xl px-4 py-3 border border-outline-variant/20">
          <code className="flex-1 text-sm font-mono text-on-surface break-all select-all">
            {plaintext}
          </code>
          <button
            onClick={copy}
            className="shrink-0 flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-primary/10 text-primary text-xs font-label font-semibold hover:bg-primary/20 transition-colors"
          >
            <span className="material-symbols-outlined text-[14px]">
              {copied ? "check" : "content_copy"}
            </span>
            {copied ? "Copied" : "Copy"}
          </button>
        </div>
        <p className="text-xs text-on-surface-variant">
          Use this key in the{" "}
          <code className="font-mono bg-surface-container-high px-1 rounded">
            Authorization: Bearer &lt;key&gt;
          </code>{" "}
          header.
        </p>
        <button
          onClick={onClose}
          className="w-full py-2.5 rounded-xl bg-primary text-on-primary text-sm font-headline font-bold hover:brightness-110 active:scale-95 transition-all"
        >
          Done
        </button>
      </div>
    </div>
  );
}

// ── Role dropdown ─────────────────────────────────────────────────────────────

function RoleDropdown({
  value,
  onChange,
}: {
  value: Role;
  onChange: (r: Role) => void;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    function handleOutside(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node))
        setOpen(false);
    }
    document.addEventListener("mousedown", handleOutside);
    return () => document.removeEventListener("mousedown", handleOutside);
  }, [open]);

  const ROLE_COLOR: Record<Role, string> = {
    admin: "text-primary",
    developer: "text-secondary",
    viewer: "text-on-surface-variant",
  };

  return (
    <div className="relative" ref={ref}>
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="flex items-center gap-2 bg-surface-container-low border border-outline-variant/30 rounded-lg px-3 py-2 text-sm text-on-surface outline-none focus:ring-1 focus:ring-primary/40 capitalize min-w-[120px] justify-between"
      >
        <span className={`font-semibold ${ROLE_COLOR[value]}`}>{value}</span>
        <span
          className="material-symbols-outlined text-[16px] text-on-surface-variant"
          style={{
            transform: open ? "rotate(180deg)" : undefined,
            transition: "transform 0.15s",
          }}
        >
          expand_more
        </span>
      </button>
      {open && (
        <div
          className="absolute left-0 top-full mt-1 z-50 rounded-xl border shadow-lg py-1 min-w-full overflow-hidden"
          style={{
            background: "rgb(var(--color-surface-container))",
            borderColor: "rgb(var(--color-outline-variant) / 0.4)",
          }}
        >
          {ROLES.map((r) => (
            <button
              key={r}
              type="button"
              onClick={() => {
                onChange(r);
                setOpen(false);
              }}
              className={`w-full flex items-center gap-2 px-4 py-2 text-sm capitalize transition-colors hover:bg-black/5 dark:hover:bg-white/5 ${
                r === value
                  ? "font-semibold"
                  : "font-normal text-on-surface-variant"
              } ${ROLE_COLOR[r]}`}
            >
              {r === value && (
                <span className="material-symbols-outlined text-[14px]">
                  check
                </span>
              )}
              {r !== value && <span className="w-[14px]" />}
              {r}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

// ── Create key form ───────────────────────────────────────────────────────────

function CreateKeyForm({
  onCreated,
}: {
  onCreated: (plaintext: string) => void;
}) {
  const [name, setName] = useState("");
  const [role, setRole] = useState<Role>("developer");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) return;
    setSubmitting(true);
    setError(null);
    try {
      const result = await api.createAPIKey(name.trim(), role);
      setName("");
      setRole("developer");
      onCreated(result.plaintext);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create API key");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="flex items-end gap-3 flex-wrap">
      <div className="flex-1 min-w-[180px]">
        <label className="block text-[10px] font-label font-bold uppercase tracking-widest text-on-surface-variant mb-1.5">
          Key Name
        </label>
        <input
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="e.g. ci-pipeline"
          required
          className="w-full bg-surface-container-low border border-outline-variant/30 rounded-lg px-3 py-2 text-sm font-mono text-on-surface outline-none focus:ring-1 focus:ring-primary/40 placeholder:text-on-surface-variant/40"
        />
      </div>
      <div>
        <label className="block text-[10px] font-label font-bold uppercase tracking-widest text-on-surface-variant mb-1.5">
          Role
        </label>
        <RoleDropdown value={role} onChange={setRole} />
      </div>
      <button
        type="submit"
        disabled={submitting || !name.trim()}
        className="flex items-center gap-2 bg-primary text-on-primary px-4 py-2 rounded-lg text-sm font-headline font-bold hover:brightness-110 active:scale-95 transition-all disabled:opacity-50"
      >
        <span className="material-symbols-outlined text-[16px]">add</span>
        {submitting ? "Creating…" : "Create Key"}
      </button>
      {error && <p className="w-full text-xs text-error">{error}</p>}
    </form>
  );
}

// ── Load more button ──────────────────────────────────────────────────────────

function LoadMoreButton({
  remaining,
  loading,
  onClick,
}: {
  remaining: number;
  loading: boolean;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      disabled={loading}
      className="w-full py-2.5 rounded-xl border border-outline-variant/20 text-sm text-on-surface-variant hover:text-on-surface hover:bg-black/5 dark:hover:bg-white/5 transition-colors disabled:opacity-50"
    >
      {loading ? "Loading…" : `Load more (${remaining} remaining)`}
    </button>
  );
}

// ── API Keys section ──────────────────────────────────────────────────────────

function APIKeysSection() {
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [plaintext, setPlaintext] = useState<string | null>(null);
  const [actionId, setActionId] = useState<string | null>(null);
  const { show: showAlert, SnackbarNode } = useSnackbar();

  const loadPage = useCallback((q: string, off: number, append: boolean) => {
    if (append) setLoadingMore(true);
    else {
      setLoading(true);
      setError(null);
    }
    api
      .listAPIKeys(q, off)
      .then((data) => {
        setKeys((prev) =>
          append ? [...prev, ...(data.keys ?? [])] : (data.keys ?? []),
        );
        setTotal(data.total);
        setOffset(off);
      })
      .catch((e) =>
        setError(e instanceof Error ? e.message : "Failed to load keys"),
      )
      .finally(() => {
        setLoading(false);
        setLoadingMore(false);
      });
  }, []);

  const isFirst = useRef(true);
  useEffect(() => {
    if (isFirst.current) {
      isFirst.current = false;
      loadPage("", 0, false);
      return;
    }
    const t = setTimeout(() => loadPage(search, 0, false), 300);
    return () => clearTimeout(t);
  }, [search, loadPage]);

  async function handleRevoke(id: string) {
    setActionId(id);
    try {
      await api.revokeAPIKey(id);
      setKeys((prev) =>
        prev.map((k) => (k.id === id ? { ...k, isActive: false } : k)),
      );
      showAlert("API key revoked successfully");
    } catch (e) {
      showAlert(
        e instanceof Error ? e.message : "Failed to revoke key",
        "error",
      );
    } finally {
      setActionId(null);
    }
  }

  async function handleDelete(id: string) {
    setActionId(id);
    try {
      await api.deleteAPIKey(id);
      setKeys((prev) => prev.filter((k) => k.id !== id));
      setTotal((t) => t - 1);
      showAlert("API key deleted");
    } catch (e) {
      showAlert(
        e instanceof Error ? e.message : "Failed to delete key",
        "error",
      );
    } finally {
      setActionId(null);
    }
  }

  function handleCreated(pt: string) {
    setPlaintext(pt);
    loadPage(search, 0, false);
  }

  const hasMore = keys.length < total;

  return (
    <div className="flex flex-col gap-8 h-full">
      {/* Role reference */}
      <section className="shrink-0">
        <div className="flex items-center gap-2 mb-3">
          <span className="material-symbols-outlined text-[18px] text-on-surface-variant">
            shield
          </span>
          <h3 className="text-sm font-headline font-bold text-on-surface">
            Role Permissions
          </h3>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-3 xl:grid-cols-3 gap-3">
          {(
            [
              {
                role: "admin",
                icon: "admin_panel_settings",
                color: "text-primary",
                bg: "bg-primary/5 border-primary/15",
                desc: "Full access - manage environments, projects, keys, and users.",
              },
              {
                role: "developer",
                icon: "code",
                color: "text-secondary",
                bg: "bg-secondary/5 border-secondary/15",
                desc: "Upload results and generate reports. No management access.",
              },
              {
                role: "viewer",
                icon: "visibility",
                color: "text-on-surface-variant",
                bg: "bg-surface-container-low border-outline-variant/10",
                desc: "Read-only access to environments and reports.",
              },
            ] as const
          ).map(({ role, icon, color, bg, desc }) => (
            <div key={role} className={`rounded-xl border p-4 ${bg}`}>
              <div className="flex items-center gap-2 mb-1.5">
                <span
                  className={`material-symbols-outlined text-[16px] ${color}`}
                >
                  {icon}
                </span>
                <span
                  className={`text-xs font-label font-bold capitalize ${color}`}
                >
                  {role}
                </span>
              </div>
              <p className="text-[11px] text-on-surface-variant leading-relaxed">
                {desc}
              </p>
            </div>
          ))}
        </div>
      </section>

      {/* Generate card */}
      <section className="shrink-0">
        <div className="flex items-center gap-2 mb-3">
          <span className="material-symbols-outlined text-[18px] text-primary">
            add_circle
          </span>
          <h3 className="text-sm font-headline font-bold text-on-surface">
            Generate New Key
          </h3>
        </div>
        <div className="bg-surface-container-low rounded-2xl p-5 border border-outline-variant/10">
          <p className="text-xs text-on-surface-variant mb-4">
            Keys are shown{" "}
            <span className="font-semibold text-on-surface">once</span> at
            creation. Store them securely - they cannot be recovered.
          </p>
          <CreateKeyForm onCreated={handleCreated} />
        </div>
      </section>

      {/* Manage keys card */}
      <section className="flex flex-col min-h-0 flex-1">
        <div className="flex items-center justify-between gap-4 mb-3 shrink-0">
          <div className="flex items-center gap-2">
            <span className="material-symbols-outlined text-[18px] text-on-surface-variant">
              vpn_key
            </span>
            <h3 className="text-sm font-headline font-bold text-on-surface">
              Active Keys
              {!loading && (
                <span className="ml-2 font-normal text-on-surface-variant">
                  ({total})
                </span>
              )}
            </h3>
          </div>
          <SearchInput
            value={search}
            onValueChange={setSearch}
            placeholder="Search by name or creator…"
          />
        </div>

        {error && (
          <div className="flex items-center gap-2 text-xs text-error bg-error/8 rounded-xl px-4 py-2.5 mb-3 border border-error/15 shrink-0">
            <span className="material-symbols-outlined text-[14px]">error</span>
            {error}
          </div>
        )}

        <div className="flex-1 min-h-0 overflow-y-auto">
          {loading ? (
            <div className="flex items-center justify-center py-14 bg-surface-container-low/40 rounded-2xl border border-outline-variant/10">
              <span className="text-on-surface-variant text-sm animate-pulse">
                Loading keys…
              </span>
            </div>
          ) : keys.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-14 bg-surface-container-low/30 rounded-2xl border border-dashed border-outline-variant/20">
              <span className="material-symbols-outlined text-[40px] text-on-surface-variant/20 mb-3">
                key_off
              </span>
              <p className="text-sm font-headline font-bold text-on-surface">
                {search ? "No keys match your search" : "No API keys yet"}
              </p>
              <p className="text-xs text-on-surface-variant mt-1">
                {search
                  ? "Try a different search term."
                  : "Generate a key above to enable CI/CD access."}
              </p>
            </div>
          ) : (
            <div className="space-y-2">
              {keys.map((k) => (
                <div
                  key={k.id}
                  className={`flex items-center gap-4 px-4 py-3.5 rounded-xl border transition-all ${
                    k.isActive
                      ? "bg-surface-container-low border-outline-variant/10"
                      : "bg-surface-container-lowest border-outline-variant/5 opacity-50"
                  }`}
                >
                  <span
                    className={`w-2 h-2 rounded-full shrink-0 ${k.isActive ? "bg-emerald-500" : "bg-on-surface-variant/30"}`}
                  />
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <span className="text-sm font-headline font-semibold text-on-surface">
                        {k.name}
                      </span>
                      <span
                        className={`text-[10px] font-label font-bold uppercase tracking-wider px-1.5 py-0.5 rounded-full ${roleBadge(k.role)}`}
                      >
                        {k.role}
                      </span>
                      {!k.isActive && (
                        <span className="text-[10px] font-label font-bold uppercase tracking-wider px-1.5 py-0.5 rounded-full bg-error/10 text-error">
                          Revoked
                        </span>
                      )}
                    </div>
                    <div className="flex items-center gap-3 mt-0.5 text-[11px] text-on-surface-variant font-label flex-wrap">
                      <span>Created by {k.createdBy}</span>
                      <span className="opacity-40">·</span>
                      <span>{formatDate(k.createdAt)}</span>
                      {k.lastUsedAt && (
                        <>
                          <span className="opacity-40">·</span>
                          <span>Last used {formatDate(k.lastUsedAt)}</span>
                        </>
                      )}
                    </div>
                  </div>
                  <div className="flex items-center gap-1 shrink-0">
                    {k.isActive && (
                      <button
                        onClick={() => handleRevoke(k.id)}
                        disabled={actionId === k.id}
                        className="px-3 py-1.5 rounded-lg text-xs font-label font-semibold text-on-surface-variant border border-outline-variant/20 hover:border-error/30 hover:text-error hover:bg-error/5 transition-colors disabled:opacity-50"
                      >
                        {actionId === k.id ? "Revoking…" : "Revoke"}
                      </button>
                    )}
                    <button
                      onClick={() => handleDelete(k.id)}
                      disabled={actionId === k.id}
                      className="p-1.5 rounded-lg text-on-surface-variant/40 hover:text-error hover:bg-error/10 transition-colors disabled:opacity-50"
                      title="Delete permanently"
                      aria-label={`Delete key ${k.name}`}
                    >
                      <span className="material-symbols-outlined text-[16px]">
                        delete
                      </span>
                    </button>
                  </div>
                </div>
              ))}
              {hasMore && (
                <LoadMoreButton
                  remaining={total - keys.length}
                  loading={loadingMore}
                  onClick={() => loadPage(search, offset + PAGE_SIZE, true)}
                />
              )}
            </div>
          )}
        </div>
      </section>

      {plaintext && (
        <PlaintextModal
          plaintext={plaintext}
          onClose={() => setPlaintext(null)}
        />
      )}
      {SnackbarNode}
    </div>
  );
}

// ── Users section ─────────────────────────────────────────────────────────────

function UsersSection() {
  const { user: me } = useAuth();
  const [users, setUsers] = useState<TrackedUser[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [savingEmail, setSavingEmail] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const { show: showAlert, SnackbarNode } = useSnackbar();

  const isAdmin = me?.role === "admin";

  const loadPage = useCallback((q: string, off: number, append: boolean) => {
    if (append) setLoadingMore(true);
    else {
      setLoading(true);
      setError(null);
    }
    api
      .listUsers(q, off)
      .then((data) => {
        setUsers((prev) =>
          append ? [...prev, ...(data.users ?? [])] : (data.users ?? []),
        );
        setTotal(data.total);
        setOffset(off);
      })
      .catch((e) =>
        setError(e instanceof Error ? e.message : "Failed to load users"),
      )
      .finally(() => {
        setLoading(false);
        setLoadingMore(false);
      });
  }, []);

  const isFirst = useRef(true);
  useEffect(() => {
    if (isFirst.current) {
      isFirst.current = false;
      loadPage("", 0, false);
      return;
    }
    const t = setTimeout(() => loadPage(search, 0, false), 300);
    return () => clearTimeout(t);
  }, [search, loadPage]);

  async function handleSetRole(email: string, role: Role) {
    setSavingEmail(email);
    setActionError(null);
    try {
      await api.setUserRole(email, role);
      setUsers((prev) =>
        prev.map((u) => (u.email === email ? { ...u, role } : u)),
      );
      showAlert(`Role updated to ${role}`);
    } catch (e) {
      showAlert(
        e instanceof Error ? e.message : "Failed to update role",
        "error",
      );
    } finally {
      setSavingEmail(null);
    }
  }

  async function handleResetRole(email: string) {
    setSavingEmail(email);
    setActionError(null);
    try {
      await api.resetUserRole(email);
      const data = await api.listUsers(search, 0);
      setUsers(data.users ?? []);
      setTotal(data.total);
      setOffset(0);
      showAlert("Role reset to baseline");
    } catch (e) {
      showAlert(
        e instanceof Error ? e.message : "Failed to reset role",
        "error",
      );
    } finally {
      setSavingEmail(null);
    }
  }

  const hasMore = users.length < total;

  return (
    <div className="space-y-6">
      {/* Info callout */}
      <div className="flex items-start gap-3 bg-surface-container-low rounded-xl px-4 py-3.5 border border-outline-variant/10">
        <span className="material-symbols-outlined text-[18px] text-on-surface-variant mt-0.5 shrink-0">
          info
        </span>
        <p className="text-xs text-on-surface-variant leading-relaxed">
          Users appear here after their first login via OAuth. Role changes take
          effect on their{" "}
          <span className="font-semibold text-on-surface">next login</span>.
          {!isAdmin && (
            <span className="ml-1">Only admins can change roles.</span>
          )}
        </p>
      </div>

      {/* List header */}
      <div className="flex items-center justify-between gap-4 flex-wrap">
        <span className="text-sm font-headline font-bold text-on-surface">
          Users
          {!loading && (
            <span className="ml-2 font-normal text-on-surface-variant">
              ({total})
            </span>
          )}
        </span>
        <SearchInput
          value={search}
          onValueChange={setSearch}
          placeholder="Search by name or email…"
        />
      </div>

      {error && (
        <div className="flex items-center gap-2 text-xs text-error bg-error/8 rounded-xl px-4 py-2.5 border border-error/15">
          <span className="material-symbols-outlined text-[14px]">error</span>
          {error}
        </div>
      )}
      {actionError && (
        <div className="flex items-center gap-2 text-xs text-error bg-error/8 rounded-xl px-4 py-2.5 border border-error/15">
          <span className="material-symbols-outlined text-[14px]">error</span>
          {actionError}
        </div>
      )}

      {loading ? (
        <div className="flex items-center justify-center py-14 bg-surface-container-low/40 rounded-2xl border border-outline-variant/10">
          <span className="text-on-surface-variant text-sm animate-pulse">
            Loading users…
          </span>
        </div>
      ) : users.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-14 bg-surface-container-low/30 rounded-2xl border border-dashed border-outline-variant/20">
          <span className="material-symbols-outlined text-[40px] text-on-surface-variant/20 mb-3">
            person_off
          </span>
          <p className="text-sm font-headline font-bold text-on-surface">
            {search ? "No users match your search" : "No users yet"}
          </p>
          <p className="text-xs text-on-surface-variant mt-1">
            {search
              ? "Try a different search term."
              : "Users appear here after their first login."}
          </p>
        </div>
      ) : (
        <div className="space-y-2">
          {users.map((u) => {
            const isSelf = u.email === me?.email;
            const canEdit = isAdmin && !isSelf;
            const saving = savingEmail === u.email;

            return (
              <div
                key={u.email}
                className="flex items-center gap-4 px-4 py-3.5 rounded-xl bg-surface-container-low border border-outline-variant/10"
              >
                {/* Avatar */}
                {u.avatarUrl ? (
                  <img
                    src={u.avatarUrl}
                    alt={u.name}
                    className="w-9 h-9 rounded-full shrink-0 ring-2 ring-outline-variant/20"
                    referrerPolicy="no-referrer"
                  />
                ) : (
                  <span className="w-9 h-9 rounded-full bg-primary/15 flex items-center justify-center text-primary text-sm font-bold font-headline shrink-0">
                    {u.name?.[0]?.toUpperCase() ?? u.email[0].toUpperCase()}
                  </span>
                )}

                {/* Info */}
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="text-sm font-headline font-semibold text-on-surface">
                      {u.name || u.email}
                    </span>
                    {isSelf && (
                      <span className="text-[10px] font-label text-on-surface-variant/60 bg-surface-container px-1.5 py-0.5 rounded">
                        you
                      </span>
                    )}
                    <span className="text-[10px] font-label text-on-surface-variant/50 bg-surface-container px-1.5 py-0.5 rounded capitalize">
                      {u.provider}
                    </span>
                  </div>
                  <p className="text-[11px] text-on-surface-variant font-label mt-0.5 truncate lowercase">
                    {u.email}
                  </p>
                </div>

                {/* Role control */}
                <div className="flex items-center gap-2 shrink-0">
                  {saving ? (
                    <span className="text-xs text-on-surface-variant animate-pulse px-2">
                      Saving…
                    </span>
                  ) : canEdit ? (
                    <>
                      <RoleDropdown
                        value={u.role as Role}
                        onChange={(role) => handleSetRole(u.email, role)}
                      />
                      <button
                        onClick={() => handleResetRole(u.email)}
                        className="p-1.5 rounded-lg text-on-surface-variant/40 hover:text-on-surface-variant hover:bg-black/5 dark:hover:bg-white/5 transition-colors"
                        title="Reset to policy.yaml baseline"
                        aria-label={`Reset ${u.email} to baseline role`}
                      >
                        <span className="material-symbols-outlined text-[16px]">
                          restart_alt
                        </span>
                      </button>
                    </>
                  ) : (
                    <span
                      className={`text-[10px] font-label font-bold uppercase tracking-wider px-1.5 py-0.5 rounded-full ${roleBadge(u.role)}`}
                    >
                      {u.role}
                    </span>
                  )}
                </div>

                {/* Timestamps */}
                <div className="text-right shrink-0 hidden lg:block">
                  <p className="text-[11px] text-on-surface-variant font-label">
                    Last login: {formatDate(u.lastLoginAt)}
                  </p>
                  <p className="text-[10px] text-on-surface-variant/50 font-label mt-0.5">
                    First: {formatDate(u.firstLoginAt)}
                  </p>
                </div>
              </div>
            );
          })}

          {hasMore && (
            <LoadMoreButton
              remaining={total - users.length}
              loading={loadingMore}
              onClick={() => loadPage(search, offset + PAGE_SIZE, true)}
            />
          )}
        </div>
      )}
      {SnackbarNode}
    </div>
  );
}

// ── Data Retention section ────────────────────────────────────────────────────

function formatDuration(startedAt: string, finishedAt: string): string {
  const ms = new Date(finishedAt).getTime() - new Date(startedAt).getTime();
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  return `${Math.floor(ms / 60000)}m ${Math.round((ms % 60000) / 1000)}s`;
}

function DataRetentionSection() {
  const [form, setForm] = useState<RetentionSettings>({
    retentionDays: 90,
    intervalHours: 6,
    dryRun: false,
  });
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [runs, setRuns] = useState<CleanupRun[]>([]);
  const [runsLoading, setRunsLoading] = useState(true);
  const { show: showAlert, SnackbarNode } = useSnackbar();

  useEffect(() => {
    api
      .getRetentionSettings()
      .then((data) => setForm(data))
      .catch((e) =>
        setError(
          e instanceof Error ? e.message : "Failed to load retention settings",
        ),
      )
      .finally(() => setLoading(false));
    api
      .getCleanupRuns(5)
      .then((data) => setRuns(data))
      .finally(() => setRunsLoading(false));
  }, []);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    setError(null);
    try {
      await api.setRetentionSettings(form);
      showAlert("Retention settings saved");
    } catch (err) {
      showAlert(
        err instanceof Error
          ? err.message
          : "Failed to save retention settings",
        "error",
      );
    } finally {
      setSaving(false);
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-14 bg-surface-container-low/40 rounded-2xl border border-outline-variant/10">
        <span className="text-on-surface-variant text-sm animate-pulse">
          Loading retention settings…
        </span>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Active policy summary */}
      <div
        className={`rounded-xl border px-5 py-4 flex items-start gap-4 ${
          form.dryRun
            ? "bg-amber-500/5 border-amber-500/20"
            : "bg-primary/5 border-primary/15"
        }`}
      >
        <span
          className={`material-symbols-outlined text-[22px] mt-0.5 shrink-0 ${form.dryRun ? "text-amber-500" : "text-primary"}`}
        >
          {form.dryRun ? "warning" : "auto_delete"}
        </span>
        <div>
          <p
            className={`text-sm font-headline font-semibold ${form.dryRun ? "text-amber-600 dark:text-amber-400" : "text-on-surface"}`}
          >
            {form.dryRun ? "Dry run mode is enabled" : "Active cleanup policy"}
          </p>
          <p className="text-xs text-on-surface-variant mt-1 leading-relaxed">
            {form.dryRun
              ? "No data will be deleted. The cleanup worker will log what it would remove but take no action."
              : `Reports older than ${form.retentionDays} day${form.retentionDays !== 1 ? "s" : ""} are permanently deleted. The cleanup worker runs every ${form.intervalHours} hour${form.intervalHours !== 1 ? "s" : ""}.`}
          </p>
        </div>
      </div>

      {/* Settings form */}
      <form onSubmit={handleSubmit} className="space-y-0">
        <div className="bg-surface-container-low rounded-2xl border border-outline-variant/10 overflow-hidden divide-y divide-outline-variant/10">
          {/* Retention Period */}
          <div className="px-5 py-4 flex items-center justify-between gap-6">
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 mb-0.5">
                <span className="material-symbols-outlined text-[16px] text-on-surface-variant">
                  history
                </span>
                <p className="text-sm font-headline font-semibold text-on-surface">
                  Retention Period
                </p>
              </div>
              <p className="text-xs text-on-surface-variant">
                Reports older than this are permanently deleted during each
                cleanup sweep.
              </p>
            </div>
            <div className="flex items-center gap-2 shrink-0">
              <input
                type="number"
                min={1}
                max={3650}
                required
                value={form.retentionDays}
                onChange={(e) =>
                  setForm((f) => ({
                    ...f,
                    retentionDays: Math.max(1, parseInt(e.target.value) || 1),
                  }))
                }
                className="w-20 bg-surface-container-lowest border border-outline-variant/30 rounded-lg px-3 py-2 text-sm font-mono text-on-surface text-center outline-none focus:ring-1 focus:ring-primary/40"
              />
              <span className="text-sm text-on-surface-variant w-8">days</span>
            </div>
          </div>

          {/* Cleanup Interval */}
          <div className="px-5 py-4 flex items-center justify-between gap-6">
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 mb-0.5">
                <span className="material-symbols-outlined text-[16px] text-on-surface-variant">
                  schedule
                </span>
                <p className="text-sm font-headline font-semibold text-on-surface">
                  Cleanup Interval
                </p>
              </div>
              <p className="text-xs text-on-surface-variant">
                How frequently the cleanup worker checks for expired reports.
              </p>
            </div>
            <div className="flex items-center gap-2 shrink-0">
              <input
                type="number"
                min={1}
                max={720}
                required
                value={form.intervalHours}
                onChange={(e) =>
                  setForm((f) => ({
                    ...f,
                    intervalHours: Math.max(1, parseInt(e.target.value) || 1),
                  }))
                }
                className="w-20 bg-surface-container-lowest border border-outline-variant/30 rounded-lg px-3 py-2 text-sm font-mono text-on-surface text-center outline-none focus:ring-1 focus:ring-primary/40"
              />
              <span className="text-sm text-on-surface-variant w-8">hrs</span>
            </div>
          </div>

          {/* Dry Run */}
          <div className="px-5 py-4 flex items-center justify-between gap-6">
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 mb-0.5">
                <span className="material-symbols-outlined text-[16px] text-on-surface-variant">
                  science
                </span>
                <p className="text-sm font-headline font-semibold text-on-surface">
                  Dry Run Mode
                </p>
              </div>
              <p className="text-xs text-on-surface-variant">
                Log what would be deleted without removing anything. Safe way to
                verify your settings.
              </p>
            </div>
            <button
              type="button"
              role="switch"
              aria-checked={form.dryRun}
              onClick={() => setForm((f) => ({ ...f, dryRun: !f.dryRun }))}
              className={`relative shrink-0 w-11 h-6 rounded-full transition-colors focus:outline-none focus:ring-2 focus:ring-primary/40 ${
                form.dryRun ? "bg-amber-500" : "bg-outline/40"
              }`}
            >
              <span
                className={`absolute top-0.5 left-0.5 w-5 h-5 rounded-full bg-white shadow-sm transition-transform ${
                  form.dryRun ? "translate-x-5" : "translate-x-0"
                }`}
              />
            </button>
          </div>
        </div>

        {error && (
          <div className="flex items-center gap-2 text-xs text-error bg-error/8 rounded-xl px-4 py-2.5 mt-4 border border-error/15">
            <span className="material-symbols-outlined text-[14px]">error</span>
            {error}
          </div>
        )}

        <div className="flex items-center gap-3 pt-4">
          <button
            type="submit"
            disabled={saving}
            className="flex items-center gap-2 bg-primary text-on-primary px-5 py-2 rounded-lg text-sm font-headline font-bold hover:brightness-110 active:scale-95 transition-all disabled:opacity-50"
          >
            <span className="material-symbols-outlined text-[16px]">
              {saving ? "hourglass_empty" : "save"}
            </span>
            {saving ? "Saving…" : "Save Changes"}
          </button>
        </div>
      </form>

      {/* Recent Cleanup Runs */}
      <section>
        <div className="flex items-center gap-2 mb-3">
          <span className="material-symbols-outlined text-[18px] text-on-surface-variant">
            history
          </span>
          <h3 className="text-sm font-headline font-bold text-on-surface">
            Recent Cleanup Runs
          </h3>
          <span className="text-xs text-on-surface-variant/60 font-label">
            last 5
          </span>
        </div>

        {runsLoading ? (
          <div className="flex items-center justify-center py-10 bg-surface-container-low/40 rounded-2xl border border-outline-variant/10">
            <span className="text-on-surface-variant text-sm animate-pulse">
              Loading runs…
            </span>
          </div>
        ) : runs.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-10 bg-surface-container-low/30 rounded-2xl border border-dashed border-outline-variant/20">
            <span className="material-symbols-outlined text-[36px] text-on-surface-variant/20 mb-2">
              schedule
            </span>
            <p className="text-sm font-headline font-bold text-on-surface">
              No runs yet
            </p>
            <p className="text-xs text-on-surface-variant mt-1">
              Cleanup runs will appear here after the first sweep.
            </p>
          </div>
        ) : (
          <div className="space-y-2">
            {runs.map((run) => (
              <div
                key={run.id}
                className={`rounded-xl border px-4 py-3.5 ${
                  run.status === "failed"
                    ? "bg-error/5 border-error/15"
                    : run.dryRun
                      ? "bg-amber-500/5 border-amber-500/15"
                      : "bg-surface-container-low border-outline-variant/10"
                }`}
              >
                <div className="flex items-center gap-3 flex-wrap">
                  {/* Status icon */}
                  <span
                    className={`material-symbols-outlined text-[18px] shrink-0 ${
                      run.status === "failed"
                        ? "text-error"
                        : "text-emerald-500"
                    }`}
                  >
                    {run.status === "failed" ? "cancel" : "check_circle"}
                  </span>

                  {/* Status + dry-run badge */}
                  <div className="flex items-center gap-2 flex-1 min-w-0">
                    <span
                      className={`text-sm font-headline font-semibold capitalize ${
                        run.status === "failed"
                          ? "text-error"
                          : "text-on-surface"
                      }`}
                    >
                      {run.status}
                    </span>
                    {run.dryRun && (
                      <span className="text-[10px] font-label font-bold uppercase tracking-wider px-1.5 py-0.5 rounded-full bg-amber-500/10 text-amber-600 dark:text-amber-400">
                        dry run
                      </span>
                    )}
                  </div>

                  {/* Stats */}
                  {run.status !== "failed" && (
                    <div className="flex items-center gap-3 text-[11px] text-on-surface-variant font-label shrink-0">
                      <span className="flex items-center gap-1">
                        <span className="material-symbols-outlined text-[12px]">
                          delete
                        </span>
                        {run.deletedCount} deleted
                      </span>
                      {run.skippedCount > 0 && (
                        <span className="flex items-center gap-1 text-amber-600 dark:text-amber-400">
                          <span className="material-symbols-outlined text-[12px]">
                            warning
                          </span>
                          {run.skippedCount} skipped
                        </span>
                      )}
                    </div>
                  )}

                  {/* Duration */}
                  <span className="text-[11px] text-on-surface-variant/60 font-label shrink-0">
                    {formatDuration(run.startedAt, run.finishedAt)}
                  </span>

                  {/* Timestamp */}
                  <span className="text-[11px] text-on-surface-variant font-label shrink-0">
                    {formatDate(run.startedAt)}
                  </span>
                </div>

                {/* Error message */}
                {run.status === "failed" && run.errorMessage && (
                  <p className="mt-2 text-xs text-error/80 font-mono bg-error/5 rounded-lg px-3 py-2 border border-error/10 break-all">
                    {run.errorMessage}
                  </p>
                )}
              </div>
            ))}
          </div>
        )}
      </section>

      {SnackbarNode}
    </div>
  );
}

// ── Allure Version section ────────────────────────────────────────────────────

function AllureVersionSection() {
  const { user: me } = useAuth();
  const isAdmin = me?.role === "admin";
  const [currentVersion, setCurrentVersion] = useState<string | null>(null);
  const [latestVersion, setLatestVersion] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [inputVersion, setInputVersion] = useState("");
  const [updating, setUpdating] = useState(false);
  const { show: showAlert, SnackbarNode } = useSnackbar();

  const updateAvailable = !!(
    latestVersion &&
    currentVersion &&
    latestVersion !== currentVersion
  );

  useEffect(() => {
    api
      .getAllureVersion()
      .then((d) => {
        setCurrentVersion(d.version);
        setLatestVersion(d.latest || null);
      })
      .catch(() => setCurrentVersion(null))
      .finally(() => setLoading(false));
  }, []);

  async function handleUpdate(e: React.FormEvent) {
    e.preventDefault();
    if (!inputVersion.trim()) return;
    setUpdating(true);
    try {
      const result = await api.updateAllureVersion(inputVersion.trim());
      setCurrentVersion(result.version);
      setInputVersion("");
      showAlert(`Allure updated to ${result.version}`);
    } catch (err) {
      showAlert(
        err instanceof Error ? err.message : "Failed to update Allure version",
        "error",
      );
    } finally {
      setUpdating(false);
    }
  }

  async function handleInstallLatest() {
    if (!latestVersion) return;
    setUpdating(true);
    try {
      const result = await api.updateAllureVersion(latestVersion);
      setCurrentVersion(result.version);
      showAlert(`Allure updated to ${result.version}`);
    } catch (err) {
      showAlert(
        err instanceof Error ? err.message : "Failed to update Allure version",
        "error",
      );
    } finally {
      setUpdating(false);
    }
  }

  return (
    <div className="space-y-6">
      {/* Update available banner */}
      {updateAvailable && (
        <div className="rounded-xl border border-amber-500/25 bg-amber-500/8 px-5 py-4 flex items-start gap-4">
          <span className="material-symbols-outlined text-[22px] text-amber-500 mt-0.5 shrink-0">
            new_releases
          </span>
          <div className="flex-1 min-w-0">
            <p className="text-sm font-headline font-semibold text-amber-600 dark:text-amber-400">
              Update available - {latestVersion}
            </p>
            <p className="text-xs text-on-surface-variant mt-0.5">
              Installed: <span className="font-mono">{currentVersion}</span>
              {" · "}Latest: <span className="font-mono">{latestVersion}</span>
            </p>
          </div>
          {isAdmin && (
            <button
              onClick={handleInstallLatest}
              disabled={updating}
              className="shrink-0 flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-amber-500 text-white text-xs font-headline font-bold hover:brightness-110 active:scale-95 transition-all disabled:opacity-50"
            >
              <span className="material-symbols-outlined text-[14px]">
                {updating ? "hourglass_empty" : "download"}
              </span>
              {updating ? "Installing…" : `Install ${latestVersion}`}
            </button>
          )}
        </div>
      )}

      {/* Current version */}
      <div className="bg-surface-container-low rounded-2xl border border-outline-variant/10 overflow-hidden divide-y divide-outline-variant/10">
        <div className="px-5 py-4 flex items-center justify-between gap-6">
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 mb-0.5">
              <span className="material-symbols-outlined text-[16px] text-on-surface-variant">
                system_update
              </span>
              <p className="text-sm font-headline font-semibold text-on-surface">
                Installed Version
              </p>
            </div>
            <p className="text-xs text-on-surface-variant">
              The Allure CLI version currently used to generate reports.
            </p>
          </div>
          <div className="shrink-0 flex items-center gap-2">
            {loading ? (
              <span className="text-sm font-mono text-on-surface-variant animate-pulse">
                checking…
              </span>
            ) : currentVersion ? (
              <span className="text-sm font-mono font-semibold text-on-surface bg-surface-container-high px-3 py-1 rounded-lg border border-outline-variant/20">
                {currentVersion}
              </span>
            ) : (
              <span className="text-sm text-error">unavailable</span>
            )}
            {latestVersion && !updateAvailable && !loading && (
              <span className="flex items-center gap-1 text-[11px] text-emerald-600 dark:text-emerald-400 font-label font-semibold">
                <span className="material-symbols-outlined text-[14px]">
                  check_circle
                </span>
                Up to date
              </span>
            )}
          </div>
        </div>
      </div>

      {/* Update form - admin only */}
      {isAdmin ? (
        <section>
          <div className="flex items-center gap-2 mb-3">
            <span className="material-symbols-outlined text-[18px] text-primary">
              upgrade
            </span>
            <h3 className="text-sm font-headline font-bold text-on-surface">
              Install a Different Version
            </h3>
          </div>
          <div className="bg-surface-container-low rounded-2xl p-5 border border-outline-variant/10">
            <p className="text-xs text-on-surface-variant mb-4">
              Enter a semver version (e.g.{" "}
              <code className="font-mono bg-surface-container-high px-1 rounded">
                3.4.0
              </code>
              ) to install from npm. The server will run{" "}
              <code className="font-mono bg-surface-container-high px-1 rounded">
                npm install -g allure@&lt;version&gt;
              </code>
              . New report generations will use the updated binary immediately.
            </p>
            <form
              onSubmit={handleUpdate}
              className="flex items-end gap-3 flex-wrap"
            >
              <div className="flex-1 min-w-[160px]">
                <label className="block text-[10px] font-label font-bold uppercase tracking-widest text-on-surface-variant mb-1.5">
                  Version
                </label>
                <input
                  type="text"
                  value={inputVersion}
                  onChange={(e) => setInputVersion(e.target.value)}
                  placeholder="e.g. 3.4.0"
                  pattern="\d+\.\d+\.\d+"
                  required
                  className="w-full bg-surface-container-lowest border border-outline-variant/30 rounded-lg px-3 py-2 text-sm font-mono text-on-surface outline-none focus:ring-1 focus:ring-primary/40 placeholder:text-on-surface-variant/40"
                />
              </div>
              <button
                type="submit"
                disabled={updating || !inputVersion.trim()}
                className="flex items-center gap-2 bg-primary text-on-primary px-4 py-2 rounded-lg text-sm font-headline font-bold hover:brightness-110 active:scale-95 transition-all disabled:opacity-50"
              >
                <span className="material-symbols-outlined text-[16px]">
                  {updating ? "hourglass_empty" : "download"}
                </span>
                {updating ? "Installing…" : "Install"}
              </button>
            </form>
          </div>
        </section>
      ) : (
        <div className="flex items-center gap-3 px-4 py-3.5 rounded-xl bg-surface-container-low border border-outline-variant/10 text-xs text-on-surface-variant">
          <span className="material-symbols-outlined text-[16px]">lock</span>
          Only admins can install a different Allure version.
        </div>
      )}

      {SnackbarNode}
    </div>
  );
}

// ── Disk Usage section ────────────────────────────────────────────────────────

function formatBytes(bytes: number): string {
  if (bytes <= 0) return "0 B";
  const units = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.min(
    Math.floor(Math.log(bytes) / Math.log(1024)),
    units.length - 1,
  );
  return `${(bytes / Math.pow(1024, i)).toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

function usageColor(pct: number) {
  if (pct >= 85) return { bar: "bg-error", text: "text-error" };
  if (pct >= 70) return { bar: "bg-amber-500", text: "text-amber-500" };
  return { bar: "bg-emerald-500", text: "text-emerald-500" };
}

function DiskUsageSection() {
  const [data, setData] = useState<import("../types").DiskUsage | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [notificationThreshold, setNotificationThreshold] = useState(85);
  const [thresholdLoading, setThresholdLoading] = useState(true);
  const [thresholdSaving, setThresholdSaving] = useState(false);
  const { show: showAlert, SnackbarNode } = useSnackbar();

  useEffect(() => {
    api
      .getDiskUsage()
      .then(setData)
      .catch((e) =>
        setError(e instanceof Error ? e.message : "Failed to load disk usage"),
      )
      .finally(() => setLoading(false));
    api
      .getDiskNotificationThreshold()
      .then((d) => setNotificationThreshold(d.thresholdPercent))
      .catch(() => {})
      .finally(() => setThresholdLoading(false));
  }, []);

  async function saveNotificationThreshold() {
    setThresholdSaving(true);
    try {
      await api.setDiskNotificationThreshold(notificationThreshold);
      showAlert("Disk notification threshold saved");
    } catch (e) {
      showAlert(
        e instanceof Error
          ? e.message
          : "Failed to save disk notification threshold",
        "error",
      );
    } finally {
      setThresholdSaving(false);
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-14 bg-surface-container-low/40 rounded-2xl border border-outline-variant/10">
        <span className="text-on-surface-variant text-sm animate-pulse">
          Loading disk usage…
        </span>
      </div>
    );
  }

  if (error || !data) {
    return (
      <div className="flex items-center gap-2 text-xs text-error bg-error/8 rounded-xl px-4 py-3 border border-error/15">
        <span className="material-symbols-outlined text-[14px]">error</span>
        {error ?? "No data"}
      </div>
    );
  }

  const usedPct =
    data.totalBytes > 0
      ? Math.round((data.usedBytes / data.totalBytes) * 100)
      : 0;
  const colors = usageColor(usedPct);

  return (
    <div className="space-y-6">
      {/* Notification level */}
      <div className="bg-surface-container-low rounded-2xl border border-outline-variant/10 p-5">
        <div className="flex items-center justify-between gap-4 flex-wrap">
          <div>
            <p className="text-sm font-headline font-semibold text-on-surface">
              Disk Usage Notification Threshold
            </p>
            <p className="text-xs text-on-surface-variant mt-0.5">
              Send disk usage notifications when used storage reaches this
              percentage.
            </p>
          </div>
          <div className="flex items-center gap-2">
            <input
              type="number"
              min={0}
              max={100}
              value={notificationThreshold}
              onChange={(e) =>
                setNotificationThreshold(
                  Math.max(0, Math.min(100, parseInt(e.target.value, 10) || 0)),
                )
              }
              disabled={thresholdLoading || thresholdSaving}
              className="w-24 bg-surface-container-lowest border border-outline-variant/30 rounded-lg px-3 py-2 text-sm font-mono text-on-surface text-center outline-none focus:ring-1 focus:ring-primary/40 disabled:opacity-60"
            />
            <span className="text-sm text-on-surface-variant">%</span>
            <button
              onClick={saveNotificationThreshold}
              disabled={thresholdLoading || thresholdSaving}
              className="flex items-center gap-2 bg-primary text-on-primary px-4 py-2 rounded-lg text-sm font-headline font-bold hover:brightness-110 active:scale-95 transition-all disabled:opacity-50"
            >
              <span className="material-symbols-outlined text-[16px]">
                {thresholdSaving ? "hourglass_empty" : "save"}
              </span>
              {thresholdSaving ? "Saving…" : "Save"}
            </button>
          </div>
        </div>
      </div>

      {/* Usage bar */}
      <div className="bg-surface-container-low rounded-2xl border border-outline-variant/10 p-5 space-y-4">
        <div className="flex items-center justify-between gap-4">
          <div>
            <p className="text-sm font-headline font-semibold text-on-surface">
              Storage Used
            </p>
            <p className="text-xs text-on-surface-variant mt-0.5">
              Data directory - reports, results, and uploads
            </p>
          </div>
          <span
            className={`text-2xl font-headline font-bold tabular-nums ${colors.text}`}
          >
            {data.totalBytes > 0 ? `${usedPct}%` : "-"}
          </span>
        </div>

        {/* Bar */}
        <div className="h-2.5 rounded-full bg-surface-container-high overflow-hidden">
          <div
            className={`h-full rounded-full transition-all ${colors.bar}`}
            style={{ width: `${Math.min(usedPct, 100)}%` }}
          />
        </div>

        {/* Stats row */}
        <div className="grid grid-cols-3 gap-3 pt-1">
          {[
            {
              label: "Used",
              value: formatBytes(data.usedBytes),
              color: colors.text,
            },
            {
              label: "Free",
              value: formatBytes(data.freeBytes),
              color: "text-on-surface",
            },
            {
              label: "Total",
              value: formatBytes(data.totalBytes),
              color: "text-on-surface",
            },
          ].map(({ label, value, color }) => (
            <div
              key={label}
              className="bg-surface-container rounded-xl px-4 py-3 border border-outline-variant/10"
            >
              <p className="text-[10px] font-label font-bold uppercase tracking-widest text-on-surface-variant mb-1">
                {label}
              </p>
              <p
                className={`text-sm font-headline font-bold tabular-nums ${color}`}
              >
                {value}
              </p>
            </div>
          ))}
        </div>
      </div>

      {/* Per-project breakdown */}
      <section>
        <div className="flex items-center gap-2 mb-3">
          <span className="material-symbols-outlined text-[18px] text-on-surface-variant">
            folder_open
          </span>
          <h3 className="text-sm font-headline font-bold text-on-surface">
            By Project
          </h3>
          <span className="text-xs text-on-surface-variant/60 font-label">
            top {data.breakdown.length}
          </span>
        </div>

        {data.breakdown.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-10 bg-surface-container-low/30 rounded-2xl border border-dashed border-outline-variant/20">
            <span className="material-symbols-outlined text-[36px] text-on-surface-variant/20 mb-2">
              folder_off
            </span>
            <p className="text-sm font-headline font-bold text-on-surface">
              No project data yet
            </p>
          </div>
        ) : (
          <div className="bg-surface-container-low rounded-2xl border border-outline-variant/10 overflow-hidden divide-y divide-outline-variant/10">
            {data.breakdown.map((entry) => {
              const pct =
                data.usedBytes > 0 ? (entry.bytes / data.usedBytes) * 100 : 0;
              return (
                <div
                  key={entry.path}
                  className="flex items-center gap-4 px-5 py-3"
                >
                  <span className="material-symbols-outlined text-[16px] text-on-surface-variant shrink-0">
                    folder
                  </span>
                  <span className="flex-1 min-w-0 text-xs font-mono text-on-surface truncate">
                    {entry.path}
                  </span>
                  {/* Mini bar */}
                  <div className="hidden sm:block w-24 h-1.5 rounded-full bg-surface-container-high overflow-hidden shrink-0">
                    <div
                      className="h-full rounded-full bg-primary/60"
                      style={{ width: `${Math.min(pct, 100)}%` }}
                    />
                  </div>
                  <span className="text-xs font-mono text-on-surface-variant shrink-0 w-16 text-right tabular-nums">
                    {formatBytes(entry.bytes)}
                  </span>
                </div>
              );
            })}
          </div>
        )}
      </section>
      {SnackbarNode}
    </div>
  );
}

// ── Page ──────────────────────────────────────────────────────────────────────

type Tab = "apikeys" | "users" | "retention" | "allure" | "disk";

const NAV_ITEMS: {
  key: Tab;
  label: string;
  description: string;
  icon: string;
}[] = [
  {
    key: "apikeys",
    label: "API Keys",
    description: "CI/CD integration tokens",
    icon: "key",
  },
  {
    key: "users",
    label: "Users",
    description: "OAuth users & role management",
    icon: "group",
  },
  {
    key: "retention",
    label: "Data Retention",
    description: "Automatic cleanup policy",
    icon: "auto_delete",
  },
  {
    key: "allure",
    label: "Allure CLI",
    description: "Manage the report generator version",
    icon: "science",
  },
  {
    key: "disk",
    label: "Disk Usage",
    description: "Storage consumed by reports & results",
    icon: "database",
  },
];

const SECTION_META: Record<Tab, { title: string; description: string }> = {
  apikeys: {
    title: "API Keys",
    description:
      "Create and manage keys used by CI/CD pipelines and external integrations.",
  },
  users: {
    title: "Users",
    description:
      "All users who have logged in via OAuth. Admins can override their roles.",
  },
  retention: {
    title: "Data Retention",
    description:
      "Configure how long reports are kept and how often expired data is cleaned up.",
  },
  allure: {
    title: "Allure CLI",
    description:
      "View and update the Allure report generator version installed on the server.",
  },
  disk: {
    title: "Disk Usage",
    description:
      "Storage consumed by the data directory, broken down by project.",
  },
};

export default function SettingsPage() {
  const [params, setParams] = useSearchParams();
  const raw = params.get("tab");
  const activeTab: Tab =
    raw === "users"
      ? "users"
      : raw === "retention"
        ? "retention"
        : raw === "allure"
          ? "allure"
          : raw === "disk"
            ? "disk"
            : "apikeys";
  const setActiveTab = (tab: Tab) => setParams({ tab });
  const { title, description } = SECTION_META[activeTab];

  return (
    <div className="flex h-full -mx-8 -my-6">
      {/* ── Left sidebar nav ── */}
      <aside className="w-56 shrink-0 border-r border-outline-variant/15 flex flex-col px-3 py-6">
        <div className="px-2 mb-5">
          <h2 className="text-xl font-bold font-headline tracking-tight text-on-surface">
            Settings
          </h2>
        </div>
        <nav className="flex flex-col gap-0.5">
          {NAV_ITEMS.map(({ key, label, icon, description: desc }) => (
            <button
              key={key}
              onClick={() => setActiveTab(key)}
              className={`flex items-center gap-3 px-3 py-2.5 rounded-lg text-left transition-colors group ${
                activeTab === key
                  ? "bg-primary/10 text-primary"
                  : "text-on-surface-variant hover:text-on-surface hover:bg-black/5 dark:hover:bg-white/5"
              }`}
            >
              <span
                className={`material-symbols-outlined text-[18px] shrink-0 ${activeTab === key ? "text-primary" : "text-on-surface-variant group-hover:text-on-surface"}`}
              >
                {icon}
              </span>
              <div className="min-w-0">
                <p
                  className={`text-sm font-label font-bold leading-tight ${activeTab === key ? "text-primary" : ""}`}
                >
                  {label}
                </p>
                <p className="text-[10px] text-on-surface-variant/70 leading-tight mt-0.5 truncate">
                  {desc}
                </p>
              </div>
              {activeTab === key && (
                <span className="ml-auto w-1 h-5 rounded-full bg-primary shrink-0" />
              )}
            </button>
          ))}
        </nav>
      </aside>

      {/* ── Content ── */}
      <div className="flex-1 flex flex-col overflow-hidden px-8 py-6">
        <div className="mb-6 shrink-0">
          <h3 className="text-2xl font-bold font-headline tracking-tight text-on-surface">
            {title}
          </h3>
          <p className="text-sm text-on-surface-variant mt-1">{description}</p>
        </div>

        {activeTab === "apikeys" ? (
          <div className="flex-1 min-h-0">
            <APIKeysSection />
          </div>
        ) : (
          <div className="flex-1 min-h-0 overflow-y-auto">
            {activeTab === "users" && <UsersSection />}
            {activeTab === "retention" && <DataRetentionSection />}
            {activeTab === "allure" && <AllureVersionSection />}
            {activeTab === "disk" && <DiskUsageSection />}
          </div>
        )}
      </div>
    </div>
  );
}
