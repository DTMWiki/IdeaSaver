import { useEffect, useState, type ReactNode } from "react";
import {
  Timeline,
  Typography,
  Tag,
  Empty,
  Spin,
  Button,
  Space,
  Alert,
} from "antd";
import {
  UploadOutlined,
  DeleteOutlined,
  EditOutlined,
  LoginOutlined,
  FileOutlined,
  VideoCameraOutlined,
  ShareAltOutlined,
  UserOutlined,
} from "@ant-design/icons";
import type { AuditLog } from "@/types";
import { getHistory } from "@/api/auth";
import { formatRelativeTime, formatDate } from "@/utils/format";

const { Title, Text } = Typography;

const ACTION_CONFIG: Record<
  string,
  { icon: ReactNode; color: string; label: string }
> = {
  login: { icon: <LoginOutlined />, color: "#1677ff", label: "登录" },
  upload: { icon: <UploadOutlined />, color: "#52c41a", label: "上传" },
  create_directory: {
    icon: <FileOutlined />,
    color: "#13c2c2",
    label: "新建文件夹",
  },
  delete: { icon: <DeleteOutlined />, color: "#ff4d4f", label: "删除" },
  permanent_delete: {
    icon: <DeleteOutlined />,
    color: "#a8071a",
    label: "彻底删除",
  },
  rename: { icon: <EditOutlined />, color: "#faad14", label: "重命名" },
  move: { icon: <FileOutlined />, color: "#722ed1", label: "移动" },
  copy: { icon: <FileOutlined />, color: "#13c2c2", label: "复制" },
  share: { icon: <ShareAltOutlined />, color: "#eb2f96", label: "分享" },
  share_delete: {
    icon: <ShareAltOutlined />,
    color: "#cf1322",
    label: "取消分享",
  },
  restore: { icon: <FileOutlined />, color: "#52c41a", label: "恢复" },
  video_upload: {
    icon: <VideoCameraOutlined />,
    color: "#1677ff",
    label: "视频上传",
  },
  video_play_request: {
    icon: <VideoCameraOutlined />,
    color: "#13c2c2",
    label: "视频播放请求",
  },
  video_delete: {
    icon: <VideoCameraOutlined />,
    color: "#ff4d4f",
    label: "视频删除",
  },
  video_status_change: {
    icon: <VideoCameraOutlined />,
    color: "#faad14",
    label: "视频状态变更",
  },
  file_appeal_submitted: {
    icon: <FileOutlined />,
    color: "#fa8c16",
    label: "提交申诉",
  },
  file_appeal_approved: {
    icon: <FileOutlined />,
    color: "#13c2c2",
    label: "申诉通过",
  },
  file_appeal_deleted: {
    icon: <DeleteOutlined />,
    color: "#cf1322",
    label: "申诉删除",
  },
  file_banned: {
    icon: <DeleteOutlined />,
    color: "#cf1322",
    label: "文件封禁",
  },
  file_unbanned: {
    icon: <FileOutlined />,
    color: "#52c41a",
    label: "文件解封",
  },
  admin_video_deleted: {
    icon: <VideoCameraOutlined />,
    color: "#cf1322",
    label: "管理员删除视频",
  },
  admin_file_deleted: {
    icon: <FileOutlined />,
    color: "#cf1322",
    label: "管理员删除文件",
  },
};

const DEFAULT_CONFIG = {
  icon: <UserOutlined />,
  color: "#8c8c8c",
  label: "操作",
};

