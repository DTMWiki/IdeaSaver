import { useEffect, useRef, useState, type ReactNode } from "react";
import client from "@/api/client";

interface FileThumbnailProps {
  fileId: string;
  alt: string;
  className?: string;
  fallback: ReactNode;
}

export default function FileThumbnail({
  fileId,
  alt,
  className,
  fallback,
}: FileThumbnailProps) {
  const hostRef = useRef<HTMLDivElement | null>(null);
  const [visible, setVisible] = useState(false);
  const [src, setSrc] = useState("");
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    const node = hostRef.current;
    if (!node) return undefined;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting)) {
          setVisible(true);
          observer.disconnect();
        }
      },
      { rootMargin: "120px" },
    );

    observer.observe(node);
    return () => observer.disconnect();
  }, []);

  useEffect(() => {
    if (!visible || src || failed) return undefined;

    const controller = new AbortController();
    let objectURL = "";

    async function loadThumbnail() {
      try {
        const response = await client.get(`/files/${fileId}/preview`, {
          params: { thumb: 1 },
          responseType: "blob",
          signal: controller.signal,
        });
        objectURL = URL.createObjectURL(response.data);
        setSrc(objectURL);
      } catch {
        if (!controller.signal.aborted) {
          setFailed(true);
        }
      }
    }

    void loadThumbnail();

    return () => {
      controller.abort();
      if (objectURL) {
        URL.revokeObjectURL(objectURL);
      }
    };
  }, [failed, fileId, src, visible]);

  return (
    <div ref={hostRef} style={{ width: "100%", height: "100%" }}>
      {src ? (
        <img src={src} alt={alt} loading="lazy" className={className} />
      ) : (
        fallback
      )}
    </div>
  );
}
