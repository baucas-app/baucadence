import { useQuery } from "@tanstack/react-query";
import { apiFetch, type MoodInsight } from "api/api";
import ArtistLinks from "./ArtistLinks";
import MediaItem, { MediaItemSkeleton } from "./primitives/MediaItem";
import CardHeader from "./primitives/CardHeader";

interface Props {
  period: string;
}

const getMoodInsight = (args: { period: string }) =>
  apiFetch<MoodInsight>("/apis/web/v1/insights/mood", args);

const capitalize = (s: string) => s.charAt(0).toUpperCase() + s.slice(1);

const title = "Mood";

export default function MoodInsightCard({ period }: Props) {
  const { isPending, isError, data, error } = useQuery({
    queryKey: ["insights/mood", period],
    queryFn: () => getMoodInsight({ period }),
  });

  if (isPending) {
    return <MoodInsightCardSkeleton />;
  } else if (isError) {
    return (
      <div className="flex flex-col items-start">
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
    <div className="flex flex-col items-start min-w-[350px] w-full max-w-[450px]">
      <CardHeader isOffset>{title}</CardHeader>
      <div className="flex flex-col gap-4 w-full p-6 card">
        {!data.mood ? (
          <p className="text-[13px] color-fg-secondary">
            Still learning your music taste for this period. Check back once
            more of your library has been analyzed.
          </p>
        ) : (
          <>
            <div>
              <div className="text-[20px] font-medium">
                {capitalize(data.mood)}
              </div>
              {data.top_tags.length > 0 && (
                <div className="flex flex-wrap gap-2 mt-2">
                  {data.top_tags.map((tag) => (
                    <span
                      key={tag}
                      className="text-[11px] color-fg-secondary bg-(--color-bg-tertiary) rounded-full px-2 py-0.5"
                    >
                      {tag}
                    </span>
                  ))}
                </div>
              )}
            </div>
            {data.top_tracks.length > 0 && (
              <div className="flex flex-col gap-3">
                {data.top_tracks.map((track) => (
                  <MediaItem
                    key={track.id}
                    className="gap-2"
                    image={track.image}
                    link={`/track/${track.id}`}
                    size="sm"
                    title={track.title}
                    alt={track.title}
                    meta={<ArtistLinks artists={track.artists} />}
                    lazy
                  />
                ))}
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}

function MoodInsightCardSkeleton() {
  return (
    <div className="flex flex-col items-start min-w-[350px] w-full max-w-[450px]">
      <CardHeader isOffset>{title}</CardHeader>
      <div className="flex flex-col gap-3 w-full p-6 card">
        <div className="w-24 h-6 bg animate-pulse rounded-(--border-radius)" />
        <MediaItemSkeleton size="sm" className="gap-2" subtitle />
        <MediaItemSkeleton size="sm" className="gap-2" subtitle />
      </div>
    </div>
  );
}
