import { useCallback, useEffect, useState } from "react";
import dayjs, { type Dayjs } from "dayjs";
import {
  Table,
  Typography,
  Empty,
  Select,
  Pagination,
  Tag,
  Space,
  Input,
  DatePicker,
  Button,
} from "antd";
import type { AuditLog } from "@/types";
import { listLogs } from "@/api/admin";
import { formatDate } from "@/utils/format";

const { Title, Text } = Typography;

const ACTION_OPTIONS = [
  { label: "全部", value: "" },
  { label: "登录", value: "login" },
  { label: "上传", value: "upload" },
  { label: "新建文件夹", value: "create_directory" },
  { label: "删除", value: "delete" },
  { label: "彻底删除", value: "permanent_delete" },
  { label: "重命名", value: "rename" },
  { label: "移动", value: "move" },
  { label: "复制", value: "copy" },
  { label: "分享", value: "share" },
  { label: "取消分享", value: "share_delete" },
  { label: "恢复", value: "restore" },
  { label: "视频上传", value: "video_upload" },
  { label: "视频播放请求", value: "video_play_request" },
  { label: "视频删除", value: "video_delete" },
  { label: "视频状态变更", value: "video_status_change" },
  { label: "管理员删视频", value: "admin_video_deleted" },
  { label: "管理员删文件", value: "admin_file_deleted" },
  { label: "文件封禁", value: "file_banned" },
  { label: "文件解封", value: "file_unbanned" },
  { label: "提交申诉", value: "file_appeal_submitted" },
  { label: "申诉通过", value: "file_appeal_approved" },
  { label: "申诉删除", value: "file_appeal_deleted" },
];

const ACTION_COLORS: Record<string, string> = {
  login: "blue",
  upload: "green",
  create_directory: "cyan",
  delete: "red",
  permanent_delete: "volcano",
  rename: "orange",
  move: "purple",
  copy: "cyan",
  share: "magenta",
  share_delete: "red",
  restore: "lime",
  video_upload: "geekblue",
  video_play_request: "cyan",
  video_delete: "volcano",
  video_status_change: "gold",
  admin_video_deleted: "volcano",
  admin_file_deleted: "volcano",
  file_banned: "red",
  file_unbanned: "green",
  file_appeal_submitted: "orange",
  file_appeal_approved: "cyan",
  file_appeal_deleted: "volcano",
};

const ACTION_LABELS = Object.fromEntries(
  ACTION_OPTIONS.filter((item) => item.value).map((item) => [
    item.value,
    item.label,
  ]),
);

