import { useEffect, useRef, useState } from "react";
import { useImportStatus } from "~/hooks/useImportStatus";
import OverallImportProgress from "./OverallImportProgress";

export default function ImportProgressToast() {
  const summary = useImportStatus();
  const [dismissed, setDismissed] = useState(false);
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
        onClick={() => setDismissed(true)}
        aria-label="Dismiss"
        className="absolute top-2 right-2 color-fg-secondary hover:text-(--color-fg) text-lg leading-none px-1"
      >
        ×
      </button>
      <div className="pr-4">
        <OverallImportProgress summary={summary} />
      </div>
    </div>
  );
}
