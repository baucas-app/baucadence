# Scrobble sources

This is a fork of [Koito](https://github.com/gabehf/koito) (MIT licensed)
with three built-in scrobble sources added directly to the Go backend, so
the whole thing ships as one binary / one Docker container:

- **Spotify** — official Web API, polls every 60s by default. Reliable.
- **Jellyfin** — official Webhook plugin pushes events to us. Reliable.
- **YouTube / YouTube Music** — no official API exists for this, so it
  authenticates with a copied browser cookie and talks to Google's
  undocumented internal "InnerTube" API. This can break if YouTube changes
  its internal response format; if listens stop appearing, check the
  container logs first.

All connectors write directly into Koito's catalog via the same internal
function the ListenBrainz API endpoint uses, so everything shows up in the
normal Koito stats pages, distinguishable by the "Client" tag (Spotify,
Jellyfin, YouTube, YouTube Music).

## 1. Required for any source: encryption key

Source credentials (Spotify refresh token, YouTube cookie) are encrypted at
rest with AES-256-GCM. Set:

```
SCROBLLI_ENCRYPTION_KEY=<random string, e.g. output of `openssl rand -base64 32`>
```

If this is not set, no source will start, even if otherwise configured
(check the logs for a warning).

Losing/changing this key does not affect your listening history — only
stored source credentials become unreadable, and you'll need to
reconnect Spotify / re-paste the YouTube cookie.

## 2. Spotify

1. Go to the [Spotify Developer Dashboard](https://developer.spotify.com/dashboard)
   and log in (Premium account required to create apps).
2. Create an app. As the **Redirect URI**, enter:
   `https://<your-domain>/apis/sources/v1/spotify/callback`
3. Copy the **Client ID** and **Client Secret** from the app's Settings page.
4. Set the environment variables:
   ```
   SCROBLLI_SPOTIFY_CLIENT_ID=<client id>
   SCROBLLI_SPOTIFY_CLIENT_SECRET=<client secret>
   SCROBLLI_SPOTIFY_REDIRECT_URI=https://<your-domain>/apis/sources/v1/spotify/callback
   ```
5. Restart the container, log into the app, open **Settings → Sources**,
   and click **Connect Spotify**.

Optional: `SCROBLLI_SPOTIFY_POLL_SECONDS` (default 60, minimum 15).

## 3. Jellyfin

1. In Jellyfin, install the official
   [Webhook plugin](https://github.com/jellyfin/jellyfin-plugin-webhook)
   from the plugin catalog.
2. Set a secret of your choosing on the app server:
   ```
   SCROBLLI_JELLYFIN_WEBHOOK_SECRET=<any random string>
   ```
3. In the Webhook plugin, add a **Generic** destination:
   - **Webhook Url**: `https://<your-domain>/apis/sources/v1/jellyfin/webhook?token=<the secret above>`
   - **Notification Type**: Playback Stop
   - **Item Type**: Songs
   - **Template**:
     ```json
     {
       "NotificationType": "{{NotificationType}}",
       "ItemType": "{{ItemType}}",
       "Name": "{{Name}}",
       "Album": "{{Album}}",
       "Artist": "{{Artist}}",
       "RunTimeTicks": {{RunTimeTicks}},
       "PlaybackPositionTicks": {{PlaybackPositionTicks}}
     }
     ```
   (The exact same fields and template are also shown in **Settings → Sources**
   in the app itself.)

There's no per-user Jellyfin login: all Jellyfin plays are attributed to
the app's admin user, since this is meant for a single-listener setup.

## 4. YouTube / YouTube Music

There is no official, supported way to read your YouTube listening/watch
history programmatically — this connector authenticates the same way a
browser does, using your session cookie. It cannot use OAuth/"Sign in with
Google" because Google does not grant third-party apps API access to this
data at all.

1. Log into `music.youtube.com` in a normal browser window.
2. Open Developer Tools → **Network** tab, reload the page.
3. Click any request made to `music.youtube.com`, and in its request
   headers find **Cookie**. Copy the entire value.
4. In the app, open **Settings → Sources → YouTube / YouTube Music**,
   paste the cookie, and click **Connect**.

Notes:

- This cookie typically expires after **1–3 months**; when it does,
  listens will silently stop appearing and you'll need to repeat the
  steps above.
- By default only YouTube Music is tracked. To also track regular
  YouTube video watches (stored as tracks with the channel as "artist"),
  set `SCROBLLI_YTMUSIC_TRACK_VIDEOS=true` on the server.
- Because YouTube's history pages don't expose an exact play timestamp
  per item, YouTube/YouTube Music listens are recorded at the time they
  are *detected* (next poll after you listened), not the exact moment
  you pressed play. Poll interval: `SCROBLLI_YTMUSIC_POLL_SECONDS`
  (default 120, minimum 15).
- This is the most fragile of the three sources (see the warning at the
  top of this file). If it stops working after a YouTube update, the
  fix lives in `internal/sources/youtube/`.

## Deploying

```
docker compose up -d --build
```

On a Synology NAS, this is the same as any other Container Manager
"Project": point it at this folder (or paste the compose file), fill in
the environment variables above, and build/start.