export default function AdminLogs() {
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [actionFilter, setActionFilter] = useState("");
  const [userFilter, setUserFilter] = useState("");
  const [keywordFilter, setKeywordFilter] = useState("");
  const [range, setRange] = useState<[Dayjs | null, Dayjs | null] | null>(null);
  const pageSize = 50;

  const fetchLogs = useCallback(
    async (
      p: number,
      overrides?: {
        action?: string;
        user?: string;
        keyword?: string;
        range?: [Dayjs | null, Dayjs | null] | null;
      },
    ) => {
      setLoading(true);
      try {
        const action = overrides?.action ?? actionFilter;
        const user = overrides?.user ?? userFilter;
        const keyword = overrides?.keyword ?? keywordFilter;
        const selectedRange = overrides?.range ?? range;
        const data = await listLogs({
          action: action || undefined,
          user: user.trim() || undefined,
          keyword: keyword.trim() || undefined,
          startAt: selectedRange?.[0]?.toISOString(),
          endAt: selectedRange?.[1]?.toISOString(),
          offset: (p - 1) * pageSize,
          limit: pageSize,
        });
        setLogs(data.logs);
        setTotal(data.total);
        setPage(p);
      } finally {
        setLoading(false);
      }
    },
    [actionFilter, keywordFilter, pageSize, range, userFilter],
  );

  useEffect(() => {
    void fetchLogs(1);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleActionChange = (action: string) => {
    setActionFilter(action);
    void fetchLogs(1, { action });
  };

  const handleSearch = () => {
    void fetchLogs(1);
  };

  const handleReset = () => {
    setActionFilter("");
    setUserFilter("");
    setKeywordFilter("");
    setRange(null);
    void fetchLogs(1, {
      action: "",
      user: "",
      keyword: "",
      range: null,
    });
  };

  const columns = [
    {
      title: "操作",
      dataIndex: "action",
      key: "action",
      width: 120,
      render: (action: string) => (
        <Tag color={ACTION_COLORS[action] || "default"}>
          {ACTION_LABELS[action] || action}
        </Tag>
      ),
    },
    {
      title: "用户",
      dataIndex: "username",
      key: "username",
      width: 180,
      ellipsis: true,
      render: (_: string | undefined, record: AuditLog) =>
        record.username || record.user_id || "-",
    },
    {
      title: "资源",
      dataIndex: "resource",
      key: "resource",
      width: 80,
      render: (resource?: string) => resource || "-",
    },
    {
      title: "详情",
      dataIndex: "details",
      key: "details",
      ellipsis: true,
      render: (details: Record<string, unknown>) => {
        const summary = summarizeAuditDetails(details);
        return summary ? (
          <Text type="secondary" style={{ fontSize: 12 }}>
            {summary}
          </Text>
        ) : (
          "-"
        );
      },
    },
    {
      title: "IP",
      dataIndex: "ip_address",
      key: "ip_address",
      width: 150,
      render: (ip: string) => ip || "-",
    },
    {
      title: "时间",
      dataIndex: "created_at",
      key: "created_at",
      width: 168,
      render: (date: string) => formatDate(date),
    },
  ];

  return (
    <div className="fade-in">
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: 16,
          flexWrap: "wrap",
          gap: 8,
        }}
      >
        <Title level={4} style={{ margin: 0 }}>
          审计日志
        </Title>
        <Space wrap>
          <Select
            value={actionFilter}
            onChange={handleActionChange}
            options={ACTION_OPTIONS}
            style={{ width: 140 }}
            size="small"
            placeholder="操作类型"
          />
          <Input
            value={userFilter}
            onChange={(e) => setUserFilter(e.target.value)}
            placeholder="用户 / 用户ID"
            allowClear
            size="small"
            style={{ width: 160 }}
            onPressEnter={handleSearch}
          />
          <Input
            value={keywordFilter}
            onChange={(e) => setKeywordFilter(e.target.value)}
            placeholder="关键词"
            allowClear
            size="small"
            style={{ width: 180 }}
            onPressEnter={handleSearch}
          />
          <DatePicker.RangePicker
            value={range}
            onChange={(values) => setRange(values)}
            allowClear
            showTime
            size="small"
          />
          <Button size="small" type="primary" onClick={handleSearch}>
            查询
          </Button>
          <Button size="small" onClick={handleReset}>
            重置
          </Button>
        </Space>
      </div>
      <div
        style={{
          background: "var(--color-bg-container)",
          borderRadius: "var(--border-radius)",
          border: "1px solid var(--color-border-secondary)",
        }}
      >
        <Table
          dataSource={logs}
          columns={columns}
          rowKey="id"
          loading={loading}
          pagination={false}
          locale={{ emptyText: <Empty description="暂无日志" /> }}
        />
      </div>
      {total > pageSize && (
        <div style={{ textAlign: "center", marginTop: 16 }}>
          <Pagination
            current={page}
            total={total}
            pageSize={pageSize}
            onChange={(p) => void fetchLogs(p)}
            showTotal={(t) => `共 ${t} 条日志`}
          />
        </div>
      )}
    </div>
  );
}

const DETAIL_LABELS: Record<string, string> = {
  file_name: "文件",
  storage_key: "存储键",
  size: "大小",
  old_name: "原名称",
  new_name: "新名称",
  title: "标题",
  vid: "VID",
  vcode: "VCode",
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

function summarizeAuditDetails(details: unknown): string {
  if (!details || typeof details !== "object" || Array.isArray(details)) {
    return "";
  }

  return Object.entries(details as Record<string, unknown>)
    .slice(0, 6)
    .map(
      ([key, value]) =>
        `${DETAIL_LABELS[key] || key}: ${formatDetailValue(value)}`,
    )
    .join(" | ");
}

function formatDetailValue(value: unknown): string {
  if (value == null) return "-";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean")
    return String(value);
  if (Array.isArray(value))
    return value.map((item) => formatDetailValue(item)).join(", ");
  if (dayjs.isDayjs(value)) return value.toISOString();
  if (typeof value === "object") return JSON.stringify(value);
  return String(value);
}
