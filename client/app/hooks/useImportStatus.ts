import { useEffect, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { getImportStatus, type ImportFileProgress } from "api/api";
import { useAppContext } from "~/providers/AppProvider";

const ACTIVE_POLL_MS = 1500;
const IDLE_POLL_MS = 30000;

export interface ImportStatusSummary {
  files: ImportFileProgress[];
  anyActive: boolean;
  allFinished: boolean;
  overallPct?: number;
  etaSeconds?: number;
  refetch: () => void;
}

// How far along a single file is, from 0 to 1 - regardless of whether its
// total item count is known yet. A file that hasn't revealed its total yet
// contributes 0, not some placeholder value, so it doesn't inflate the
// overall progress before any real work has happened on it.
function fileCompletionFraction(f: ImportFileProgress): number {
  if (f.done || f.error) return 1;
  if (f.total > 0) return Math.min(1, f.processed / f.total);
  return 0;
}

export function useImportStatus(): ImportStatusSummary {
  const { user } = useAppContext();
  const isAdmin = user?.role === "admin";

  const { data, refetch } = useQuery({
    queryKey: ["import-status"],
    queryFn: getImportStatus,
    enabled: isAdmin,
    refetchInterval: (query) => {
      const files = query.state.data;
      if (!files || files.length === 0) return IDLE_POLL_MS;
      const active = files.some((f) => !f.done && !f.error);
      return active ? ACTIVE_POLL_MS : IDLE_POLL_MS;
    },
  });

  const files = data ?? [];
  const anyActive = files.some((f) => !f.done && !f.error);
  const allFinished = files.length > 0 && !anyActive;

  // Every file counts equally toward the overall figure, whether or not
  // its own item count is known yet - so a batch of 10 files where only
  // 1 has started doesn't read as "80% done" just because that one file
  // happens to be 80% through itself.
  const overallFraction =
    files.length > 0
      ? files.reduce((s, f) => s + fileCompletionFraction(f), 0) / files.length
      : 0;
  const overallPct =
    files.length > 0 ? Math.round(overallFraction * 100) : undefined;

  const rateRef = useRef<{ time: number; fraction: number } | null>(null);
  const [etaSeconds, setEtaSeconds] = useState<number>();

  useEffect(() => {
    if (!anyActive) {
      rateRef.current = null;
      setEtaSeconds(undefined);
      return;
    }
    const now = Date.now();
    const prev = rateRef.current;
    if (prev) {
      const dt = (now - prev.time) / 1000;
      const dFrac = overallFraction - prev.fraction;
      if (dt > 0 && dFrac > 0) {
        const rate = dFrac / dt;
        const remaining = 1 - overallFraction;
        setEtaSeconds(remaining > 0 ? remaining / rate : 0);
      }
    }
    rateRef.current = { time: now, fraction: overallFraction };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [overallFraction, anyActive]);

  return { files, anyActive, allFinished, overallPct, etaSeconds, refetch };
}

export function formatEta(seconds: number): string {
  if (seconds < 45) return "less than a minute";
  const mins = Math.round(seconds / 60);
  if (mins < 60) return `~${mins} min`;
  const hours = Math.floor(mins / 60);
  const remMins = mins % 60;
  return remMins > 0 ? `~${hours}h ${remMins}m` : `~${hours}h`;
}
