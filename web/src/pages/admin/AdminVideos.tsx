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
  Grid,
  Input,
  Switch,
} from "antd";
import type { Breakpoint } from "antd";
import {
  DeleteOutlined,
  EyeOutlined,
  CheckCircleOutlined,
  StopOutlined,
  ShareAltOutlined,
  CopyOutlined,
} from "@ant-design/icons";
import type { Video } from "@/types";
import {
  listAllVideos,
  adminDeleteVideo,
  adminGetVideoPlayInfo,
  adminSetVideoStatus,
} from "@/api/admin";
import { formatBytes, formatDate, copyToClipboard } from "@/utils/format";
import VideoThumbnail from "@/components/VideoThumbnail";

const { Title, Text } = Typography;
const TABLE_MD: Breakpoint[] = ["md"];
const TABLE_LG: Breakpoint[] = ["lg"];
const TABLE_XL: Breakpoint[] = ["xl"];

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
  const screens = Grid.useBreakpoint();
  const isMobile = !screens.md;
  const [shareVideo, setShareVideo] = useState<Video | null>(null);
  const [shareLoading, setShareLoading] = useState(false);
  const [shareMessage, setShareMessage] = useState("");
  const [shareContext, setShareContext] = useState<{ vcode: string; userId: string } | null>(null);
  const [shareAutoPlay, setShareAutoPlay] = useState(false);
  const [shareWidth, setShareWidth] = useState("");
  const [shareHeight, setShareHeight] = useState("");

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

  const handleShare = async (video: Video) => {
    setShareVideo(video);
    setShareLoading(true);
    setShareMessage("");
    setShareContext(null);
    setShareAutoPlay(false);
    setShareWidth("");
    setShareHeight("");

    try {
      const info = await adminGetVideoPlayInfo(video.id);
      if (!info.ready) {
        setShareMessage(info.message || "视频仍在转码中，完成后即可生成嵌入代码。");
        return;
      }

      const vcode = (info.vcode || video.vcode || "").trim();
      const playerUserId = firstNonEmptyString(
        info.player_user_id,
        video.player_user_id,
        playerUserIDFromPlayURL(info.play_url),
        playerUserIDFromPlayURL(video.play_url),
      );
      if (!vcode || !playerUserId) {
        setShareMessage("当前视频缺少 VCode 或多吉云用户 ID，暂时无法生成分享链接。");
        return;
      }

      setShareVideo({ ...video, play_count: info.play_count ?? video.play_count });
      setShareContext({ vcode, userId: playerUserId });
    } catch (error: unknown) {
      const maybeMessage = (
        error as { response?: { data?: { error?: string } } }
      )?.response?.data?.error;
      setShareMessage(maybeMessage || "生成分享链接失败");
    } finally {
      setShareLoading(false);
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
            <br />
            <Text type="secondary" style={{ fontSize: 12 }}>
              播放次数: {record.play_count ?? 0}
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
      responsive: TABLE_LG,
      ellipsis: true,
      render: (id: string) => id.slice(0, 8) + "...",
    },
    {
      title: "状态",
      key: "status",
      width: 150,
      responsive: TABLE_MD,
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
      responsive: TABLE_LG,
      render: (size: number) => formatBytes(size),
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
      width: isMobile ? 180 : 320,
      render: (_: unknown, record: Video) => (
        <Space size={4} wrap>
          <Button
            type="text"
            size="small"
            icon={<ShareAltOutlined />}
            onClick={() => handleShare(record)}
          >
            {!isMobile && "分享"}
          </Button>
          <Button
            type="text"
            size="small"
            icon={<EyeOutlined />}
            onClick={() => handlePreview(record)}
          >
            {!isMobile && "查看"}
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
              {!isMobile && "禁用"}
            </Button>
          ) : (
            <Button
              type="text"
              size="small"
              icon={<CheckCircleOutlined />}
              loading={actionLoadingId === record.id}
              onClick={() => handleSetStatus(record, 1)}
            >
              {!isMobile && "启用"}
            </Button>
          )}
          <Popconfirm
            title="删除此视频？"
            onConfirm={() => handleDelete(record.id)}
            okText="删除"
            cancelText="取消"
          >
            <Button type="text" size="small" danger icon={<DeleteOutlined />}>
              {!isMobile && "删除"}
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
      <div className="page-card">
        <Table
          dataSource={videos}
          columns={columns}
          rowKey="id"
          loading={loading}
          pagination={false}
          size={isMobile ? "small" : "middle"}
          scroll={isMobile ? { x: 880 } : { x: 1120 }}
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
        width={isMobile ? "calc(100vw - 24px)" : 860}
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

      <Modal
        title={shareVideo ? `分享视频：${shareVideo.title}` : "分享视频"}
        open={!!shareVideo}
        onCancel={() => {
          setShareVideo(null);
          setShareLoading(false);
          setShareMessage("");
          setShareContext(null);
          setShareAutoPlay(false);
          setShareWidth("");
          setShareHeight("");
        }}
        footer={null}
        width={isMobile ? "calc(100vw - 24px)" : 680}
        destroyOnClose
      >
        <Space direction="vertical" size={12} style={{ width: "100%" }}>
          <Alert
            type={shareContext ? "success" : "info"}
            showIcon
            message="使用嵌入代码接入多吉云播放器"
            description="已开启防盗链时，不建议继续暴露直达地址。这里默认只提供 iframe 嵌入代码。"
          />
          <div>
            <Text strong>使用方式（Markdown / HTML 嵌入）</Text>
            <ol style={{ margin: "8px 0 0", paddingInlineStart: 18, color: "rgba(71,85,105,0.9)" }}>
              <li>复制下方 iframe 代码，粘贴到支持 HTML 的页面中。</li>
              <li>若系统支持 Markdown 中嵌入 HTML，可直接使用这段代码。</li>
              <li>若视频仍在转码中，请等待多吉云回调完成后再复制。</li>
            </ol>
          </div>
          {shareLoading ? (
            <Alert type="info" showIcon message="正在生成嵌入代码..." />
          ) : shareContext ? (
            <Space direction="vertical" size={8} style={{ width: "100%" }}>
              <Text type="secondary">播放次数：{shareVideo?.play_count ?? 0}</Text>
              <Space wrap size={12} style={{ width: "100%" }}>
                <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
                  <Text type="secondary">自动播放</Text>
                  <Switch checked={shareAutoPlay} onChange={setShareAutoPlay} />
                </div>
                <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
                  <Text type="secondary">宽度</Text>
                  <Input
                    value={shareWidth}
                    onChange={(e) => setShareWidth(e.target.value.replace(/[^\d]/g, ""))}
                    placeholder="600"
                    style={{ width: isMobile ? "100%" : 120 }}
                  />
                </div>
                <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
                  <Text type="secondary">高度</Text>
                  <Input
                    value={shareHeight}
                    onChange={(e) => setShareHeight(e.target.value.replace(/[^\d]/g, ""))}
                    placeholder="400"
                    style={{ width: isMobile ? "100%" : 120 }}
                  />
                </div>
              </Space>
              <div style={{ position: "relative", borderRadius: 14, overflow: "hidden", background: "#0f172a" }}>
                <Button
                  type="primary"
                  icon={<CopyOutlined />}
                  size="small"
                  style={{ position: "absolute", top: 12, right: 12, zIndex: 1 }}
                  onClick={async () => {
                    await copyToClipboard(buildDogeIframeCode({
                      vcode: shareContext.vcode,
                      userId: shareContext.userId,
                      autoPlay: shareAutoPlay,
                      width: normalizeDimension(shareWidth, "600"),
                      height: normalizeDimension(shareHeight, "400"),
                    }));
                    message.success("嵌入代码已复制");
                  }}
                >
                  复制
                </Button>
                <pre style={{ margin: 0, padding: "52px 16px 16px", color: "#e2e8f0", whiteSpace: "pre-wrap", wordBreak: "break-word", fontSize: 13, lineHeight: 1.6 }}>
                  <code>
                    {buildDogeIframeCode({
                      vcode: shareContext.vcode,
                      userId: shareContext.userId,
                      autoPlay: shareAutoPlay,
                      width: normalizeDimension(shareWidth, "600"),
                      height: normalizeDimension(shareHeight, "400"),
                    })}
                  </code>
                </pre>
              </div>
            </Space>
          ) : (
            <Alert type="warning" showIcon message={shareMessage || "暂时无法生成分享链接"} />
          )}
        </Space>
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

function firstNonEmptyString(...values: Array<string | undefined>) {
  for (const value of values) {
    const next = value?.trim();
    if (next) {
      return next;
    }
  }
  return "";
}

function playerUserIDFromPlayURL(raw?: string) {
  const value = raw?.trim();
  if (!value) return "";
  try {
    const u = new URL(value);
    return firstNonEmptyString(
      u.searchParams.get("userId") ?? "",
      u.searchParams.get("userid") ?? "",
      u.searchParams.get("uid") ?? "",
    );
  } catch {
    return "";
  }
}

function buildDogeIframeCode(input: { vcode: string; userId: string; autoPlay: boolean; width: string; height: string }) {
  const params = new URLSearchParams({
    vcode: input.vcode.trim(),
    userId: input.userId.trim(),
    inFrame: "true",
  });
  if (input.autoPlay) {
    params.set("autoPlay", "true");
  }
  const src = `https://player.dogecloud.com/web/player.html?${params.toString()}`;
  return `<iframe id="dogePlayerFrame" src="${src}" allowfullscreen="true" msallowfullscreen="true" webkitallowfullscreen="true" mozallowfullscreen="true" oallowfullscreen="true" allowtransparency="true" scrolling="no" width="${input.width}" height="${input.height}" frameborder="0" allow="accelerometer; autoplay; encrypted-media; gyroscope; picture-in-picture; fullscreen" referrerPolicy="unsafe-url"></iframe>`;
}

function normalizeDimension(value: string, fallback: string) {
  const next = value.trim();
  return next || fallback;
}
