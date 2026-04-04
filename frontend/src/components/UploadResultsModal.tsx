import { useEffect, useRef, useState, useCallback } from "react";
import { load as parseYaml } from "js-yaml";
import { useUpload } from "../context/UploadContext";
import { useFocusTrap } from "../hooks/useFocusTrap";

interface Props {
  isOpen: boolean;
  onClose: () => void;
  envId: string;
  projectId: string;
}

function generateBuildId(): string {
  const now = new Date();
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${now.getFullYear()}${pad(now.getMonth() + 1)}${pad(now.getDate())}-${pad(now.getHours())}${pad(now.getMinutes())}${pad(now.getSeconds())}`;
}

export default function UploadResultsModal({
  isOpen,
  onClose,
  envId,
  projectId,
}: Props) {
  const { startUpload } = useUpload();
  const [buildId, setBuildId] = useState("");
  const [file, setFile] = useState<File | null>(null);
  const [fileError, setFileError] = useState<string | null>(null);
  const [dragging, setDragging] = useState(false);
  const [configOpen, setConfigOpen] = useState(false);
  const [configText, setConfigText] = useState("");
  const [configError, setConfigError] = useState<string | null>(null);
  const fileRef = useRef<HTMLInputElement>(null);
  const dialogRef = useRef<HTMLDivElement>(null);
  useFocusTrap(dialogRef, isOpen);

  const MAX_FILE_BYTES = 512 * 1024 * 1024; // 512 MB - matches server limit

  function acceptFile(f: File | undefined) {
    if (!f) return;
    if (f.size > MAX_FILE_BYTES) {
      setFileError(
        `File too large (${(f.size / 1024 / 1024).toFixed(0)} MB). Maximum is 512 MB.`,
      );
      setFile(null);
      return;
    }
    setFileError(null);
    setFile(f);
  }

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setDragging(false);
    const dropped = e.dataTransfer.files[0];
    if (
      dropped &&
      (dropped.type === "application/zip" || dropped.name.endsWith(".zip"))
    ) {
      acceptFile(dropped);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setDragging(true);
  }, []);

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setDragging(false);
  }, []);

  // Reset state whenever the modal opens.
  useEffect(() => {
    if (isOpen) {
      setBuildId(generateBuildId());
      setFile(null);
      setFileError(null);
      setConfigOpen(false);
      setConfigText("");
      setConfigError(null);
      if (fileRef.current) fileRef.current.value = "";
    }
  }, [isOpen]);

  useEffect(() => {
    if (!isOpen) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [isOpen, onClose]);

  const BUILD_ID_PATTERN = /^[a-zA-Z0-9._-]+$/;

  function validateConfigYaml(value: string): string | null {
    const trimmed = value.trim();
    if (!trimmed) return null;
    try {
      const parsed = parseYaml(trimmed);
      if (
        parsed !== null &&
        parsed !== undefined &&
        (typeof parsed !== "object" || Array.isArray(parsed))
      ) {
        return "Config must be a YAML mapping (key: value pairs).";
      }
      return null;
    } catch (err) {
      return err instanceof Error ? err.message : "Invalid YAML.";
    }
  }

  function handleConfigChange(value: string) {
    setConfigText(value);
    setConfigError(validateConfigYaml(value));
  }

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!file || !buildId.trim() || fileError) return;
    if (!BUILD_ID_PATTERN.test(buildId.trim())) return;

    let reportConfig: Record<string, unknown> | undefined;
    const trimmedConfig = configText.trim();
    if (trimmedConfig) {
      const err = validateConfigYaml(trimmedConfig);
      if (err) {
        setConfigError(err);
        return;
      }
      const parsed = parseYaml(trimmedConfig);
      reportConfig = (parsed ?? undefined) as
        | Record<string, unknown>
        | undefined;
    }

    startUpload(envId, projectId, buildId.trim(), file, reportConfig);
    onClose();
  }

  if (!isOpen) return null;

  return (
    <div
      ref={dialogRef}
      className="fixed inset-0 z-50 flex items-center justify-center p-4"
      role="dialog"
      aria-modal="true"
      aria-labelledby="upload-modal-title"
      tabIndex={-1}
    >
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/40 backdrop-blur-sm"
        onClick={onClose}
      />

      {/* Modal */}
      <div className="relative bg-surface-container-low rounded-2xl border border-outline-variant/20 shadow-2xl w-full max-w-md p-6 flex flex-col gap-5">
        {/* Header */}
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-xl bg-primary/10 flex items-center justify-center">
              <span className="material-symbols-outlined text-[20px] text-primary">
                upload_file
              </span>
            </div>
            <div>
              <h3
                id="upload-modal-title"
                className="text-base font-headline font-bold text-on-surface"
              >
                Upload Results
              </h3>
              <p className="text-[11px] text-on-surface-variant font-label">
                Zip file containing Allure result files
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            aria-label="Close modal"
            className="w-8 h-8 flex items-center justify-center rounded-full hover:bg-surface-container transition-colors text-on-surface-variant"
          >
            <span
              className="material-symbols-outlined text-[20px]"
              aria-hidden="true"
            >
              close
            </span>
          </button>
        </div>

        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          {/* Build ID */}
          <div>
            <label
              htmlFor="upload-build-id"
              className="text-[11px] font-label font-bold uppercase tracking-wider text-on-surface-variant mb-1.5 block"
            >
              Build ID
            </label>
            <input
              id="upload-build-id"
              type="text"
              value={buildId}
              onChange={(e) => setBuildId(e.target.value)}
              required
              maxLength={128}
              pattern="[a-zA-Z0-9._\-]+"
              className="w-full bg-surface-container border border-outline-variant/30 rounded-lg px-3 py-2.5 text-sm font-mono text-on-surface outline-none focus:ring-1 focus:ring-primary/40 placeholder:text-on-surface-variant/40"
            />
            {buildId.trim() && !/^[a-zA-Z0-9._-]+$/.test(buildId.trim()) && (
              <p className="text-[11px] text-error font-medium mt-1">
                Only letters, numbers, dots, hyphens, and underscores allowed.
              </p>
            )}
          </div>

          {/* File picker */}
          <div>
            <label
              htmlFor="upload-zip-file"
              className="text-[11px] font-label font-bold uppercase tracking-wider text-on-surface-variant mb-1.5 block"
            >
              Results ZIP
            </label>
            <div
              className={`relative flex flex-col items-center justify-center gap-2 border-2 border-dashed rounded-xl p-5 cursor-pointer transition-colors
                ${dragging ? "border-primary bg-primary/10 scale-[1.01]" : file ? "border-primary/40 bg-primary/5" : "border-outline-variant hover:border-primary/50 hover:bg-surface-container"}`}
              onClick={() => fileRef.current?.click()}
              onDrop={handleDrop}
              onDragOver={handleDragOver}
              onDragLeave={handleDragLeave}
            >
              <input
                ref={fileRef}
                id="upload-zip-file"
                type="file"
                accept=".zip,application/zip"
                className="sr-only"
                onChange={(e) => acceptFile(e.target.files?.[0])}
              />
              {file ? (
                <>
                  <span className="material-symbols-outlined text-[28px] text-primary">
                    task
                  </span>
                  <p className="text-sm font-label font-semibold text-on-surface text-center truncate max-w-full px-2">
                    {file.name}
                  </p>
                  <p className="text-[11px] text-on-surface-variant">
                    {(file.size / 1024 / 1024).toFixed(1)} MB
                  </p>
                </>
              ) : (
                <>
                  <span className="material-symbols-outlined text-[28px] text-on-surface-variant/40">
                    cloud_upload
                  </span>
                  <p className="text-sm font-label text-on-surface-variant">
                    {dragging
                      ? "Drop to upload"
                      : "Drag & drop or click to select a .zip file"}
                  </p>
                </>
              )}
            </div>
            {fileError && (
              <p className="text-[11px] text-error font-medium mt-1.5">
                {fileError}
              </p>
            )}
          </div>

          {/* Report Config (optional, collapsible) */}
          <div>
            <button
              type="button"
              onClick={() => setConfigOpen((o) => !o)}
              className="flex items-center gap-1.5 text-[11px] font-label font-bold uppercase tracking-wider text-on-surface-variant hover:text-on-surface transition-colors"
              aria-expanded={configOpen}
            >
              <span
                className="material-symbols-outlined text-[14px] transition-transform"
                style={{
                  transform: configOpen ? "rotate(90deg)" : "rotate(0deg)",
                }}
                aria-hidden="true"
              >
                chevron_right
              </span>
              Report Config
              <span className="normal-case tracking-normal font-normal text-on-surface-variant/60 ml-1">
                (optional)
              </span>
            </button>

            {configOpen && (
              <div className="mt-2 flex flex-col gap-1.5">
                <p className="text-[11px] text-on-surface-variant leading-relaxed">
                  Override <span className="font-mono">allurerc.yml</span>{" "}
                  settings for this upload. Enter valid YAML - server-controlled
                  keys (<span className="font-mono">output</span>,{" "}
                  <span className="font-mono">historyPath</span>) are ignored.
                </p>
                <textarea
                  aria-label="Report config YAML"
                  value={configText}
                  onChange={(e) => handleConfigChange(e.target.value)}
                  rows={6}
                  spellCheck={false}
                  className={`w-full bg-surface-container border rounded-lg px-3 py-2.5 text-xs font-mono text-on-surface outline-none focus:ring-1 resize-y placeholder:text-on-surface-variant/40 ${
                    configError
                      ? "border-error/60 focus:ring-error/40"
                      : "border-outline-variant/30 focus:ring-primary/40"
                  }`}
                  placeholder={
                    "plugins:\n  awesome:\n    options:\n      reportName: My Report"
                  }
                />
                {configError && (
                  <p className="text-[11px] text-error font-medium">
                    {configError}
                  </p>
                )}
              </div>
            )}
          </div>

          {/* Actions */}
          <div className="flex items-center gap-3 pt-1">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 px-4 py-2.5 rounded-lg border border-outline-variant/30 text-sm font-headline font-bold text-on-surface-variant hover:bg-surface-container transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={
                !file ||
                !buildId.trim() ||
                !!fileError ||
                !/^[a-zA-Z0-9._-]+$/.test(buildId.trim()) ||
                !!configError
              }
              className="flex-1 flex items-center justify-center gap-2 px-4 py-2.5 rounded-lg bg-primary text-on-primary text-sm font-headline font-bold hover:brightness-110 active:scale-95 transition-all disabled:opacity-40 disabled:cursor-not-allowed"
            >
              <span className="material-symbols-outlined text-[16px]">
                upload
              </span>
              Start Upload
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
