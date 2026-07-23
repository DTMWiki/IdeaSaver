import {
  Table,
  Checkbox,
  Typography,
  Button,
  Dropdown,
  Tag,
  Tooltip,
  Grid,
  Empty,
} from "antd";
import type { Breakpoint } from "antd";
import {
  FolderFilled,
  FileImageOutlined,
  FileTextOutlined,
  SoundOutlined,
  VideoCameraOutlined,
  FileOutlined,
  EllipsisOutlined,
  UploadOutlined,
} from "@ant-design/icons";
import type { MenuProps } from "antd";
import type { FileItem } from "@/types";
import { useFileStore } from "@/stores/fileStore";
import {
  formatBytes,
  formatDate,
  isImage,
  isAudio,
  isVideo,
  isText,
} from "@/utils/format";

const { Text } = Typography;
const TABLE_MD: Breakpoint[] = ["md"];
const TABLE_LG: Breakpoint[] = ["lg"];

interface FileTableProps {
  files: FileItem[];
  loading: boolean;
  selectedIds: Set<string>;
  onContextMenu: (e: React.MouseEvent, file: FileItem) => void;
  onPreview: (file: FileItem) => void;
  actionItems: (file: FileItem) => MenuProps["items"];
  onUpload?: () => void;
}

function getFileIcon(file: FileItem) {
  if (file.is_directory)
    return <FolderFilled style={{ color: "#faad14", fontSize: 18 }} />;
  if (isImage(file.mime_type))
    return <FileImageOutlined style={{ color: "#1677ff", fontSize: 18 }} />;
  if (isAudio(file.mime_type))
    return <SoundOutlined style={{ color: "#722ed1", fontSize: 18 }} />;
  if (isVideo(file.mime_type))
    return <VideoCameraOutlined style={{ color: "#eb2f96", fontSize: 18 }} />;
  if (isText(file.mime_type))
    return <FileTextOutlined style={{ color: "#52c41a", fontSize: 18 }} />;
  return <FileOutlined style={{ color: "#8c8c8c", fontSize: 18 }} />;
}

export default function FileTable({
  files,
  loading,
  selectedIds,
  onContextMenu,
  onPreview,
  actionItems,
  onUpload,
}: FileTableProps) {
  const { toggleSelect, navigateTo } = useFileStore();
  const screens = Grid.useBreakpoint();
  const isMobile = !screens.md;

  const handleRowClick = (file: FileItem) => {
    if (file.is_directory) {
      navigateTo(file.id, file.name);
    } else {
      onPreview(file);
    }
  };

  const columns = [
    {
      title: "",
      dataIndex: "id",
      key: "select",
      width: 40,
      render: (_: string, record: FileItem) => (
        <Checkbox
          checked={selectedIds.has(record.id)}
          onChange={() => toggleSelect(record.id)}
          onClick={(e) => e.stopPropagation()}
        />
      ),
    },
    {
      title: "名称",
      dataIndex: "name",
      key: "name",
      ellipsis: true,
      render: (_: string, record: FileItem) => (
        <div className="file-table-name-cell">
          {getFileIcon(record)}
          <div className="file-table-name-content">
            <Text className="file-table-name-text" ellipsis={{ tooltip: record.name }}>
              {record.name}
            </Text>
            {isMobile && (
              <div className="file-table-name-meta">
                {!record.is_directory && (
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {formatBytes(record.size)}
                  </Text>
                )}
                {!record.is_directory &&
                  (record.moderation_status === "banned" ? (
                    <Tooltip title={record.moderation_reason?.trim() || "管理员已封禁该文件"}>
                      <Tag color="red" style={{ marginInlineEnd: 0 }}>
                        已封禁
                      </Tag>
                    </Tooltip>
                  ) : (
                    <Tag color="green" style={{ marginInlineEnd: 0 }}>
                      正常
                    </Tag>
                  ))}
                <Text type="secondary" style={{ fontSize: 12 }}>
                  {formatDate(record.updated_at)}
                </Text>
              </div>
            )}
          </div>
        </div>
      ),
    },
    {
      title: "大小",
      dataIndex: "size",
      key: "size",
      width: 100,
      responsive: TABLE_MD,
      render: (size: number, record: FileItem) =>
        record.is_directory ? "-" : formatBytes(size),
    },
    {
      title: "状态",
      dataIndex: "moderation_status",
      key: "moderation_status",
      width: 120,
      responsive: TABLE_MD,
      render: (_: string, record: FileItem) => {
        if (record.is_directory) return "-";
        if (record.moderation_status === "banned") {
          const reason = record.moderation_reason?.trim();
          return (
            <Tooltip title={reason || "管理员已封禁该文件"}>
              <Tag color="red">已封禁</Tag>
            </Tooltip>
          );
        }
        return <Tag color="green">正常</Tag>;
      },
    },
    {
      title: "修改时间",
      dataIndex: "updated_at",
      key: "updated_at",
      width: 160,
      responsive: TABLE_LG,
      render: (date: string) => formatDate(date),
    },
    {
      title: "操作",
      key: "actions",
      width: isMobile ? 52 : 96,
      align: "center" as const,
      render: (_: unknown, record: FileItem) => (
        <Dropdown
          trigger={["click"]}
          menu={{ items: actionItems(record) }}
          placement="bottomRight"
        >
          <Button
            type="text"
            size="small"
            className="file-action-button"
            icon={<EllipsisOutlined />}
            onClick={(e) => e.stopPropagation()}
          >
            {!isMobile && "操作"}
          </Button>
        </Dropdown>
      ),
    },
  ];

  return (
    <div className="file-list-container">
      <Table
        dataSource={files}
        columns={columns}
        rowKey="id"
        loading={loading}
        pagination={false}
        size={isMobile ? "small" : "middle"}
        scroll={isMobile ? { x: 520 } : undefined}
        locale={{
          emptyText: (
            <Empty description="此文件夹为空">
              {onUpload && (
                <Button type="primary" icon={<UploadOutlined />} onClick={onUpload}>
                  上传文件
                </Button>
              )}
            </Empty>
          ),
        }}
        onRow={(record) => ({
          onClick: () => handleRowClick(record),
          onContextMenu: (e) => {
            // Prevent bubbling to dashboard background "empty area" context menu.
            e.stopPropagation()
            onContextMenu(e, record)
          },
          className: "file-table-row",
        })}
      />
    </div>
  );
}
