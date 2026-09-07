import { useQuery } from "@tanstack/react-query";
import { apiFetch, type VideoFormatSplit } from "api/api";
import CardHeader from "./primitives/CardHeader";

interface Props {
  period: string;
}

const getVideoFormatSplit = (period: string) =>
  apiFetch<VideoFormatSplit>("/apis/web/v1/insights/video-format-split", {
    period,
  });

const title = "Longform vs Shorts";

export default function VideoFormatSplitCard({ period }: Props) {
  const { isPending, isError, data, error } = useQuery({
    queryKey: ["insights/video-format-split", period],
    queryFn: () => getVideoFormatSplit(period),
  });

  if (isPending) {
    return (
      <div className="w-[350px] flex flex-col gap-3">
        <CardHeader>{title}</CardHeader>
        <div className="w-full h-16 bg animate-pulse rounded-(--border-radius)" />
      </div>
    );
  } else if (isError) {
    return (
      <div className="w-[350px]">
        <CardHeader>{title}</CardHeader>
        <p className="error">Error: {error.message}</p>
      </div>
    );
  }

  const total = data.longform + data.shortform;

  return (
    <div className="w-[350px]">
      <CardHeader>{title}</CardHeader>
      {total === 0 ? (
        <div className="card p-6">
          <p className="text-[13px] color-fg-secondary">Nothing to show</p>
        </div>
      ) : (
        <div className="card p-6 flex flex-col gap-4">
          <div className="flex h-3 w-full rounded-full overflow-hidden bg-(--color-bg-tertiary)">
            <div
              className="h-full bg-(--color-primary)"
              style={{ width: `${(data.longform / total) * 100}%` }}
            />
            <div
              className="h-full bg-(--color-fg-tertiary)"
              style={{ width: `${(data.shortform / total) * 100}%` }}
            />
          </div>
          <div className="flex justify-between text-[13px]">
            <div className="flex items-center gap-2">
              <span className="h-2 w-2 rounded-full bg-(--color-primary)" />
              Longform · {Math.round((data.longform / total) * 100)}%
            </div>
            <div className="flex items-center gap-2">
              <span className="h-2 w-2 rounded-full bg-(--color-fg-tertiary)" />
              Shorts · {Math.round((data.shortform / total) * 100)}%
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
