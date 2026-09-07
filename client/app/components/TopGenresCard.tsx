import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { apiFetch, type TopGenresResponse, type GenreTrack } from "api/api";
import { ChevronDown } from "lucide-react";
import CardHeader from "./primitives/CardHeader";
import MediaItem from "./primitives/MediaItem";

interface Props {
  period: string;
  limit?: number;
  className?: string;
}

const getTopGenres = (args: { period: string; limit: number }) =>
  apiFetch<TopGenresResponse>("/apis/web/v1/insights/genres", args);

const getGenreTracks = (genre: string, period: string) =>
  apiFetch<GenreTrack[]>(
    `/apis/web/v1/insights/genres/${encodeURIComponent(genre)}/tracks`,
    { period },
  );

const capitalize = (s: string) => s.charAt(0).toUpperCase() + s.slice(1);

const title = "Top genres";

function GenreTracksExpansion({
  genre,
  period,
}: {
  genre: string;
  period: string;
}) {
  const { isPending, isError, data } = useQuery({
    queryKey: ["insights/genres", genre, period, "tracks"],
    queryFn: () => getGenreTracks(genre, period),
  });

  return (
    <div className="pl-6 pb-1 flex flex-col gap-2 border-l border-(--color-bg-tertiary) ml-1.5">
      {isPending ? (
        <p className="text-[12px] color-fg-secondary">Loading...</p>
      ) : isError ? (
        <p className="text-[12px] error">Failed to load tracks</p>
      ) : data.length === 0 ? (
        <p className="text-[12px] color-fg-secondary">No tracks found</p>
      ) : (
        data.map((track) => (
          <MediaItem
            key={track.id}
            image={track.image}
            size="sm"
            link={`/track/${track.id}`}
            alt={track.title}
            title={track.title}
            subtitle={track.artists.map((a) => a.name).join(", ")}
            meta={`${track.listen_count} ${track.listen_count === 1 ? "play" : "plays"}`}
          />
        ))
      )}
    </div>
  );
}

export default function TopGenresCard({
  period,
  limit = 5,
  className = "w-[350px]",
}: Props) {
  const [expanded, setExpanded] = useState<string | null>(null);

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
          {data.genres.map((genre) => {
            const isOpen = expanded === genre.name;
            return (
              <div key={genre.name} className="flex flex-col gap-3">
                <button
                  className="flex items-center justify-between gap-3 text-left w-full hover-bg-secondary rounded-(--border-radius) -m-1 p-1"
                  onClick={() => setExpanded(isOpen ? null : genre.name)}
                >
                  <div className="flex items-center gap-3 min-w-0">
                    <span className="color-fg-secondary text-[12px] w-3 shrink-0">
                      {genre.rank}
                    </span>
                    <span className="truncate">{capitalize(genre.name)}</span>
                  </div>
                  <div className="flex items-center gap-2 shrink-0">
                    <span className="color-fg-secondary text-[12px]">
                      {genre.listen_count}{" "}
                      {genre.listen_count === 1 ? "play" : "plays"}
                    </span>
                    <ChevronDown
                      size={14}
                      className={`color-fg-secondary transition-transform ${isOpen ? "rotate-180" : ""}`}
                    />
                  </div>
                </button>
                {isOpen && (
                  <GenreTracksExpansion genre={genre.name} period={period} />
                )}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

function TopGenresCardSkeleton({
  className = "w-[350px]",
  count = 5,
}: {
  className?: string;
  count?: number;
}) {
  return (
    <div className={`flex flex-col items-start ${className}`}>
      <CardHeader isOffset>{title}</CardHeader>
      <div className="w-full card p-6 flex flex-col gap-4">
        {Array.from({ length: count }).map((_, i) => (
          <div
            key={i}
            className="w-full h-4 bg animate-pulse rounded-(--border-radius)"
          />
        ))}
      </div>
    </div>
  );
}
