import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  getSpotifyStatus,
  disconnectSpotify,
  spotifyAuthorizeUrl,
  getYoutubeStatus,
  connectYoutube,
  disconnectYoutube,
} from "api/api";
import { useState } from "react";
import { Copy, Check } from "lucide-react";
import SubHeader from "../primitives/SubHeader";
import { AsyncButton } from "../AsyncButton";

const JELLYFIN_TEMPLATE = `{
  "NotificationType": "{{NotificationType}}",
  "ItemType": "{{ItemType}}",
  "Name": "{{Name}}",
  "Album": "{{Album}}",
  "Artist": "{{Artist}}",
  "RunTimeTicks": {{RunTimeTicks}},
  "PlaybackPositionTicks": {{PlaybackPositionTicks}}
}`;

function CopyField({ value, label }: { value: string; label?: string }) {
  const [copied, setCopied] = useState(false);
  const handleCopy = () => {
    navigator.clipboard?.writeText(value).catch(() => {});
    setCopied(true);
    setTimeout(() => setCopied(false), 1200);
  };
  return (
    <div className="flex gap-2">
      <pre className="bg p-3 rounded-md flex-grow overflow-x-auto text-sm whitespace-pre-wrap select-text">
        {value}
      </pre>
      <button
        onClick={handleCopy}
        title={label ? `Copy ${label}` : "Copy"}
        className="large-button px-4 rounded-md self-start"
      >
        {copied ? <Check size={16} /> : <Copy size={16} />}
      </button>
    </div>
  );
}

function StatusBadge({ connected }: { connected: boolean }) {
  return (
    <span
      className={`inline-flex items-center gap-2 text-sm px-2 py-1 rounded-md ${
        connected
          ? "bg-(--color-success-bg,transparent) text-(--color-success,inherit)"
          : "text-(--color-fg-secondary)"
      }`}
    >
      <span
        className={`h-2 w-2 rounded-full ${connected ? "bg-green-500" : "bg-(--color-fg-secondary)"}`}
      />
      {connected ? "Connected" : "Not connected"}
    </span>
  );
}

function SpotifySection() {
  const queryClient = useQueryClient();
  const [loading, setLoading] = useState(false);

  const { data, isPending } = useQuery({
    queryKey: ["source-status", "spotify"],
    queryFn: getSpotifyStatus,
  });

  const handleConnect = () => {
    const popup = window.open(
      spotifyAuthorizeUrl(),
      "spotify-connect",
      "width=500,height=700",
    );
    const timer = setInterval(() => {
      if (popup?.closed) {
        clearInterval(timer);
        queryClient.invalidateQueries({ queryKey: ["source-status", "spotify"] });
      }
    }, 800);
  };

  const handleDisconnect = () => {
    setLoading(true);
    disconnectSpotify().finally(() => {
      setLoading(false);
      queryClient.invalidateQueries({ queryKey: ["source-status", "spotify"] });
    });
  };

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <SubHeader className="!mb-0">Spotify</SubHeader>
        {!isPending && <StatusBadge connected={!!data?.connected} />}
      </div>
      <p className="text-sm text-(--color-fg-secondary)">
        Automatically tracks what you play on Spotify, on any device signed
        into your account.
      </p>
      {data?.connected ? (
        <AsyncButton loading={loading} onClick={handleDisconnect} confirm danger>
          Disconnect
        </AsyncButton>
      ) : (
        <button onClick={handleConnect} className="large-button px-4 py-2 rounded-md w-fit">
          Connect Spotify
        </button>
      )}
    </div>
  );
}

