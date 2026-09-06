import { useQuery } from "@tanstack/react-query";
import {
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { apiFetch, type ListenActivityResponse } from "api/api";
import { useTheme } from "~/hooks/useTheme";
import CardHeader from "./primitives/CardHeader";

const RANGE_DAYS = 90;

const getActivity = () =>
  apiFetch<ListenActivityResponse>("/apis/web/v1/listen-activity", {
    step: "day",
    range: RANGE_DAYS,
    month: 0,
    year: 0,
    artist_id: 0,
    album_id: 0,
    track_id: 0,
  });

const formatDate = (value: string | Date) =>
  new Date(value).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
  });

function DailyListensTooltip({
  active,
  payload,
}: {
  active?: boolean;
  payload?: { payload: { start_time: Date; listens: number } }[];
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
      <div className="font-medium">
        {point.listens} {point.listens === 1 ? "listen" : "listens"}
      </div>
    </div>
  );
}

export default function DailyListensChart() {
  const { isPending, isError, data, error } = useQuery({
    queryKey: ["listen-activity", "daily-chart", RANGE_DAYS],
    queryFn: getActivity,
  });

  const { theme } = useTheme();
  const color = theme.primary;

  const title = "Listens per day";

  if (isPending) {
    return <DailyListensChartSkeleton />;
  } else if (isError) {
    return (
      <div className="flex flex-col items-start w-full">
        <CardHeader isOffset>{title}</CardHeader>
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
      <CardHeader isOffset>{title}</CardHeader>
      <div className="w-full h-[260px] text-[12px] p-6 card">
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={sorted} margin={{ top: 10, right: 8, left: -20 }}>
            <CartesianGrid vertical={false} stroke="var(--color-bg-tertiary)" />
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
              tick={{ fill: "var(--color-fg-secondary)", fontSize: 11 }}
              tickLine={false}
              axisLine={false}
              width={36}
            />
            <Tooltip
              content={<DailyListensTooltip />}
              cursor={{ fill: "var(--color-bg-tertiary)", opacity: 0.4 }}
            />
            <Bar
              dataKey="listens"
              fill={color}
              radius={[3, 3, 0, 0]}
              maxBarSize={14}
              isAnimationActive={false}
            />
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}

function DailyListensChartSkeleton() {
  return (
    <div className="flex flex-col items-start w-full">
      <CardHeader isOffset>Listens per day</CardHeader>
      <div className="w-full h-[260px] p-6 card">
        <div className="w-full h-full bg rounded-(--border-radius) animate-pulse" />
      </div>
    </div>
  );
}
