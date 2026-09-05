import { useEffect, useRef, useState } from "react";
import { uploadImportFile, getImportStatus, type ImportFileProgress } from "api/api";
import { AsyncButton } from "../AsyncButton";
import SubHeader from "../primitives/SubHeader";

const POLL_INTERVAL_MS = 1500;

export default function ImportModal() {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [loading, setLoading] = useState(false);
  const [err, setError] = useState<string>();
  const [progress, setProgress] = useState<ImportFileProgress[]>();
  const pollRef = useRef<ReturnType<typeof setInterval> | undefined>(
    undefined,
  );

  const stopPolling = () => {
    if (pollRef.current) {
      clearInterval(pollRef.current);
      pollRef.current = undefined;
    }
  };

  const pollStatus = () => {
    getImportStatus()
      .then((p) => {
        setProgress(p);
        if (p.length > 0 && p.every((f) => f.done)) {
          stopPolling();
        }
      })
      .catch(() => stopPolling());
  };

  const startPolling = () => {
    stopPolling();
    pollStatus();
    pollRef.current = setInterval(pollStatus, POLL_INTERVAL_MS);
  };

  useEffect(() => stopPolling, []);

  const handleUpload = () => {
    if (!selectedFile) {
      setError("choose a file first");
      return;
    }
    setError(undefined);
    setLoading(true);
    uploadImportFile(selectedFile)
      .then(() => {
        setSelectedFile(null);
        if (fileInputRef.current) fileInputRef.current.value = "";
        startPolling();
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  };

  return (
    <div className="">
      <SubHeader>Import Listening History</SubHeader>
      <p className="mb-4 opacity-80">
        Upload a listening-history export to backfill old listens — a
        Spotify "Extended Streaming History" export, a Google Takeout
        YouTube/YouTube Music export, or a Maloja/LastFM/ListenBrainz/Koito
        export. You can upload the whole .zip as downloaded, or a single
        already-extracted export file. Import runs in the background after
        upload; it can take a while for large histories.
      </p>
      <div className="flex flex-col gap-3 bg-secondary p-5 rounded-lg w-3/5">
        <input
          ref={fileInputRef}
          type="file"
          accept=".zip,.json"
          onChange={(e) => setSelectedFile(e.target.files?.[0] ?? null)}
          className="bg-(--color-bg) p-3 rounded-md"
        />
        <AsyncButton loading={loading} onClick={handleUpload}>
          Upload &amp; Import
        </AsyncButton>
      </div>
      {err && <p className="error mt-3">{err}</p>}
      {progress && progress.length > 0 && (
        <div className="flex flex-col gap-3 mt-5 w-3/5">
          {progress.map((f) => (
            <ImportProgressRow key={f.filename} progress={f} />
          ))}
        </div>
      )}
    </div>
  );
}

function ImportProgressRow({ progress }: { progress: ImportFileProgress }) {
  const known = progress.total > 0;
  const pct = known
    ? Math.min(100, Math.round((progress.processed / progress.total) * 100))
    : undefined;

  return (
    <div className="bg-secondary p-3 rounded-md">
      <div className="flex justify-between gap-2 text-sm mb-1">
        <span className="truncate">{progress.filename}</span>
        <span className="shrink-0">
          {progress.error
            ? "failed"
            : progress.done
              ? "done"
              : known
                ? `${pct}% (${progress.processed}/${progress.total})`
                : `processing… (${progress.processed})`}
        </span>
      </div>
      <div className="h-2 w-full rounded-full bg-(--color-bg) overflow-hidden">
        <div
          className={`h-full rounded-full transition-all ${
            progress.error ? "bg-(--color-error)" : "bg-(--color-primary)"
          }`}
          style={{
            width:
              progress.error || progress.done
                ? "100%"
                : known
                  ? `${pct}%`
                  : "35%",
          }}
        />
      </div>
      {progress.error && (
        <p className="error mt-1 text-sm">{progress.error}</p>
      )}
    </div>
  );
}
