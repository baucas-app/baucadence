import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  apiFetch,
  type PaginatedResponse,
  type VideoCategoryCount,
  type VideoWatch,
} from "api/api";
import VideosTable from "~/components/VideosTable";

export function meta() {
  return [
    { title: "Videos | BauCadence" },
    { name: "description", content: "BauCadence" },
  ];
}

type FormatFilter = "" | "video" | "short";

const getVideos = (args: { limit: number; page: number; format: string }) =>
  apiFetch<PaginatedResponse<VideoWatch>>("/apis/web/v1/videos", args);

const getVideoCategories = () =>
  apiFetch<VideoCategoryCount[]>("/apis/web/v1/insights/video-categories", {
    period: "all_time",
  });

function VideoCategoryBreakdown({
  categories,
}: {
  categories: VideoCategoryCount[];
}) {
  const total = categories.reduce((sum, c) => sum + c.count, 0);
  if (total === 0) return null;
  const top = categories.slice(0, 8);

  return (
    <div className="card p-6 flex flex-col gap-3 w-full">
      <div className="font-medium">Categories</div>
      {top.map((c) => (
        <div key={c.category} className="flex items-center gap-3">
          <span className="w-36 shrink-0 truncate text-[13px]">
            {c.category}
          </span>
          <div className="flex-1 h-2 rounded-full bg-(--color-bg-tertiary) overflow-hidden">
            <div
              className="h-full rounded-full bg-(--color-primary)"
              style={{ width: `${(c.count / top[0].count) * 100}%` }}
            />
          </div>
          <span className="w-10 shrink-0 text-end text-[12px] color-fg-secondary">
            {c.count}
          </span>
        </div>
      ))}
    </div>
  );
}

export default function Videos() {
  const [page, setPage] = useState(1);
  const [format, setFormat] = useState<FormatFilter>("");

  const { isPending, isError, data, error } = useQuery({
    queryKey: ["videos", page, format],
    queryFn: () => getVideos({ limit: 25, page, format }),
  });

  const { data: categories } = useQuery({
    queryKey: ["insights/video-categories"],
    queryFn: getVideoCategories,
  });

  const setFormatFilter = (f: FormatFilter) => {
    setFormat(f);
    setPage(1);
  };

  return (
    <main className="flex grow justify-center pb-4 w-full">
      <div className="flex-1 flex flex-col items-center gap-8 min-h-0 mt-8 sm:mt-10 mx-4 sm:mx-10">
        <div className="flex flex-col gap-5 text-sm md:text-[16px] w-11/12 max-w-[1000px]">
          <h1>Videos</h1>

          {categories && categories.length > 0 && (
            <VideoCategoryBreakdown categories={categories} />
          )}

          <div className="flex gap-2">
            {(["", "video", "short"] as FormatFilter[]).map((f) => (
              <button
                key={f || "all"}
                className={`text-[13px] px-3 py-1.5 rounded-full ${
                  format === f
                    ? "bg-(--color-bg-secondary) color-fg"
                    : "color-fg-tertiary hover-bg-secondary"
                }`}
                onClick={() => setFormatFilter(f)}
              >
                {f === "" ? "All" : f === "video" ? "Videos" : "Shorts"}
              </button>
            ))}
          </div>

          {isPending ? (
            <p className="color-fg-secondary">Loading...</p>
          ) : isError ? (
            <p className="error">Error: {error.message}</p>
          ) : data.items.length === 0 ? (
            <p className="color-fg-secondary">
              No videos watched yet. Non-music YouTube videos show up here
              once imported from a Google Takeout watch-history export.
            </p>
          ) : (
            <>
              <div className="flex gap-15 mx-auto">
                <button
                  className="default"
                  onClick={() => setPage((p) => p - 1)}
                  disabled={page <= 1}
                >
                  Prev
                </button>
                <button
                  className="default"
                  onClick={() => setPage((p) => p + 1)}
                  disabled={!data.has_next_page}
                >
                  Next
                </button>
              </div>
              <VideosTable watches={data.items} />
              <div className="flex gap-15 mx-auto">
                <button
                  className="default"
                  onClick={() => setPage((p) => p - 1)}
                  disabled={page <= 1}
                >
                  Prev
                </button>
                <button
                  className="default"
                  onClick={() => setPage((p) => p + 1)}
                  disabled={!data.has_next_page}
                >
                  Next
                </button>
              </div>
            </>
          )}
        </div>
      </div>
    </main>
  );
}
