import { useEffect, useMemo, useRef } from "react";
import { Menu } from "antd";
import type { MenuProps } from "antd";
import type { FileItem } from "@/types";

interface FileContextMenuProps {
  x: number;
  y: number;
  file: FileItem | null;
  onClose: () => void;
  fileItems: MenuProps["items"];
  backgroundItems: MenuProps["items"];
}

export default function FileContextMenu({
  x,
  y,
  file,
  onClose,
  fileItems,
  backgroundItems,
}: FileContextMenuProps) {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClick = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        onClose();
      }
    };
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, [onClose]);

  const adjustedX = Math.min(x, window.innerWidth - 220);
  const adjustedY = Math.min(y, window.innerHeight - 320);

  const items = useMemo(() => {
    return (file ? fileItems : backgroundItems) || [];
  }, [backgroundItems, file, fileItems]);

  return (
    <div
      ref={ref}
      className="file-context-menu"
      style={{ left: adjustedX, top: adjustedY }}
    >
      <Menu
        items={items}
        selectable={false}
        onClick={onClose}
        style={{
          borderRadius: 14,
          boxShadow: "0 18px 48px rgba(15, 23, 42, 0.18)",
          border: "1px solid rgba(148, 163, 184, 0.22)",
          minWidth: 188,
          padding: 6,
          backdropFilter: "blur(14px)",
        }}
      />
    </div>
  );
}
