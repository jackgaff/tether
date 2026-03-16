import { Camera, RefreshCcw, Upload, X } from "lucide-react";
import { useRef, useState } from "react";
import { Avatar } from "../Avatar";

interface ProfilePhotoFieldProps {
  name: string;
  value: string;
  onChange: (value: string) => void;
}

export function ProfilePhotoField({ name, value, onChange }: ProfilePhotoFieldProps) {
  const inputRef = useRef<HTMLInputElement | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function handleFile(file: File) {
    setBusy(true);
    setError(null);
    try {
      const next = await resizeImage(file);
      onChange(next);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not read that image.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="app-panel-muted flex flex-col gap-4 p-5">
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="eyebrow mb-1">Profile Photo</p>
          <h3 className="text-base font-semibold text-slate-900">Put a face to the profile</h3>
          <p className="mt-1 text-sm text-slate-500">
            Upload a square or portrait image. We resize it automatically for the caregiver dashboard.
          </p>
        </div>
        <div className="rounded-2xl bg-white/80 p-2 text-slate-400 shadow-[0_12px_24px_rgba(15,23,42,0.08)]">
          <Camera size={18} strokeWidth={1.8} />
        </div>
      </div>

      <div className="flex flex-col gap-4 rounded-[28px] border border-dashed border-slate-200 bg-white/90 p-5 md:flex-row md:items-center">
        <Avatar name={name} imageUrl={value || undefined} size="xl" accent="amber" />
        <div className="flex-1">
          <p className="text-sm font-medium text-slate-700">
            {value ? "Photo ready" : "No photo uploaded yet"}
          </p>
          <p className="mt-1 text-sm text-slate-500">
            JPG, PNG, or WebP. Large files are resized to keep the app fast.
          </p>
          <div className="mt-4 flex flex-wrap gap-2">
            <button
              type="button"
              onClick={() => inputRef.current?.click()}
              className="app-btn-primary"
              disabled={busy}
            >
              <Upload size={15} strokeWidth={2} />
              {busy ? "Processing..." : value ? "Replace photo" : "Upload photo"}
            </button>
            {value && (
              <button
                type="button"
                onClick={() => onChange("")}
                className="app-btn-secondary"
              >
                <X size={15} strokeWidth={2} />
                Remove
              </button>
            )}
          </div>
          {error && <p className="mt-3 text-sm text-rose-600">{error}</p>}
        </div>
      </div>

      <input
        ref={inputRef}
        type="file"
        accept="image/png,image/jpeg,image/webp"
        className="hidden"
        onChange={(event) => {
          const file = event.target.files?.[0];
          if (file) {
            void handleFile(file);
          }
          event.target.value = "";
        }}
      />
    </div>
  );
}

function resizeImage(file: File): Promise<string> {
  if (!file.type.startsWith("image/")) {
    return Promise.reject(new Error("Please choose an image file."));
  }

  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onerror = () => reject(new Error("Could not load the selected file."));
    reader.onload = () => {
      const source = new Image();
      source.onerror = () => reject(new Error("That image format could not be processed."));
      source.onload = () => {
        const maxEdge = 720;
        const scale = Math.min(1, maxEdge / Math.max(source.width, source.height));
        const width = Math.max(1, Math.round(source.width * scale));
        const height = Math.max(1, Math.round(source.height * scale));
        const canvas = document.createElement("canvas");
        canvas.width = width;
        canvas.height = height;
        const context = canvas.getContext("2d");
        if (!context) {
          reject(new Error("Could not prepare the image editor."));
          return;
        }
        context.drawImage(source, 0, 0, width, height);

        const mimeType = file.type === "image/png" ? "image/png" : "image/jpeg";
        const quality = mimeType === "image/png" ? undefined : 0.86;
        const result = canvas.toDataURL(mimeType, quality);
        if (result.length > 1_200_000) {
          reject(new Error("Please choose a smaller image."));
          return;
        }
        resolve(result);
      };
      source.src = String(reader.result);
    };
    reader.readAsDataURL(file);
  });
}
