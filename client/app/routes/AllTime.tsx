import AllTimeStats from "~/components/AllTimeStats";
import ActivityGrid from "~/components/ActivityGrid";
import TopTracks from "~/components/TopTracks";

export function meta() {
  return [
    { title: "All Time | BauCadence" },
    { name: "description", content: "BauCadence" },
  ];
}

export default function AllTime() {
  return (
    <main className="flex grow justify-center pb-4 w-full">
      <div className="flex-1 flex flex-col items-center gap-10 md:gap-12 min-h-0 mt-8 sm:mt-10 mx-4 sm:mx-10">
        <div className="flex flex-col lg:flex-row gap-10 md:gap-20">
          <AllTimeStats />
          <ActivityGrid configurable />
        </div>
        <TopTracks
          period="all_time"
          limit={10}
          showSeeMore
          className="w-full max-w-[750px]"
        />
      </div>
    </main>
  );
}
