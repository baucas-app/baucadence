import { useEffect, useRef, useState } from "react";
import { ChevronDown, ChevronUp } from "lucide-react";
import { useImportStatus } from "~/hooks/useImportStatus";
import { useAppContext } from "~/providers/AppProvider";
import OverallImportProgress from "./OverallImportProgress";
import ImportProgressRow from "./ImportProgressRow";

export default function ImportProgressToast() {
  const summary = useImportStatus();
  const { openSettings } = useAppContext();
  const [dismissed, setDismissed] = useState(false);
  const [expanded, setExpanded] = useState(false);
  const wasActive = useRef(false);

  useEffect(() => {
    if (summary.anyActive && !wasActive.current) {
      setDismissed(false);
    }
    wasActive.current = summary.anyActive;
  }, [summary.anyActive]);

  if (summary.files.length === 0 || dismissed) return null;

  return (
    <div className="fixed bottom-4 right-4 z-50 w-[320px] card p-4 shadow-lg">
      <button
        onClick={(e) => {
          e.stopPropagation();
          setDismissed(true);
        }}
        aria-label="Dismiss"
        className="absolute top-2 right-2 color-fg-secondary hover:text-(--color-fg) text-lg leading-none px-1"
      >
        ×
      </button>
      <div
        className="pr-4 cursor-pointer"
        role="button"
        onClick={() => openSettings("Export", "import")}
        title="Open import settings"
      >
        <OverallImportProgress summary={summary} />
      </div>
      <button
        onClick={() => setExpanded((v) => !v)}
        className="flex items-center gap-1 mt-2 text-[12px] color-fg-secondary hover:text-(--color-fg)"
      >
        {expanded ? (
          <>
            Hide files <ChevronUp size={14} />
          </>
        ) : (
          <>
            Show {summary.files.length} file
            {summary.files.length === 1 ? "" : "s"} <ChevronDown size={14} />
          </>
        )}
      </button>
      {expanded && (
        <div className="flex flex-col gap-2 mt-3 max-h-[280px] overflow-y-auto pr-1">
          {summary.files.map((f) => (
            <ImportProgressRow key={f.filename} progress={f} />
          ))}
        </div>
      )}
    </div>
  );
}
