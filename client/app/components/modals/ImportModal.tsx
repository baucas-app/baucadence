import SubHeader from "../primitives/SubHeader";
import OverallImportProgress from "../OverallImportProgress";
import ImportProgressRow from "../ImportProgressRow";
import ImportSourceUploader from "../ImportSourceUploader";
import { useImportStatus } from "~/hooks/useImportStatus";

export default function ImportModal() {
  const summary = useImportStatus();

  return (
    <div className="">
      <SubHeader>Import Listening History</SubHeader>
      <p className="mb-4 opacity-80">
        Upload a listening-history export to backfill old listens. You can
        upload the whole .zip as downloaded, or a single already-extracted
        export file. Import runs in the background after upload; it can take
        a while for large histories.
      </p>
      <div className="flex flex-col gap-4 w-full">
        <ImportSourceUploader
          title="Spotify"
          description={`Your "Extended Streaming History" request from Spotify's privacy settings page.`}
          accept=".zip,.json"
          onUploaded={summary.refetch}
        />
        <ImportSourceUploader
          title="YouTube / YouTube Music"
          description={`Your "YouTube and YouTube Music" export from Google Takeout, JSON or HTML format (watch-history.json / watch-history.html).`}
          accept=".zip,.json,.html"
          onUploaded={summary.refetch}
        />
        <ImportSourceUploader
          title="Other (Maloja / Last.fm / ListenBrainz / Koito)"
          description="An export from one of these other supported services."
          accept=".zip,.json"
          onUploaded={summary.refetch}
        />
      </div>
      {summary.files.length > 0 && (
        <div className="flex flex-col gap-3 mt-5 w-full">
          <OverallImportProgress summary={summary} bordered />
          {summary.files.map((f) => (
            <ImportProgressRow key={f.filename} progress={f} />
          ))}
        </div>
      )}
    </div>
  );
}
