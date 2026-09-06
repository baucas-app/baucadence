import { useEffect } from "react";

interface Props {
  setter: Function;
  current: string;
  disableCache?: boolean;
  className?: string;
}

export default function PeriodSelector({
  setter,
  current,
  disableCache = false,
  className,
}: Props) {
  const periods = ["day", "week", "month", "year", "all_time"];

  const shortLabels: Record<string, string> = {
    day: "D",
    week: "W",
    month: "M",
    year: "Y",
    all_time: "Max",
  };

  const periodDisplay = (str: string) => shortLabels[str] ?? str;

  const setPeriod = (val: string) => {
    setter(val);
    if (!disableCache) {
      localStorage.setItem(
        "period_selection_" + window.location.pathname.split("/")[1],
        val,
      );
    }
  };

  useEffect(() => {
    if (!disableCache) {
      const cached = localStorage.getItem(
        "period_selection_" + window.location.pathname.split("/")[1],
      );
      if (cached) {
        setter(cached);
      }
    }
  }, []);

  return (
    <div
      className={`flex gap-5 sm:gap-6 grow-0 text-sm sm:text-[16px] px-4 sm:px-6 py-2 card ${className ?? ""}`}
    >
      {periods.map((p) => (
        <div key={`period_setter_${p}`}>
          <button
            className={`period-selector ${
              p === current ? "color-fg" : "color-fg-tertiary"
            }`}
            onClick={() => setPeriod(p)}
            disabled={p === current}
          >
            {periodDisplay(p)}
          </button>
        </div>
      ))}
    </div>
  );
}
