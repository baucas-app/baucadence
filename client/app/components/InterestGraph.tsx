import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { apiFetch, type InterestBucket } from "api/api";
import { useTheme } from "~/hooks/useTheme";
import { Area, AreaChart, XAxis, YAxis, Tooltip } from "recharts";

const WIDTH_CLASSES = "w-[350px] sm:w-[514px] md:w-[550px]";

const RANGES = [
  { key: "D", label: "D", days: 1, buckets: 24 },
  { key: "W", label: "W", days: 7, buckets: 7 },
  { key: "M", label: "M", days: 30, buckets: 30 },
  { key: "Y", label: "Y", days: 365, buckets: 12 },
  { key: "Max", label: "Max", days: 0, buckets: 16 },
] as const;

type RangeKey = (typeof RANGES)[number]["key"];

const formatTick = (value: string | Date, showTime: boolean) =>
  showTime
    ? new Date(value).toLocaleTimeString(undefined, {
        hour: "2-digit",
        minute: "2-digit",
      })
    : new Date(value).toLocaleDateString(undefined, {
        month: "short",
        day: "numeric",
      });

const formatRange = (bucket: InterestBucket, showTime: boolean) => {
  const start = new Date(bucket.bucket_start);
  const end = new Date(bucket.bucket_end);
  if (showTime) {
    return `${formatTick(start, true)} – ${formatTick(end, true)}`;
  }
  if (start.toDateString() === end.toDateString()) {
    return start.toLocaleDateString(undefined, {
      month: "short",
      day: "numeric",
      year: "numeric",
    });
  }
  return `${formatTick(start, false)} – ${formatTick(end, false)}`;
};

function InterestTooltip({
  active,
  payload,
  showTime,
}: {
  active?: boolean;
  payload?: { payload: InterestBucket }[];
  showTime: boolean;
}) {
  if (!active || !payload?.length) {
    return null;
  }
  const bucket = payload[0].payload;
  return (
    <div className="card px-3 py-2 text-[12px]">
      <div className="text-(--color-fg-secondary)">
        {formatRange(bucket, showTime)}
      </div>
      <div className="font-medium">
        {bucket.listen_count} {bucket.listen_count === 1 ? "listen" : "listens"}
      </div>
    </div>
  );
}

function RangeSelector({
  value,
  onChange,
}: {
  value: RangeKey;
  onChange: (key: RangeKey) => void;
}) {
  return (
    <div className="flex gap-1">
      {RANGES.map((r) => (
        <button
          key={r.key}
          onClick={() => onChange(r.key)}
          className={`text-[11px] px-1.5 py-0.5 rounded-sm ${
            value === r.key
              ? "bg-(--color-bg-secondary) color-fg"
              : "color-fg-secondary hover-bg-secondary"
          }`}
        >
          {r.label}
        </button>
      ))}
    </div>
  );
}

interface Props {
  type: string;
  id: number;
}

const getInterest = (args: {
  buckets: number;
  days: number;
  type: string;
  id: number;
}) =>
  apiFetch<InterestBucket[]>(
    `/apis/web/v1/${args.type.toLowerCase()}/${args.id}/interest`,
    args,
  );

export default function InterestGraph({ type, id }: Props) {
  const [range, setRange] = useState<RangeKey>("Max");
  const preset = RANGES.find((r) => r.key === range)!;

  const args = {
    buckets: preset.buckets,
    days: preset.days,
    type,
    id,
  };
  const { isPending, isError, data, error } = useQuery({
    queryKey: ["interest", args],
    queryFn: () => getInterest(args),
  });

  const { theme } = useTheme();
  const color = theme.primary;
  const showTime = range === "D";

  const title = "Interest over time";

  const header = (
    <div
      className={`flex items-center justify-between pl-6 pr-1 ${WIDTH_CLASSES}`}
    >
      <h3 className="color-fg-secondary inline-block sm:mb-1">{title}</h3>
      <RangeSelector value={range} onChange={setRange} />
    </div>
  );

  if (isPending) {
    return <InterestGraphSkeleton />;
  } else if (isError) {
    return (
      <div className="flex flex-col items-start">
        {header}
        <p className="error pl-6">Error: {error.message}</p>
      </div>
    );
  }

  // Note: I would really like to have the animation for the graph, however
  // the line graph can get weirdly clipped before the animation is done
  // so I think I just have to remove it for now.

  return (
    <div className="flex flex-col items-start">
      {header}
      <div
        className={`flex flex-col items-center h-[180px] sm:h-[205px] text-[12px] p-6 card ${WIDTH_CLASSES}`}
      >
        <AreaChart
          style={{
            width: "100%",
            maxWidth: 450,
            overflow: "visible",
            height: "150px",
          }}
          data={data}
          margin={{ top: 20, bottom: 5, left: -20 }}
        >
          <defs>
            <linearGradient id="colorGradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={color} stopOpacity={0.5} />
              <stop offset="95%" stopColor={color} stopOpacity={0} />
            </linearGradient>
          </defs>
          <XAxis
            dataKey="bucket_start"
            tickFormatter={(v) => formatTick(v, showTime)}
            tick={{ fill: "var(--color-fg-secondary)", fontSize: 11 }}
            tickLine={false}
            axisLine={{ stroke: "var(--color-bg-tertiary)" }}
            minTickGap={24}
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
            content={<InterestTooltip showTime={showTime} />}
            cursor={{ stroke: "var(--color-bg-tertiary)", strokeWidth: 1 }}
          />
          <Area
            dataKey="listen_count"
            type="natural"
            stroke="none"
            fill="url(#colorGradient)"
            animationDuration={0}
            animationEasing="ease-in-out"
            activeDot={false}
          />
          <Area
            dataKey="listen_count"
            type="natural"
            stroke={color}
            fill="none"
            strokeWidth={2}
            animationDuration={0}
            animationEasing="ease-in-out"
            dot={false}
            activeDot={{ r: 4, fill: color, stroke: "var(--color-bg)" }}
            style={{ filter: `drop-shadow(0px 0px 0px ${color})` }}
          />
        </AreaChart>
      </div>
    </div>
  );
}

export function InterestGraphSkeleton() {
  return (
    <div className="flex flex-col items-start">
      <h3 className={`color-fg-secondary inline-block sm:mb-1 pl-6`}>
        Interest over time
      </h3>
      <div
        className={`flex flex-col items-center h-[180px] sm:h-[205px] text-[12px] p-6 card ${WIDTH_CLASSES}`}
      >
        <div className="w-full max-w-[450px] h-[150px] relative overflow-hidden rounded-(--border-radius) flex justify-around items-center">
          <div className="w-full h-3/4 bg rounded-(--border-radius) animate-pulse" />
        </div>
      </div>
    </div>
  );
}