export default function HistoryPage() {
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [offset, setOffset] = useState(0);
  const [error, setError] = useState("");
  const limit = 30;

  const fetchLogs = async (newOffset: number) => {
    setLoading(true);
    setError("");
    try {
      const data = await getHistory(newOffset, limit);
      if (newOffset === 0) {
        setLogs(data.logs || []);
      } else {
        setLogs((prev) => [...prev, ...(data.logs || [])]);
      }
      setTotal(data.total || 0);
      setOffset(newOffset);
    } catch (err: unknown) {
      const message = (err as { response?: { data?: { error?: string } } })
        ?.response?.data?.error;
      setError(message || "加载操作历史失败");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchLogs(0);
  }, []);

  const handleLoadMore = () => {
    fetchLogs(offset + limit);
  };

  return (
    <div className="fade-in">
      <Title level={4} style={{ marginBottom: 16 }}>
        操作历史
      </Title>

      {error && (
        <Alert
          type="error"
          showIcon
          style={{ marginBottom: 16 }}
          message="操作历史加载失败"
          description={error}
          action={
            <Button size="small" onClick={() => fetchLogs(0)}>
              重试
            </Button>
          }
        />
      )}

      {logs.length === 0 && !loading ? (
        <Empty description={error ? "请稍后重试" : "暂无操作记录"} />
      ) : (
        <div
          style={{
            background: "var(--color-bg-container)",
            borderRadius: "var(--border-radius)",
            border: "1px solid var(--color-border-secondary)",
            padding: 24,
          }}
        >
          <Timeline
            items={logs.map((log) => {
              const config = ACTION_CONFIG[log.action] || DEFAULT_CONFIG;
              return {
                dot: config.icon,
                color: config.color,
                children: (
                  <div style={{ paddingBottom: 4 }}>
                    <Space size={8} wrap>
                      <Tag color={config.color}>{config.label}</Tag>
                      {log.resource && (
                        <Text type="secondary" style={{ fontSize: 13 }}>
                          {log.resource}
                        </Text>
                      )}
                    </Space>
                    {renderAuditDetails(log.details)}
                    <Text
                      type="secondary"
                      style={{ display: "block", fontSize: 12, marginTop: 2 }}
                    >
                      {formatRelativeTime(log.created_at)} ·{" "}
                      {formatDate(log.created_at)}
                    </Text>
                    {log.ip_address && (
                      <Text
                        type="secondary"
                        style={{ display: "block", fontSize: 11 }}
                      >
                        IP: {log.ip_address}
                      </Text>
                    )}
                  </div>
                ),
              };
            })}
          />

          {loading && (
            <div style={{ textAlign: "center", padding: 16 }}>
              <Spin />
            </div>
          )}

          {!loading && logs.length < total && (
            <div style={{ textAlign: "center" }}>
              <Button onClick={handleLoadMore}>加载更多</Button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

function renderAuditDetails(details: unknown) {
  const text = summarizeAuditDetails(details);
  if (!text) return null;
  return (
    <Text
      type="secondary"
      style={{ display: "block", fontSize: 12, marginTop: 2 }}
    >
      {text}
    </Text>
  );
}

function summarizeAuditDetails(details: unknown): string {
  if (details == null) return "";
  if (typeof details === "string") return details;
  if (typeof details === "number" || typeof details === "boolean")
    return String(details);
  if (Array.isArray(details)) {
    return details
      .map((item) => summarizeAuditDetails(item))
      .filter(Boolean)
      .join(" | ");
  }
  if (typeof details === "object") {
    const entries = Object.entries(details as Record<string, unknown>);
    if (entries.length === 0) return "";
    return entries
      .slice(0, 6)
      .map(
        ([key, value]) =>
          `${DETAIL_LABELS[key] || key}: ${formatDetailValue(value)}`,
      )
      .join(" | ");
  }
  return "";
}

const DETAIL_LABELS: Record<string, string> = {
  file_name: "文件",
  storage_key: "存储键",
  size: "大小",
  old_name: "原名称",
  new_name: "新名称",
  title: "标题",
  vid: "VID",
  vcode: "播放码",
  status: "状态",
  ready: "就绪",
  transcode_status: "转码状态",
  owner_id: "所属用户",
  username: "用户名",
  role: "角色",
  reason: "原因",
  comment: "备注",
  count: "数量",
  name: "名称",
  parent_id: "父目录",
  from_parent_id: "原目录",
  to_parent_id: "目标目录",
  source_name: "源文件",
  copied_name: "复制后名称",
  source_file_id: "源文件ID",
  videos: "视频列表",
};

function formatDetailValue(value: unknown): string {
  if (value == null) return "-";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean")
    return String(value);
  if (Array.isArray(value))
    return value.map((item) => formatDetailValue(item)).join(", ");
  if (typeof value === "object") return JSON.stringify(value);
  return String(value);
}
