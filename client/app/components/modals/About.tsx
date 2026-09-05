import { useAppContext } from "~/providers/AppProvider";
import { ExternalLink } from "lucide-react";
import { Link } from "react-router";

export default function About() {
  const { currentVersion } = useAppContext();

  return (
    <div>
      <div className="w-full bg p-6 rounded-sm flex flex-col items-center gap-4">
        <div className="inline-flex items-center gap-3">
          <img
            src="/web-app-manifest-192x192.png"
            alt="BauCadence logo"
            style={{ width: 70 }}
          />
          <h5 className="text-6xl font-semibold">BauCadence</h5>
        </div>
        <div className="px-2 py-1 rounded-sm bg-secondary">
          BauCadence {currentVersion}
        </div>
        <p className="text-(--color-fg-secondary) text-sm text-center max-w-sm">
          Built on top of{" "}
          <Link
            className="text-(--color-info) hover:underline"
            to="https://github.com/gabehf/koito"
            target="_blank"
          >
            Koito
            <ExternalLink size={12} className="inline ml-0.5 mb-0.5" />
          </Link>
          , an open-source (MIT) scrobbler, extended with built-in Spotify,
          Jellyfin, and YouTube/YouTube Music sources.
        </p>
      </div>
    </div>
  );
}