function YoutubeSection() {
  const queryClient = useQueryClient();
  const [cookie, setCookie] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string>();

  const { data, isPending } = useQuery({
    queryKey: ["source-status", "youtube"],
    queryFn: getYoutubeStatus,
  });

  const handleConnect = () => {
    setError(undefined);
    if (!cookie.trim()) {
      setError("paste your YouTube cookie first");
      return;
    }
    setLoading(true);
    connectYoutube(cookie.trim())
      .then(() => {
        setCookie("");
        queryClient.invalidateQueries({ queryKey: ["source-status", "youtube"] });
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  };

  const handleDisconnect = () => {
    setLoading(true);
    disconnectYoutube().finally(() => {
      setLoading(false);
      queryClient.invalidateQueries({ queryKey: ["source-status", "youtube"] });
    });
  };

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <SubHeader className="!mb-0">YouTube / YouTube Music</SubHeader>
        {!isPending && <StatusBadge connected={!!data?.connected} />}
      </div>
      <p className="text-sm text-(--color-fg-secondary)">
        There is no official login for this: YouTube does not offer a
        supported way to read your listening/watch history automatically.
        Instead, paste your browser's YouTube session cookie below. It is
        stored encrypted and only used to check your history in the
        background.
      </p>
      <ol className="text-sm text-(--color-fg-secondary) list-decimal ml-5 flex flex-col gap-1">
        <li>Open music.youtube.com in your browser and make sure you're logged in.</li>
        <li>Open Developer Tools → Network tab, reload the page.</li>
        <li>
          Click any request to music.youtube.com, find the{" "}
          <code>Cookie</code> request header, and copy its full value.
        </li>
        <li>Paste it below.</li>
      </ol>
      {data?.connected ? (
        <AsyncButton loading={loading} onClick={handleDisconnect} confirm danger>
          Disconnect
        </AsyncButton>
      ) : (
        <div className="flex flex-col gap-2">
          <textarea
            className="fg bg rounded-md p-3 text-sm font-mono h-24"
            placeholder="Paste your YouTube cookie header here"
            value={cookie}
            onChange={(e) => setCookie(e.target.value)}
          />
          <AsyncButton loading={loading} onClick={handleConnect}>
            Connect YouTube
          </AsyncButton>
          {error && <p className="error">{error}</p>}
        </div>
      )}
      <p className="text-xs text-(--color-fg-secondary)">
        Regular YouTube video watches are only tracked in addition to music
        if the server admin enabled it (SCROBLLI_YTMUSIC_TRACK_VIDEOS=true).
        Cookies expire every 1–3 months and will need to be pasted again
        when that happens.
      </p>
    </div>
  );
}

function JellyfinSection() {
  const webhookUrl =
    typeof window !== "undefined"
      ? `${window.location.origin}/apis/sources/v1/jellyfin/webhook?token=YOUR_SECRET`
      : "/apis/sources/v1/jellyfin/webhook?token=YOUR_SECRET";

  return (
    <div className="flex flex-col gap-3">
      <SubHeader className="!mb-0">Jellyfin</SubHeader>
      <p className="text-sm text-(--color-fg-secondary)">
        Jellyfin is set up by a server admin, not per-user. It requires the{" "}
        <code>SCROBLLI_JELLYFIN_WEBHOOK_SECRET</code> environment variable to
        be set on the server, and the official{" "}
        <a
          className="underline"
          href="https://github.com/jellyfin/jellyfin-plugin-webhook"
          target="_blank"
          rel="noreferrer"
        >
          Jellyfin Webhook plugin
        </a>{" "}
        installed on your Jellyfin server.
      </p>
      <div>
        <p className="text-sm mb-1">
          1. In the Webhook plugin, add a generic destination pointing at
          (replace YOUR_SECRET with the value of
          SCROBLLI_JELLYFIN_WEBHOOK_SECRET):
        </p>
        <CopyField value={webhookUrl} label="webhook URL" />
      </div>
      <div>
        <p className="text-sm mb-1">
          2. Enable the "Playback Stop" notification type for Audio items,
          and set the message template to:
        </p>
        <CopyField value={JELLYFIN_TEMPLATE} label="webhook template" />
      </div>
    </div>
  );
}

export default function SourcesModal() {
  return (
    <div className="flex flex-col gap-8">
      <SpotifySection />
      <hr className="border-(--color-border)" />
      <YoutubeSection />
      <hr className="border-(--color-border)" />
      <JellyfinSection />
    </div>
  );
}
