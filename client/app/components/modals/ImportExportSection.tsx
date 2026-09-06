import { useState } from "react";
import ExportModal from "./ExportModal";
import ImportModal from "./ImportModal";

interface Props {
  initialView?: "export" | "import";
}

export default function ImportExportSection({ initialView }: Props) {
  const [view, setView] = useState<"export" | "import">(
    initialView ?? "export",
  );

  const btnClasses = (active: boolean) =>
    `px-4 py-1.5 rounded-md text-sm border-(--color-border) ${
      active
        ? "bg-(--color-bg-secondary) border"
        : "color-fg-secondary hover-bg-secondary"
    }`;

  return (
    <div>
      <div className="flex gap-2 mb-6">
        <button
          className={btnClasses(view === "export")}
          onClick={() => setView("export")}
        >
          Export
        </button>
        <button
          className={btnClasses(view === "import")}
          onClick={() => setView("import")}
        >
          Import
        </button>
      </div>
      {view === "export" ? <ExportModal /> : <ImportModal />}
    </div>
  );
}
