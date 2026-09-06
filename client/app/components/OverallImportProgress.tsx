import { formatEta, type ImportStatusSummary } from "~/hooks/useImportStatus";

interface Props {
  summary: ImportStatusSummary;
  bordered?: boolean;
}

export default function OverallImportProgress({ summary, bordered }: Props) {
  const { files, anyActive, allFinished, overallPct, etaSeconds } = summary;
  if (files.length === 0) return null;

  const hasError = files.some((f) => f.error);
  const pct = allFinished ? 100 : (overallPct ?? 0);

  return (
    <div
      className={
        bordered ? "pb-3 mb-1 border-b border-(--color-bg-tertiary)" : undefined
      }
    >
      <div className="flex justify-between gap-2 text-sm mb-1">
        <span className="font-medium">
          {allFinished
            ? hasError
              ? "Import finished with errors"
              : "Import complete"
            : "Overall progress"}
        </span>
        <span className="shrink-0 color-fg-secondary">
          {pct}%
          {anyActive &&
            etaSeconds !== undefined &&
            ` · ${formatEta(etaSeconds)} left`}
        </span>
      </div>
      <div className="h-2 w-full rounded-full bg-(--color-bg) overflow-hidden relative">
        <div
          className={`absolute inset-y-0 left-0 rounded-full transition-all ${
            allFinished && hasError
              ? "bg-(--color-error)"
              : "bg-(--color-primary)"
          }`}
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  );
}
