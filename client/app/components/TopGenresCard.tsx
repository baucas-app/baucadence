import { useQuery } from "@tanstack/react-query";
import { apiFetch, type TopGenresResponse } from "api/api";
import CardHeader from "./primitives/CardHeader";

interface Props {
  period: string;
  limit?: number;
  className?: string;
}

const getTopGenres = (args: { period: string; limit: number }) =>
  apiFetch<TopGenresResponse>("/apis/web/v1/insights/genres", args);

const capitalize = (s: string) => s.charAt(0).toUpperCase() + s.slice(1);

const title = "Top genres";

export default function TopGenresCard({
  period,
  limit = 5,
  className = "w-[350px]",
}: Props) {
  const { isPending, isError, data, error } = useQuery({
    queryKey: ["insights/genres", period, limit],
    queryFn: () => getTopGenres({ period, limit }),
  });

  if (isPending) {
    return <TopGenresCardSkeleton className={className} count={limit} />;
  } else if (isError) {
    return (
      <div className={`flex flex-col items-start ${className}`}>
        <CardHeader isOffset>{title}</CardHeader>
        <p className="error">Error: {error.message}</p>
      </div>
    );
  }

  // LastFM isn't configured on this instance - nothing to show
  if (!data.enabled) {
    return null;
  }

  return (
    <div className={`flex flex-col items-start ${className}`}>
      <CardHeader isOffset>{title}</CardHeader>
      {data.genres.length === 0 ? (
        <div className="w-full card p-6">
          <p className="text-[13px] color-fg-secondary">
            Still learning your music taste for this period. Check back once
            more of your library has been analyzed.
          </p>
        </div>
      ) : (
        <div className="w-full card p-6 flex flex-col gap-3">
          {data.genres.map((genre) => (
            <div
              key={genre.name}
              className="flex items-center justify-between gap-3"
            >
              <div className="flex items-center gap-3 min-w-0">
                <span className="color-fg-secondary text-[12px] w-3 shrink-0">
                  {genre.rank}
                </span>
                <span className="truncate">{capitalize(genre.name)}</span>
              </div>
              <span className="color-fg-secondary text-[12px] shrink-0">
                {genre.listen_count}{" "}
                {genre.listen_count === 1 ? "play" : "plays"}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function TopGenresCardSkeleton() {
  return (
    <div className="flex flex-col items-start w-[350px]">
      <CardHeader isOffset>{title}</CardHeader>
      <div className="max-w-[350px] w-full card p-6 flex flex-col gap-4">
        {Array.from({ length: 5 }).map((_, i) => (
          <div
            key={i}
            className="w-full h-4 bg animate-pulse rounded-(--border-radius)"
          />
        ))}
      </div>
    </div>
  );
}
