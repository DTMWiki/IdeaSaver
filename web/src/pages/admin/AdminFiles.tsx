import { useCallback, useEffect, useState } from "react";
import {
  Table,
  Button,
  Typography,
  Empty,
  Popconfirm,
  App,
  Pagination,
  Tag,
  Space,
  Modal,
  Input,
  Tooltip,
  Grid,
} from "antd";
import type { Breakpoint } from "antd";
import {
  DeleteOutlined,
  StopOutlined,
  CheckCircleOutlined,
  EyeOutlined,
  LinkOutlined,
  ReloadOutlined,
  DownloadOutlined,
} from "@ant-design/icons";
import type { FileItem } from "@/types";
import {
  listAllFiles,
  adminDeleteFile,
  adminBanFile,
  adminUnbanFile,
} from "@/api/admin";
import { getFileURL } from "@/api/files";
import FilePreview from "@/components/FilePreview";
import { formatBytes, formatDate, copyToClipboard } from "@/utils/format";

const { Title } = Typography;
const TABLE_MD: Breakpoint[] = ["md"];
const TABLE_LG: Breakpoint[] = ["lg"];
const TABLE_XL: Breakpoint[] = ["xl"];

export default function AdminFiles() {
  const [files, setFiles] = useState<FileItem[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [banTarget, setBanTarget] = useState<FileItem | null>(null);
  const [banReason, setBanReason] = useState("");
  const [actionLoading, setActionLoading] = useState(false);
  const [previewFile, setPreviewFile] = useState<FileItem | null>(null);
  const [keyword, setKeyword] = useState("");
  const pageSize = 50;
  const { message } = App.useApp();
  const screens = Grid.useBreakpoint();
  const isMobile = !screens.md;

  const fetchFiles = useCallback(async (p: number, nextKeyword = keyword) => {
    setLoading(true);
    try {
      const data = await listAllFiles((p - 1) * pageSize, pageSize, nextKeyword);
      setFiles(data.files);
      setTotal(data.total);
      setPage(p);
    } finally {
      setLoading(false);
    }
  }, [keyword]);

  useEffect(() => {
    void fetchFiles(1);
  }, [fetchFiles]);

  const handleCopyLink = async (file: FileItem) => {
    try {
      const { url } = await getFileURL(file.id);
      await copyToClipboard(url);
      message.success("直链已复制");
    } catch {
      message.error("获取链接失败");
    }
  };

  const handleDownload = async (file: FileItem) => {
    try {
      const { url } = await getFileURL(file.id);
      window.open(url, "_blank", "noopener,noreferrer");
    } catch {
      message.error("获取下载链接失败");
    }
  };

  const handleDelete = async (id: string) => {
    await adminDeleteFile(id);
    message.success("已删除");
    fetchFiles(page);
  };

  const handleBan = async () => {
    if (!banTarget) return;
    const reason = banReason.trim();
    if (!reason) {
      message.warning("请输入封禁理由");
      return;
    }
    setActionLoading(true);
    try {
      await adminBanFile(banTarget.id, reason);
      message.success("文件已封禁");
      setBanTarget(null);
      setBanReason("");
      fetchFiles(page);
    } catch (error: unknown) {
      const maybeMessage = (
        error as { response?: { data?: { error?: string } } }
      )?.response?.data?.error;
      message.error(maybeMessage || "封禁失败");
    } finally {
      setActionLoading(false);
    }
  };

  const handleUnban = async (record: FileItem) => {
    setActionLoading(true);
    try {
      await adminUnbanFile(record.id);
      message.success("已解除封禁");
      fetchFiles(page);
    } catch (error: unknown) {
      const maybeMessage = (
        error as { response?: { data?: { error?: string } } }
      )?.response?.data?.error;
      message.error(maybeMessage || "解除失败");
    } finally {
      setActionLoading(false);
    }
  };

  const columns = [
    {
      title: "文件名",
      dataIndex: "name",
      key: "name",
      ellipsis: true,
      render: (_: string, record: FileItem) => (
        <div style={{ minWidth: 0 }}>
          <div
            style={{
              overflow: "hidden",
              textOverflow: "ellipsis",
              whiteSpace: "nowrap",
              fontWeight: 600,
            }}
            title={record.name}
          >
            {record.name}
          </div>
          <Typography.Text type="secondary" style={{ fontSize: 12 }}>
            用户: {record.username || record.user_id || "-"}
          </Typography.Text>
        </div>
      ),
    },
    {
      title: "用户",
      dataIndex: "username",
      key: "username",
      width: 140,
      responsive: TABLE_MD,
      ellipsis: true,
      render: (_: string | undefined, record: FileItem) =>
        record.username || record.user_id || "-",
    },
    {
      title: "类型",
      dataIndex: "is_directory",
      key: "type",
      width: 80,
      responsive: TABLE_MD,
      render: (isDir: boolean) => (isDir ? "文件夹" : "文件"),
    },
    {
      title: "状态",
      dataIndex: "moderation_status",
      key: "moderation_status",
      width: 100,
      responsive: TABLE_MD,
      render: (status: FileItem["moderation_status"], record: FileItem) => {
        if (record.is_directory) return "-";
        return status === "banned" ? (
          <Tag color="red">已封禁</Tag>
        ) : (
          <Tag color="green">正常</Tag>
        );
      },
    },
    {
      title: "大小",
      dataIndex: "size",
      key: "size",
      width: 100,
      responsive: TABLE_LG,
      render: (size: number, record: FileItem) =>
        record.is_directory ? "-" : formatBytes(size),
    },
    {
      title: "封禁原因",
      dataIndex: "moderation_reason",
      key: "moderation_reason",
      ellipsis: true,
      responsive: TABLE_XL,
      render: (reason?: string) =>
        reason ? <Tooltip title={reason}>{reason}</Tooltip> : "-",
    },
    {
      title: "创建时间",
      dataIndex: "created_at",
      key: "created_at",
      width: 160,
      responsive: TABLE_XL,
      render: (date: string) => formatDate(date),
    },
    {
      title: "操作",
      key: "actions",
      width: isMobile ? 188 : 280,
      render: (_: unknown, record: FileItem) => (
        <Space size={4}>
          {!record.is_directory && (
            <>
              <Button
                type="text"
                size="small"
                icon={<EyeOutlined />}
                onClick={() => setPreviewFile(record)}
              >
                {!isMobile && "预览"}
              </Button>
              <Button
                type="text"
                size="small"
                icon={<LinkOutlined />}
                onClick={() => handleCopyLink(record)}
              >
                {!isMobile && "直链"}
              </Button>
              <Button
                type="text"
                size="small"
                icon={<DownloadOutlined />}
                onClick={() => handleDownload(record)}
              >
                {!isMobile && "下载"}
              </Button>
            </>
          )}
          {!record.is_directory &&
            (record.moderation_status === "banned" ? (
              <Button
                type="text"
                size="small"
                icon={<CheckCircleOutlined />}
                onClick={() => handleUnban(record)}
                loading={actionLoading}
              >
                {!isMobile && "解封"}
              </Button>
            ) : (
              <Button
                type="text"
                size="small"
                danger
                icon={<StopOutlined />}
                onClick={() => setBanTarget(record)}
              >
                {!isMobile && "封禁"}
              </Button>
            ))}
          <Popconfirm
            title="永久删除此文件？"
            onConfirm={() => handleDelete(record.id)}
            okText="删除"
            cancelText="取消"
          >
            <Button type="text" size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className="fade-in">
      <div
        className="page-header-bar"
      >
        <Title level={4} style={{ margin: 0 }}>
          全局文件管理
        </Title>
        <div className="page-header-actions">
          <Input
            value={keyword}
            onChange={(e) => setKeyword(e.target.value)}
            onPressEnter={() => void fetchFiles(1, keyword)}
            placeholder="搜索文件名 / 扩展名 / 用户名"
            allowClear
            style={{ width: isMobile ? "100%" : 260 }}
          />
          <Button onClick={() => void fetchFiles(1, keyword)}>
            查询
          </Button>
          <Button icon={<ReloadOutlined />} onClick={() => fetchFiles(page)}>
            刷新
          </Button>
        </div>
      </div>
      <div className="page-card">
        <Table
          dataSource={files}
          columns={columns}
          rowKey="id"
          loading={loading}
          pagination={false}
          size={isMobile ? "small" : "middle"}
          scroll={isMobile ? { x: 900 } : { x: 1200 }}
          locale={{ emptyText: <Empty description="暂无文件" /> }}
        />
      </div>
      {total > pageSize && (
        <div style={{ textAlign: "center", marginTop: 16 }}>
          <Pagination
            current={page}
            total={total}
            pageSize={pageSize}
            onChange={(nextPage) => void fetchFiles(nextPage)}
            showTotal={(t) => `共 ${t} 个文件`}
          />
        </div>
      )}

      <FilePreview file={previewFile} onClose={() => setPreviewFile(null)} />

      <Modal
        title={banTarget ? `封禁文件：${banTarget.name}` : "封禁文件"}
        open={!!banTarget}
        onOk={handleBan}
        onCancel={() => {
          setBanTarget(null);
          setBanReason("");
        }}
        okText="确认封禁"
        cancelText="取消"
        confirmLoading={actionLoading}
        okButtonProps={{ danger: true }}
        destroyOnClose
      >
        <Input.TextArea
          value={banReason}
          onChange={(e) => setBanReason(e.target.value)}
          placeholder="请输入封禁原因（将展示给用户）"
          rows={4}
          maxLength={1000}
          showCount
        />
      </Modal>
    </div>
  );
}
