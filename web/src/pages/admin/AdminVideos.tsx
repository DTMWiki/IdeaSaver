import { useEffect, useState } from "react";
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
  Alert,
} from "antd";
import {
  DeleteOutlined,
  EyeOutlined,
  CheckCircleOutlined,
  StopOutlined,
} from "@ant-design/icons";
import type { Video } from "@/types";
import {
  listAllVideos,
  adminDeleteVideo,
  adminGetVideoPlayInfo,
  adminSetVideoStatus,
} from "@/api/admin";
import { formatBytes, formatDate } from "@/utils/format";
import VideoThumbnail from "@/components/VideoThumbnail";

const { Title, Text } = Typography;

export default function AdminVideos() {
  const [videos, setVideos] = useState<Video[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [actionLoadingId, setActionLoadingId] = useState<string | null>(null);
  const [previewVideo, setPreviewVideo] = useState<Video | null>(null);
  const [previewState, setPreviewState] = useState<{
    loading: boolean;
    ready: boolean;
    playURL: string;
    message: string;
  }>({
    loading: false,
    ready: false,
    playURL: "",
    message: "",
  });
  const pageSize = 50;
  const { message } = App.useApp();

  const fetchVideos = async (p: number) => {
    setLoading(true);
    try {
      const data = await listAllVideos((p - 1) * pageSize, pageSize);
      setVideos(data.videos);
      setTotal(data.total);
      setPage(p);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchVideos(1);
  }, []);

  const handleDelete = async (id: string) => {
    await adminDeleteVideo(id);
    message.success("已删除");
    fetchVideos(page);
  };

  const handlePreview = async (video: Video) => {
    setPreviewVideo(video);
    setPreviewState({
      loading: true,
      ready: false,
      playURL: "",
      message: "",
    });

    try {
      const info = await adminGetVideoPlayInfo(video.id);
      setPreviewState({
        loading: false,
        ready: info.ready,
        playURL: info.play_url || "",
        message: info.message || "",
      });
    } catch (error: unknown) {
      const maybeMessage = (
        error as { response?: { data?: { error?: string } } }
      )?.response?.data?.error;
      setPreviewState({
        loading: false,
        ready: false,
        playURL: "",
        message: maybeMessage || "加载视频信息失败",
      });
    }
  };

  const handleSetStatus = async (video: Video, status: number) => {
    setActionLoadingId(video.id);
    try {
      await adminSetVideoStatus(video.id, status);
      message.success(status === 1 ? "已启用" : "已禁用");
      fetchVideos(page);
    } catch (error: unknown) {
      const maybeMessage = (
        error as { response?: { data?: { error?: string } } }
      )?.response?.data?.error;
      message.error(maybeMessage || "更新状态失败");
    } finally {
      setActionLoadingId(null);
    }
  };

  const columns = [
    {
      title: "视频",
      dataIndex: "title",
      key: "title",
      render: (_: string, record: Video) => (
        <div
          style={{
            display: "flex",
            alignItems: "center",
            gap: 12,
            minWidth: 0,
          }}
        >
          <VideoThumbnail
            src={record.thumbnail_small_url || record.thumbnail_url}
            alt={record.title}
            width={96}
            height={54}
            borderRadius={10}
            iconSize={20}
          />
          <div style={{ minWidth: 0 }}>
            <div
              style={{
                overflow: "hidden",
                textOverflow: "ellipsis",
                whiteSpace: "nowrap",
                fontWeight: 600,
              }}
              title={record.title}
            >
              {record.title}
            </div>
            <Text type="secondary" style={{ fontSize: 12 }}>
              VCode: {record.vcode || "-"}
            </Text>
          </div>
        </div>
      ),
    },
    {
      title: "用户",
      dataIndex: "user_id",
      key: "user_id",
      width: 120,
      ellipsis: true,
      render: (id: string) => id.slice(0, 8) + "...",
    },
    {
      title: "状态",
      key: "status",
      width: 150,
      render: (_: unknown, record: Video) => (
        <Space size={6} wrap>
          {record.status === 1 ? (
            <Tag color="green">启用</Tag>
          ) : (
            <Tag color="red">禁用</Tag>
          )}
          {renderTranscodeTag(record.transcode_status)}
        </Space>
      ),
    },
    {
      title: "大小",
      dataIndex: "size",
      key: "size",
      width: 100,
      render: (size: number) => formatBytes(size),
    },
    {
      title: "创建时间",
      dataIndex: "created_at",
      key: "created_at",
      width: 160,
      render: (date: string) => formatDate(date),
    },
    {
      title: "操作",
      key: "actions",
      width: 260,
      render: (_: unknown, record: Video) => (
        <Space size={4} wrap>
          <Button
            type="text"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => handlePreview(record)}
          >
            查看
          </Button>
          {record.status === 1 ? (
            <Button
              type="text"
              size="small"
              danger
              icon={<StopOutlined />}
              loading={actionLoadingId === record.id}
              onClick={() => handleSetStatus(record, 0)}
            >
              禁用
            </Button>
          ) : (
            <Button
              type="text"
              size="small"
              icon={<CheckCircleOutlined />}
              loading={actionLoadingId === record.id}
              onClick={() => handleSetStatus(record, 1)}
            >
              启用
            </Button>
          )}
          <Popconfirm
            title="删除此视频？"
            onConfirm={() => handleDelete(record.id)}
            okText="删除"
            cancelText="取消"
          >
            <Button type="text" size="small" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className="fade-in">
      <Title level={4} style={{ marginBottom: 16 }}>
        全局视频管理
      </Title>
      <div
        style={{
          background: "var(--color-bg-container)",
          borderRadius: "var(--border-radius)",
          border: "1px solid var(--color-border-secondary)",
        }}
      >
        <Table
          dataSource={videos}
          columns={columns}
          rowKey="id"
          loading={loading}
          pagination={false}
          locale={{ emptyText: <Empty description="暂无视频" /> }}
        />
      </div>
      {total > pageSize && (
        <div style={{ textAlign: "center", marginTop: 16 }}>
          <Pagination
            current={page}
            total={total}
            pageSize={pageSize}
            onChange={fetchVideos}
            showTotal={(t) => `共 ${t} 个视频`}
          />
        </div>
      )}

      <Modal
        title={previewVideo ? `查看视频：${previewVideo.title}` : "查看视频"}
        open={!!previewVideo}
        onCancel={() => {
          setPreviewVideo(null);
          setPreviewState({
            loading: false,
            ready: false,
            playURL: "",
            message: "",
          });
        }}
        footer={null}
        width={860}
        destroyOnClose
      >
        {previewState.loading ? (
          <Alert type="info" showIcon message="正在加载视频信息..." />
        ) : previewState.ready && previewState.playURL ? (
          <video
            src={previewState.playURL}
            controls
            autoPlay
            style={{
              width: "100%",
              maxHeight: 520,
              borderRadius: 12,
              background: "#000",
            }}
          />
        ) : (
          <Alert
            type="warning"
            showIcon
            message={previewState.message || "视频暂不可播放"}
          />
        )}
      </Modal>
    </div>
  );
}

function renderTranscodeTag(status: Video["transcode_status"]) {
  switch (status) {
    case "ready":
      return <Tag color="green">可播放</Tag>;
    case "failed":
      return <Tag color="red">转码失败</Tag>;
    case "blocked":
      return <Tag color="volcano">已屏蔽</Tag>;
    case "processing":
      return <Tag color="blue">转码中</Tag>;
    default:
      return <Tag>排队中</Tag>;
  }
}
