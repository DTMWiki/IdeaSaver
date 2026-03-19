import { useEffect, useState } from "react";
import { Modal, Spin, Typography, Image, Alert } from "antd";
import type { FileItem } from "@/types";
import { isImage, isAudio, isText } from "@/utils/format";

const { Text, Paragraph } = Typography;

interface FilePreviewProps {
  file: FileItem | null;
  onClose: () => void;
}

interface PreviewState {
  fileId: string;
  textContent: string | null;
  objectURL: string;
}

export default function FilePreview({ file, onClose }: FilePreviewProps) {
  const [previewState, setPreviewState] = useState<PreviewState>({
    fileId: "",
    textContent: null,
    objectURL: "",
  });
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!file || file.is_directory) {
      return undefined;
    }
    if (
      !isText(file.mime_type) &&
      !isImage(file.mime_type) &&
      !isAudio(file.mime_type)
    ) {
      return undefined;
    }

    let cancelled = false;
    let objectURL = "";
    const fileID = file.id;
    const mimeType = file.mime_type;

    async function loadPreview() {
      setLoading(true);
      try {
        const token = localStorage.getItem("token") || "";
        const response = await fetch(`/api/files/${fileID}/preview`, {
          headers: token ? { Authorization: `Bearer ${token}` } : undefined,
        });

        if (!response.ok) {
          throw new Error(`preview failed with status ${response.status}`);
        }

        if (isText(mimeType)) {
          const text = await response.text();
          if (!cancelled) {
            setPreviewState({
              fileId: fileID,
              textContent: text,
              objectURL: "",
            });
          }
          return;
        }

        const blob = await response.blob();
        objectURL = URL.createObjectURL(blob);
        if (!cancelled) {
          setPreviewState({ fileId: fileID, textContent: null, objectURL });
        }
      } catch {
        if (!cancelled) {
          setPreviewState({
            fileId: fileID,
            textContent: isText(mimeType) ? "无法加载文件内容" : null,
            objectURL: "",
          });
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    void loadPreview();

    return () => {
      cancelled = true;
      if (objectURL) {
        URL.revokeObjectURL(objectURL);
      }
    };
  }, [file]);

  if (!file) return null;

  const isCurrentPreview = previewState.fileId === file.id;
  const textContent = isCurrentPreview ? previewState.textContent : null;
  const previewURL = isCurrentPreview ? previewState.objectURL : "";

  const renderContent = () => {
    if (file.moderation_status === "banned") {
      return (
        <div style={{ padding: 20 }}>
          <Alert
            type="error"
            showIcon
            message="该文件已被管理员封禁"
            description={
              file.moderation_reason ||
              "当前无法预览或访问此文件，可在文件列表发起申诉工单。"
            }
          />
        </div>
      );
    }

    if (loading) {
      return (
        <div style={{ textAlign: "center", padding: 40 }}>
          <Spin size="large" />
        </div>
      );
    }

    if (isImage(file.mime_type)) {
      return (
        <div style={{ textAlign: "center", padding: 16 }}>
          <Image
            src={previewURL}
            alt={file.name}
            style={{ maxHeight: "60vh", objectFit: "contain" }}
            fallback="data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mN88P/BfwAJhAPkGz0hkgAAAABJRU5ErkJggg=="
          />
        </div>
      );
    }

    if (isAudio(file.mime_type)) {
      return (
        <div style={{ textAlign: "center", padding: 40 }}>
          <audio
            controls
            src={previewURL}
            style={{ width: "100%", maxWidth: 500 }}
          >
            您的浏览器不支持音频播放
          </audio>
        </div>
      );
    }

    if (isText(file.mime_type) && textContent !== null) {
      return (
        <div style={{ maxHeight: "60vh", overflow: "auto" }}>
          <pre
            style={{
              padding: 16,
              background: "var(--color-bg-spotlight)",
              borderRadius: "var(--border-radius-sm)",
              fontSize: 13,
              lineHeight: 1.6,
              whiteSpace: "pre-wrap",
              wordBreak: "break-all",
              margin: 0,
            }}
          >
            <code>{textContent}</code>
          </pre>
        </div>
      );
    }

    return (
      <div style={{ textAlign: "center", padding: 40 }}>
        <Paragraph type="secondary">此文件类型暂不支持预览</Paragraph>
        <Text type="secondary" style={{ fontSize: 12 }}>
          MIME: {file.mime_type || "未知"}
        </Text>
      </div>
    );
  };

  return (
    <Modal
      title={file.name}
      open={!!file}
      onCancel={onClose}
      footer={null}
      width={720}
      destroyOnClose
    >
      {renderContent()}
    </Modal>
  );
}
