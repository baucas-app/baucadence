import { Settings } from "lucide-react";
import SettingsModal from "../modals/SettingsModal";
import SidebarItem from "./SidebarItem";
import { useEffect } from "react";
import { useAppContext } from "~/providers/AppProvider";

interface Props {
  size: number;
}

export default function SidebarSettings({ size }: Props) {
  const { updateAvailable, settingsOpen, setSettingsOpen, openSettings } =
    useAppContext();

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      const active = document.activeElement;
      const isTyping =
        active &&
        (active.tagName === "INPUT" ||
          active.tagName === "TEXTAREA" ||
          (active as HTMLElement).isContentEditable);

      if (!isTyping && e.key === "\\") {
        e.preventDefault();
        setSettingsOpen(!settingsOpen);
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [settingsOpen]);

  return (
    <SidebarItem
      space={30}
      keyHint="\"
      name="Settings"
      onClick={() => openSettings()}
      modal={<SettingsModal open={settingsOpen} setOpen={setSettingsOpen} />}
    >
      <Settings size={size} />
      {updateAvailable && (
        <div className="h-1.5 w-1.5 rounded-full bg-(--color-info) absolute top-1 right-1" />
      )}
    </SidebarItem>
  );
}
