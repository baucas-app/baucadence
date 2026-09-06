import { type ImportFileProgress } from "api/api";

export default function ImportProgressRow({
  progress,
}: {
  progress: ImportFileProgress;
}) {
  const known = progress.total > 0;
  const pct = known
    ? Math.min(100, Math.round((progress.processed / progress.total) * 100))
    : undefined;

  return (
    <div className="bg-secondary p-3 rounded-md">
      <div className="flex justify-between gap-2 text-sm mb-1">
        <span className="truncate">
          {progress.source && (
            <span className="color-fg-secondary">{progress.source} · </span>
          )}
          {progress.filename}
        </span>
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
      <div className="h-2 w-full rounded-full bg-(--color-bg) overflow-hidden relative">
        <div
          className={`absolute inset-y-0 left-0 rounded-full transition-all ${
            progress.error ? "bg-(--color-error)" : "bg-(--color-primary)"
          }`}
          style={{
            width:
              progress.error || progress.done
                ? "100%"
                : known
                  ? `${pct}%`
                  : progress.processed > 0
                    ? "35%"
                    : "0%",
          }}
        />
      </div>
      {progress.error && <p className="error mt-1 text-sm">{progress.error}</p>}
    </div>
  );
}
