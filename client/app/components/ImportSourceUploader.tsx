import { useRef, useState } from "react";
import { uploadImportFile } from "api/api";
import { AsyncButton } from "./AsyncButton";

interface Props {
  title: string;
  description: string;
  accept: string;
  onUploaded: () => void;
}

export default function ImportSourceUploader({
  title,
  description,
  accept,
  onUploaded,
}: Props) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [loading, setLoading] = useState(false);
  const [err, setError] = useState<string>();

  const handleUpload = () => {
    if (!selectedFile) {
      setError("choose a file first");
      return;
    }
    setError(undefined);
    setLoading(true);
    uploadImportFile(selectedFile)
      .then(() => {
        setSelectedFile(null);
        if (fileInputRef.current) fileInputRef.current.value = "";
        onUploaded();
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false));
  };

  return (
    <div className="flex flex-col gap-3 bg-secondary p-5 rounded-lg w-full">
      <div>
        <div className="font-medium">{title}</div>
        <p className="text-[13px] opacity-80 mt-1">{description}</p>
      </div>
      <input
        ref={fileInputRef}
        type="file"
        accept={accept}
        onChange={(e) => setSelectedFile(e.target.files?.[0] ?? null)}
        className="bg-(--color-bg) p-3 rounded-md"
      />
      <AsyncButton loading={loading} onClick={handleUpload}>
        Upload &amp; Import
      </AsyncButton>
      {err && <p className="error">{err}</p>}
    </div>
  );
}
