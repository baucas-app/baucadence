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

function knownTotalFiles(files: ImportFileProgress[]) {
  return files.filter((f) => f.total > 0 && !f.error);
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

  const known = knownTotalFiles(files);
  const totalKnown = known.reduce((s, f) => s + f.total, 0);
  const processedKnown = known.reduce((s, f) => s + f.processed, 0);
  const overallPct =
    totalKnown > 0
      ? Math.round((processedKnown / totalKnown) * 100)
      : undefined;

  const rateRef = useRef<{ time: number; processed: number } | null>(null);
  const [etaSeconds, setEtaSeconds] = useState<number>();

  useEffect(() => {
    if (!anyActive) {
      rateRef.current = null;
      setEtaSeconds(undefined);
      return;
    }
    const now = Date.now();
    const prev = rateRef.current;
    if (prev && totalKnown > 0) {
      const dt = (now - prev.time) / 1000;
      const dp = processedKnown - prev.processed;
      if (dt > 0 && dp > 0) {
        const rate = dp / dt;
        const remaining = totalKnown - processedKnown;
        setEtaSeconds(remaining > 0 ? remaining / rate : 0);
      }
    }
    rateRef.current = { time: now, processed: processedKnown };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [processedKnown, totalKnown, anyActive]);

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
