import { useRef, useState } from "react";
import { uploadImportFile } from "api/api";
import { AsyncButton } from "../AsyncButton";
import SubHeader from "../primitives/SubHeader";

export default function ImportModal() {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [loading, setLoading] = useState(false);
  const [err, setError] = useState<string>();
  const [startedFiles, setStartedFiles] = useState<string[]>();

  const handleUpload = () => {
    if (!selectedFile) {
      setError("choose a file first");
      return;
    }
    setError(undefined);
    setStartedFiles(undefined);
    setLoading(true);
    uploadImportFile(selectedFile)
      .then((r) => {
        setStartedFiles(r.started_files);
        setSelectedFile(null);
        if (fileInputRef.current) fileInputRef.current.value = "";
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  };

  return (
    <div className="">
      <SubHeader>Import Listening History</SubHeader>
      <p className="mb-4 opacity-80">
        Upload a listening-history export to backfill old listens — a
        Spotify "Extended Streaming History" export, a Google Takeout
        YouTube/YouTube Music export, or a Maloja/LastFM/ListenBrainz/Koito
        export. You can upload the whole .zip as downloaded, or a single
        already-extracted export file. Import runs in the background after
        upload; it can take a while for large histories.
      </p>
      <div className="flex flex-col gap-3 bg-secondary p-5 rounded-lg w-3/5">
        <input
          ref={fileInputRef}
          type="file"
          accept=".zip,.json"
          onChange={(e) => setSelectedFile(e.target.files?.[0] ?? null)}
          className="bg-(--color-bg) p-3 rounded-md"
        />
        <AsyncButton loading={loading} onClick={handleUpload}>
          Upload &amp; Import
        </AsyncButton>
      </div>
      {err && <p className="error mt-3">{err}</p>}
      {startedFiles && (
        <p className="mt-3">
          Import started for: {startedFiles.join(", ")}. Check back later —
          it runs in the background.
        </p>
      )}
    </div>
  );
}
