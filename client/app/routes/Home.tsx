import type { Route } from "./+types/Home";
import LastPlayed from "~/components/LastPlayed";
import TopArtistsCard from "~/components/TopArtistsCard";
import { useState } from "react";
import PeriodSelector from "~/components/PeriodSelector";
import { useAppContext } from "~/providers/AppProvider";
import TopAlbumsCard from "~/components/TopAlbumsCard";
import TopGenresCard from "~/components/TopGenresCard";
import PinnedItemGrid from "~/components/PinnedItemGrid";
import DailyListensChart from "~/components/DailyListensChart";
import MoodInsightCard from "~/components/MoodInsightCard";

export function meta({}: Route.MetaArgs) {
  return [
    { title: "BauCadence" },
    { name: "description", content: "BauCadence" },
  ];
}

export default function Home() {
  const [period, setPeriod] = useState("week");

  const { homeItems } = useAppContext();

  const gradientClasses =
    "bg-linear-to-b to-(--color-bg) from-(--color-bg-secondary) to-60%";

  return (
    <main className="flex grow justify-center pb-4 w-full">
      <div className="flex-1 flex flex-col items-center gap-10 md:gap-12 min-h-0 mt-8 sm:mt-10 mx-4 sm:mx-10">
        <div className="flex flex-col items-stretch gap-10 w-full">
          <DailyListensChart />
          <PeriodSelector
            setter={setPeriod}
            current={period}
            className="self-center"
          />
          <div className="justify-center flex flex-wrap gap-10">
            {/*<PinnedItemGrid />*/}
            <TopArtistsCard period={period} />
            <TopAlbumsCard period={period} />
            <TopGenresCard
              period={period}
              limit={10}
              className="min-w-[350px] w-full max-w-[750px] 2xl:max-w-[450px]"
            />
            <MoodInsightCard period={period} />
            <LastPlayed showNowPlaying={true} limit={28} showSeeMore />
          </div>
        </div>
      </div>
    </main>
  );
}
