import { useAuth } from "../context/AuthContext";

const ROLE_LABEL: Record<string, { label: string; color: string }> = {
  admin: { label: "Admin", color: "bg-primary/10 text-primary" },
  developer: { label: "Developer", color: "bg-secondary/10 text-secondary" },
  viewer: {
    label: "Viewer",
    color: "bg-surface-container-high text-on-surface-variant",
  },
};

const PROVIDER_ICON: Record<string, string> = {
  google: "g_translate",
  github: "code",
  local: "lock",
};

export default function ProfilePage() {
  const { user, logout } = useAuth();

  if (!user) return null;

  const role = ROLE_LABEL[user.role] ?? {
    label: user.role,
    color: "bg-surface-container-high text-on-surface-variant",
  };
  const providerIcon = PROVIDER_ICON[user.provider] ?? "account_circle";

  const fields: {
    icon: string;
    label: string;
    value: string;
    lowercase?: boolean;
  }[] = [
    { icon: "badge", label: "Display name", value: user.name || "-" },
    { icon: "email", label: "Email", value: user.email, lowercase: true },
    { icon: "login", label: "Provider", value: user.provider },
    { icon: "admin_panel_settings", label: "Role", value: user.role },
  ];

  return (
    <div className="max-w-xl mx-auto py-10">
      {/* Avatar + name card */}
      <div
        className="rounded-2xl p-8 flex flex-col items-center text-center mb-6 border"
        style={{
          background: "rgb(var(--color-surface-container-low))",
          borderColor: "rgb(var(--color-outline-variant) / 0.25)",
        }}
      >
        {user.avatarUrl ? (
          <img
            src={user.avatarUrl}
            alt={user.name}
            referrerPolicy="no-referrer"
            className="w-24 h-24 rounded-full ring-4 ring-primary/20 mb-4"
          />
        ) : (
          <div className="w-24 h-24 rounded-full bg-primary/20 flex items-center justify-center mb-4 ring-4 ring-primary/20">
            <span className="text-4xl font-black font-headline text-primary">
              {user.name?.[0]?.toUpperCase() ?? user.email[0].toUpperCase()}
            </span>
          </div>
        )}

        <h2 className="text-2xl font-headline font-bold text-on-surface tracking-tight">
          {user.name || user.email}
        </h2>
        <p className="text-sm text-on-surface-variant font-body mt-1">
          {user.email}
        </p>

        <div className="flex items-center gap-2 mt-3">
          <span
            className={`text-xs font-bold font-label px-2.5 py-1 rounded-full capitalize ${role.color}`}
          >
            {role.label}
          </span>
          <span className="flex items-center gap-1 text-xs font-label text-on-surface-variant px-2.5 py-1 rounded-full bg-surface-container-high capitalize">
            <span className="material-symbols-outlined text-[13px]">
              {providerIcon}
            </span>
            {user.provider}
          </span>
        </div>
      </div>

      {/* Details */}
      <div
        className="rounded-2xl border overflow-hidden"
        style={{
          background: "rgb(var(--color-surface-container-low))",
          borderColor: "rgb(var(--color-outline-variant) / 0.25)",
        }}
      >
        {fields.map(({ icon, label, value, lowercase }) => (
          <div key={label} className="flex items-center gap-4 px-6 py-4">
            <span
              className="material-symbols-outlined text-[20px] shrink-0"
              style={{ color: "rgb(var(--color-on-surface-variant))" }}
            >
              {icon}
            </span>
            <div className="flex-1 min-w-0">
              <p className="text-[11px] font-label font-semibold uppercase tracking-widest text-on-surface-variant mb-0.5">
                {label}
              </p>
              <p
                className={`text-sm font-body text-on-surface truncate ${lowercase ? "lowercase" : "capitalize"}`}
              >
                {value}
              </p>
            </div>
          </div>
        ))}
      </div>

      {/* Sign out */}
      <button
        onClick={logout}
        className="mt-6 w-full flex items-center justify-center gap-2 py-2.5 rounded-xl border border-error/20
                   text-error text-sm font-headline font-semibold hover:bg-error/5 active:scale-[0.98] transition-all"
      >
        <span className="material-symbols-outlined text-[18px]">logout</span>
        Sign out
      </button>
    </div>
  );
}
