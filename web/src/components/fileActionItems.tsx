import type { MenuProps } from "antd";
import {
  CopyOutlined,
  DeleteOutlined,
  EditOutlined,
  EyeOutlined,
  FlagOutlined,
  FolderOpenOutlined,
  LinkOutlined,
  ScissorOutlined,
  ShareAltOutlined,
} from "@ant-design/icons";
import type { FileItem } from "@/types";

export interface FileActionHandlers {
  onOpen: (file: FileItem) => void;
  onPreview: (file: FileItem) => void;
  onCopyLink: (file: FileItem) => void;
  onCopyMarkdown: (file: FileItem) => void;
  onShare: (file: FileItem) => void;
  onAppeal: (file: FileItem) => void;
  onRename: (file: FileItem) => void;
  onCopy: (file: FileItem) => void;
  onCut: (file: FileItem) => void;
  onDelete: (file: FileItem) => void;
}

export function buildFileActionItems(
  file: FileItem,
  handlers: FileActionHandlers,
): MenuProps["items"] {
  const items: NonNullable<MenuProps["items"]> = [];

  if (file.is_directory) {
    items.push({
      key: "open",
      icon: <FolderOpenOutlined />,
      label: "打开",
      onClick: () => handlers.onOpen(file),
    });
  } else {
    items.push({
      key: "preview",
      icon: <EyeOutlined />,
      label: "预览",
      onClick: () => handlers.onPreview(file),
    });
    items.push({
      key: "links",
      icon: <LinkOutlined />,
      label: "链接",
      children: [
        {
          key: "copy-markdown",
          icon: <CopyOutlined />,
          label: "复制 Markdown",
          onClick: () => handlers.onCopyMarkdown(file),
        },
        {
          key: "copy-link",
          icon: <LinkOutlined />,
          label: "复制直链",
          onClick: () => handlers.onCopyLink(file),
        },
      ],
    });
    if (file.moderation_status === "banned") {
      items.push({
        key: "appeal",
        icon: <FlagOutlined />,
        label: "提交申诉",
        onClick: () => handlers.onAppeal(file),
      });
    } else {
      items.push({
        key: "share",
        icon: <ShareAltOutlined />,
        label: "分享",
        onClick: () => handlers.onShare(file),
      });
    }
  }

  items.push({ type: "divider" });
  items.push({
    key: "rename",
    icon: <EditOutlined />,
    label: "重命名",
    onClick: () => handlers.onRename(file),
  });
  items.push({
    key: "copy",
    icon: <CopyOutlined />,
    label: "复制",
    onClick: () => handlers.onCopy(file),
  });
  items.push({
    key: "cut",
    icon: <ScissorOutlined />,
    label: "剪切",
    onClick: () => handlers.onCut(file),
  });
  items.push({ type: "divider" });
  items.push({
    key: "delete",
    icon: <DeleteOutlined />,
    label: "删除",
    danger: true,
    onClick: () => handlers.onDelete(file),
  });

  return items;
}
