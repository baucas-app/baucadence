import { useQuery } from "@tanstack/react-query";
import { apiFetch, type VideoChannelRank } from "api/api";
import CardHeader from "./primitives/CardHeader";
import Image from "./primitives/Image";

interface Props {
  period: string;
}

const getTopVideoChannels = (period: string) =>
  apiFetch<VideoChannelRank[]>("/apis/web/v1/insights/video-channels", {
    period,
  });

const title = "Top channels";

const channelUrl = (c: VideoChannelRank) =>
  c.channel_id
    ? `https://www.youtube.com/channel/${c.channel_id}`
    : undefined;

export default function TopVideoChannelsCard({ period }: Props) {
  const { isPending, isError, data, error } = useQuery({
    queryKey: ["insights/video-channels", period],
    queryFn: () => getTopVideoChannels(period),
  });

  if (isPending) {
    return <TopVideoChannelsCardSkeleton />;
  } else if (isError) {
    return (
      <div className="w-[350px]">
        <CardHeader>{title}</CardHeader>
        <p className="error">Error: {error.message}</p>
      </div>
    );
  }

  if (!data[0]) {
    return (
      <div className="w-[350px]">
        <CardHeader>{title}</CardHeader>
        <p className="mt-6">Nothing to show</p>
      </div>
    );
  }

  const [first, ...rest] = data;

  return (
    <div>
      <CardHeader>{title}</CardHeader>
      <div className="max-w-[350px] card">
        <div className="relative">
          <a href={channelUrl(first)} target="_blank" rel="noreferrer">
            <img
              src={first.thumbnail}
              alt={first.channel_name}
              className="w-full aspect-square object-cover"
              style={{
                borderRadius: "var(--border-radius) var(--border-radius) 0 0",
              }}
            />
          </a>
          <div
            className="absolute inset-0 bg-linear-to-t"
            style={{
              backgroundImage: `linear-gradient(to top,
              var(--color-bg-secondary) 0%,
              color-mix(in srgb, var(--color-bg-secondary) 99%, transparent) 5%,
              color-mix(in srgb, var(--color-bg-secondary) 95%, transparent) 12%,
              color-mix(in srgb, var(--color-bg-secondary) 86%, transparent) 20%,
              color-mix(in srgb, var(--color-bg-secondary) 72%, transparent) 28%,
              color-mix(in srgb, var(--color-bg-secondary) 55%, transparent) 36%,
              color-mix(in srgb, var(--color-bg-secondary) 37%, transparent) 44%,
              color-mix(in srgb, var(--color-bg-secondary) 22%, transparent) 51%,
              color-mix(in srgb, var(--color-bg-secondary) 11%, transparent) 57%,
              color-mix(in srgb, var(--color-bg-secondary) 4%, transparent) 61%,
              color-mix(in srgb, var(--color-bg-secondary) 1%, transparent) 63.5%,
              transparent 65%
              )`,
              borderRadius: "var(--border-radius) var(--border-radius) 0 0",
            }}
          />
          <div className="absolute bottom-8 left-5">
            <a href={channelUrl(first)} target="_blank" rel="noreferrer">
              <h5 className="text-3xl font-semibold line-clamp-3 wrap-anywhere text-shadow-lg">
                {first.channel_name}
              </h5>
            </a>
            <div className="color-fg-secondary">
              {first.watch_count} watches
            </div>
          </div>
        </div>
        <div className="flex flex-col items-start">
          {rest.map((c) => (
            <div className="flex items-center gap-3 px-6 pb-6 w-full" key={c.channel_name}>
              <a
                href={channelUrl(c)}
                target="_blank"
                rel="noreferrer"
                className="shrink-0"
              >
                <Image
                  src={c.thumbnail}
                  size={56}
                  alt={c.channel_name}
                  className="aspect-square object-cover rounded-(--border-radius)"
                />
              </a>
              <div className="min-w-0">
                <a href={channelUrl(c)} target="_blank" rel="noreferrer">
                  <div className="line-clamp-2 wrap-anywhere hover:text-(--color-fg-secondary)">
                    {c.channel_name}
                  </div>
                </a>
                <div className="color-fg-secondary text-[12px] sm:text-[14px]">
                  {c.watch_count} watches
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

function TopVideoChannelsCardSkeleton() {
  return (
    <div className="w-[350px] flex flex-col gap-3">
      <CardHeader>{title}</CardHeader>
      <div className="w-full aspect-square bg animate-pulse rounded-(--border-radius)" />
    </div>
  );
}
