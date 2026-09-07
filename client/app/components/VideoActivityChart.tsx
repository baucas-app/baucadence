import { useQuery } from "@tanstack/react-query";
import {
  Area,
  AreaChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { apiFetch, type VideoActivityResponse } from "api/api";
import { useTheme } from "~/hooks/useTheme";
import CardHeader from "./primitives/CardHeader";
import PeriodSelector from "./PeriodSelector";
import { periodToStepRange } from "~/utils/utils";

interface Props {
  period: string;
  setPeriod: (p: string) => void;
}

const getVideoActivity = (step: string, range: number) =>
  apiFetch<VideoActivityResponse>("/apis/web/v1/insights/video-activity", {
    step,
    range,
  });

const formatDate = (value: string | Date) =>
  new Date(value).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
  });

function VideoActivityTooltip({
  active,
  payload,
}: {
  active?: boolean;
  payload?: {
    payload: { start_time: Date; longform: number; shortform: number };
  }[];
}) {
  if (!active || !payload?.length) {
    return null;
  }
  const point = payload[0].payload;
  return (
    <div className="card px-3 py-2 text-[12px]">
      <div className="text-(--color-fg-secondary)">
        {new Date(point.start_time).toLocaleDateString(undefined, {
          weekday: "short",
          month: "short",
          day: "numeric",
        })}
      </div>
      <div className="font-medium">{point.longform} longform</div>
      <div className="font-medium">{point.shortform} shorts</div>
    </div>
  );
}

export default function VideoActivityChart({ period, setPeriod }: Props) {
  const { step, range } = periodToStepRange(period);

  const { isPending, isError, data, error } = useQuery({
    queryKey: ["insights/video-activity", step, range],
    queryFn: () => getVideoActivity(step, range),
  });

  const { theme } = useTheme();
  const longformColor = theme.primary;
  const shortformColor = "var(--color-fg-tertiary)";

  const title = "Longform vs Shorts per day";

  const header = (
    <div className="flex items-center justify-between w-full flex-wrap gap-3">
      <CardHeader isOffset>{title}</CardHeader>
      <PeriodSelector current={period} setter={setPeriod} />
    </div>
  );

  if (isPending) {
    return (
      <div className="flex flex-col items-start w-full">
        {header}
        <div className="w-full h-[260px] p-6 card">
          <div className="w-full h-full bg rounded-(--border-radius) animate-pulse" />
        </div>
      </div>
    );
  } else if (isError) {
    return (
      <div className="flex flex-col items-start w-full">
        {header}
        <p className="error">Error: {error.message}</p>
      </div>
    );
  }

  const sorted = [...data.activity].sort(
    (a, b) =>
      new Date(a.start_time).getTime() - new Date(b.start_time).getTime(),
  );

  return (
    <div className="flex flex-col items-start w-full">
      {header}
      <div className="w-full h-[260px] text-[12px] p-6 card">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={sorted} margin={{ top: 10, right: 8, left: -20 }}>
            <defs>
              <linearGradient id="longformGradient" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor={longformColor} stopOpacity={0.5} />
                <stop offset="95%" stopColor={longformColor} stopOpacity={0} />
              </linearGradient>
              <linearGradient id="shortformGradient" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor={shortformColor} stopOpacity={0.4} />
                <stop offset="95%" stopColor={shortformColor} stopOpacity={0} />
              </linearGradient>
            </defs>
            <XAxis
              dataKey="start_time"
              tickFormatter={formatDate}
              tick={{ fill: "var(--color-fg-secondary)", fontSize: 11 }}
              tickLine={false}
              axisLine={{ stroke: "var(--color-bg-tertiary)" }}
              minTickGap={32}
            />
            <YAxis
              allowDecimals={false}
              domain={[0, "dataMax"]}
              tickCount={3}
              tick={{ fill: "var(--color-fg-secondary)", fontSize: 11 }}
              tickLine={false}
              axisLine={false}
              width={28}
            />
            <Tooltip
              content={<VideoActivityTooltip />}
              cursor={{ stroke: "var(--color-bg-tertiary)", strokeWidth: 1 }}
            />
            <Area
              dataKey="longform"
              type="monotone"
              stroke={longformColor}
              fill="url(#longformGradient)"
              strokeWidth={2}
              animationDuration={0}
              dot={false}
              activeDot={{ r: 4, fill: longformColor, stroke: "var(--color-bg)" }}
            />
            <Area
              dataKey="shortform"
              type="monotone"
              stroke={shortformColor}
              fill="url(#shortformGradient)"
              strokeWidth={2}
              animationDuration={0}
              dot={false}
              activeDot={{ r: 4, fill: shortformColor, stroke: "var(--color-bg)" }}
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>
      <div className="flex gap-4 mt-2 text-[12px] color-fg-secondary">
        <div className="flex items-center gap-2">
          <span className="h-2 w-2 rounded-full" style={{ background: longformColor }} />
          Longform
        </div>
        <div className="flex items-center gap-2">
          <span className="h-2 w-2 rounded-full" style={{ background: shortformColor }} />
          Shorts
        </div>
      </div>
    </div>
  );
}
