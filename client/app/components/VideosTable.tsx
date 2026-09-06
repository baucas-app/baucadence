import Image from "./primitives/Image";
import { type VideoWatch } from "api/api";
import { timeSince } from "~/utils/utils";

interface VideosTableProps {
  watches: VideoWatch[];
}

export default function VideosTable({ watches }: VideosTableProps) {
  const imgColSizeClasses = "py-3 min-w-14 sm:min-w-18";
  const timeColClasses = "text-(--color-fg-tertiary) pr-2 sm:pr-4 sm:text-sm";

  return (
    <table className="table border-collapse mt-6 w-full">
      <tbody>
        {watches.map((item) => (
          <tr
            key={`video_watch_${item.video.id}_${item.time}`}
            className="group border-b border-(--color-bg-tertiary) relative last:border-b-0"
          >
            <td className={imgColSizeClasses}>
              <a
                href={`https://www.youtube.com/watch?v=${item.video.youtube_id}`}
                target="_blank"
                rel="noreferrer"
              >
                <Image
                  src={item.video.thumbnail}
                  size={64}
                  alt={item.video.title}
                  className="aspect-video object-cover"
                  lazy
                />
              </a>
            </td>
            <td className="max-w-0 w-full px-2 py-2">
              <div>
                <span className="color-fg-secondary">
                  {item.video.channel_name}
                </span>
                {" — "}
                <a
                  className="hover:text-(--color-fg-secondary)"
                  href={`https://www.youtube.com/watch?v=${item.video.youtube_id}`}
                  target="_blank"
                  rel="noreferrer"
                >
                  {item.video.title}
                </a>
              </div>
              <div className="flex items-center gap-2 mt-0.5">
                {item.video.category && (
                  <span className="text-[11px] color-fg-tertiary">
                    {item.video.category}
                  </span>
                )}
                {item.video.format === "short" && (
                  <span className="text-[10px] uppercase tracking-wide color-fg-secondary bg-(--color-bg-tertiary) rounded-full px-1.5 py-0.5">
                    Short
                  </span>
                )}
              </div>
            </td>
            <td
              className={`text-end whitespace-nowrap ${timeColClasses}`}
              title={new Date(item.time).toString()}
            >
              {timeSince(new Date(item.time))}
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
