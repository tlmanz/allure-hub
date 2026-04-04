import React from "react";
import { APP_NAME, APP_VERSION } from "../design-system/tokens";
import { useTheme } from "../context/ThemeContext";

const GoogleIcon: React.FC = () => (
  <svg viewBox="0 0 24 24" className="w-5 h-5" aria-hidden="true">
    <path
      fill="#4285F4"
      d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
    />
    <path
      fill="#34A853"
      d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
    />
    <path
      fill="#FBBC05"
      d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l3.66-2.84z"
    />
    <path
      fill="#EA4335"
      d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
    />
  </svg>
);

export default function LoginPage() {
  const { theme, toggleTheme } = useTheme();
  return (
    <div className="min-h-screen flex flex-col md:flex-row bg-background relative">
      {/* Theme toggle - top-right corner */}
      <button
        onClick={toggleTheme}
        className="absolute top-4 right-4 z-20 p-2.5 rounded-xl border text-on-surface-variant hover:text-on-surface hover:bg-black/5 dark:hover:bg-white/5 transition-all hover:scale-105 active:scale-95"
        style={{ borderColor: "rgb(var(--color-outline-variant) / 0.4)" }}
        aria-label={
          theme === "dark" ? "Switch to light mode" : "Switch to dark mode"
        }
        title={theme === "dark" ? "Light mode" : "Dark mode"}
      >
        <span className="material-symbols-outlined text-[20px]">
          {theme === "dark" ? "light_mode" : "dark_mode"}
        </span>
      </button>

      {/* Left branded panel */}
      <div
        className="hidden md:flex md:w-1/2 flex-col items-center justify-center relative overflow-hidden"
        style={{ background: "rgb(var(--color-surface-container-low))" }}
      >
        {/* Subtle grid pattern */}
        <div
          className="absolute inset-0 opacity-[0.04]"
          style={{
            backgroundImage:
              "linear-gradient(rgb(var(--color-primary)) 1px, transparent 1px), linear-gradient(90deg, rgb(var(--color-primary)) 1px, transparent 1px)",
            backgroundSize: "40px 40px",
          }}
        />

        <div className="relative z-10 flex flex-col items-center gap-8 px-12">
          <div className="flex items-center gap-4 animate-fade-in-up">
            <img
              src="/images/logo.png"
              alt=""
              className="h-20 drop-shadow-lg"
            />
            <h1 className="text-4xl font-black tracking-tighter font-headline leading-none">
              <span className="text-gray-900 dark:text-white">Allure</span>
              <br />
              <span className="text-gray-500 dark:text-gray-400">Hub</span>
            </h1>
          </div>
          <div className="flex flex-col items-center gap-3 text-center animate-fade-in-up">
            <p className="text-base text-on-surface-variant font-label max-w-xs leading-relaxed">
              Centralized test reporting and analytics for your team
            </p>
          </div>
          <span
            className="text-xs font-mono text-on-surface-variant/50 tracking-wider animate-fade-in"
            style={{ animationDelay: "0.3s" }}
          >
            {APP_VERSION}
          </span>
        </div>

        {/* Animated corner accents */}
        <div
          className="absolute -bottom-20 -left-20 w-64 h-64 rounded-full opacity-[0.06] animate-float"
          style={{
            background: "rgb(var(--color-primary))",
            animationDelay: "1s",
          }}
        />
        <div
          className="absolute -top-16 -right-16 w-48 h-48 rounded-full opacity-[0.04] animate-float"
          style={{
            background: "rgb(var(--color-secondary))",
            animationDelay: "2s",
          }}
        />
        <div
          className="absolute bottom-1/4 right-12 w-20 h-20 rounded-full opacity-[0.05] animate-float"
          style={{
            background: "rgb(var(--color-tertiary))",
            animationDelay: "3s",
          }}
        />
      </div>

      {/* Right sign-in panel */}
      <div className="flex-1 flex items-center justify-center px-6 py-12">
        <div className="w-full max-w-sm flex flex-col items-center gap-8">
          {/* Mobile-only logo + brand */}
          <div className="flex items-center gap-3 md:hidden animate-scale-in">
            <img src="/images/logo.png" alt="" className="h-14" />
            <span className="text-2xl font-black tracking-tighter font-headline leading-none select-none">
              <span className="text-gray-900 dark:text-white">Allure</span>
              <br />
              <span className="text-gray-500 dark:text-gray-400">Hub</span>
            </span>
          </div>

          {/* Sign-in card */}
          <div
            className="w-full rounded-2xl border p-8 flex flex-col items-center gap-6 animate-fade-in-up"
            style={{
              background: "rgb(var(--color-surface-container-lowest))",
              borderColor: "rgb(var(--color-outline-variant) / 0.4)",
            }}
          >
            <div className="flex flex-col items-center gap-1">
              <span className="text-xl font-bold text-on-surface font-headline">
                Welcome back
              </span>
              <span className="text-sm text-on-surface-variant font-label">
                Sign in to continue to {APP_NAME}
              </span>
            </div>

            <a
              href="/auth/google"
              className="w-full flex items-center justify-center gap-3 px-4 py-3 rounded-xl border font-medium text-sm font-headline transition-all hover:bg-black/5 dark:hover:bg-white/5 hover:border-primary/30 hover:scale-[1.02] active:scale-[0.98]"
              style={{
                borderColor: "rgb(var(--color-outline-variant) / 0.6)",
                color: "rgb(var(--color-on-surface))",
              }}
            >
              <GoogleIcon />
              Continue with Google
            </a>
          </div>

          <p
            className="text-[11px] text-on-surface-variant/50 font-label text-center leading-relaxed max-w-xs animate-fade-in"
            style={{ animationDelay: "0.4s" }}
          >
            By signing in you agree to your organization's access policies
          </p>
        </div>
      </div>
    </div>
  );
}
