import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Button, Space, Breadcrumb, Input, Modal, Segmented, App } from "antd";
import type { MenuProps } from "antd";
import {
  UploadOutlined,
  FolderAddOutlined,
  ReloadOutlined,
  AppstoreOutlined,
  BarsOutlined,
  SnippetsOutlined,
  SelectOutlined,
} from "@ant-design/icons";
import { useFileStore } from "@/stores/fileStore";
import { useUploadStore } from "@/stores/uploadStore";
import FileTable from "@/components/FileTable";
import FileGrid from "@/components/FileGrid";
import FileContextMenu from "@/components/FileContextMenu";
import FilePreview from "@/components/FilePreview";
import CreateShareModal from "@/components/CreateShareModal";
import { buildFileActionItems } from "@/components/fileActionItems";
import type { FileItem } from "@/types";
import { getFileURL, submitFileAppeal } from "@/api/files";
import { copyToClipboard } from "@/utils/format";
import "./Dashboard.css";

export default function Dashboard() {
  const {
    files,
    loading,
    breadcrumbs,
    selectedIds,
    viewMode,
    clipboardAction,
    fetchFiles,
    refresh,
    navigateTo,
    navigateToBreadcrumb,
    setViewMode,
    createDirectory,
    renameFile,
    deleteFiles,
    selectAll,
    clearSelection,
    setSelected,
    setClipboard,
    pasteFiles,
  } = useFileStore();
  const { addFiles } = useUploadStore();
  const { message, modal } = App.useApp();

  const fileInputRef = useRef<HTMLInputElement>(null);
  const [mkdirVisible, setMkdirVisible] = useState(false);
  const [mkdirName, setMkdirName] = useState("");
  const [previewFile, setPreviewFile] = useState<FileItem | null>(null);
  const [shareFileId, setShareFileId] = useState<string | null>(null);
  const [appealFile, setAppealFile] = useState<FileItem | null>(null);
  const [appealReason, setAppealReason] = useState("");
  const [appealSubmitting, setAppealSubmitting] = useState(false);
  const [renameTarget, setRenameTarget] = useState<FileItem | null>(null);
  const [renameName, setRenameName] = useState("");
  const [renameSubmitting, setRenameSubmitting] = useState(false);
  const [contextMenu, setContextMenu] = useState<{
    x: number;
    y: number;
    file: FileItem | null;
  } | null>(null);

  useEffect(() => {
    fetchFiles(null);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const selectedCount = selectedIds.size;

  const handleUploadClick = () => fileInputRef.current?.click();

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const fileList = e.target.files;
    if (!fileList || fileList.length === 0) return;
    const currentParentId = useFileStore.getState().currentParentId;
    addFiles(Array.from(fileList), currentParentId);
    e.target.value = "";
  };

  const handleMkdir = async () => {
    if (!mkdirName.trim()) return;
    try {
      await createDirectory(mkdirName.trim());
      message.success("文件夹已创建");
      setMkdirVisible(false);
      setMkdirName("");
    } catch {
      message.error("创建失败");
    }
  };

  const handlePaste = useCallback(async () => {
    try {
      await pasteFiles();
      message.success("粘贴完成");
    } catch {
      message.error("粘贴失败");
    }
  }, [message, pasteFiles]);

  const handleRefresh = useCallback(async () => {
    try {
      await refresh();
      message.success("文件列表已刷新");
    } catch {
      message.error("刷新失败");
    }
  }, [message, refresh]);

  const resolveTargetIDs = (file: FileItem) => {
    const currentSelection = useFileStore.getState().selectedIds;
    if (currentSelection.has(file.id) && currentSelection.size > 1) {
      return Array.from(currentSelection);
    }
    return [file.id];
  };

  const openRename = (file: FileItem) => {
    setRenameTarget(file);
    setRenameName(file.name);
  };

  const handleConfirmRename = async () => {
    if (!renameTarget || !renameName.trim()) return;
    setRenameSubmitting(true);
    try {
      await renameFile(renameTarget.id, renameName.trim());
      message.success("重命名成功");
      setRenameTarget(null);
      setRenameName("");
    } catch {
      message.error("重命名失败");
    } finally {
      setRenameSubmitting(false);
    }
  };

  const handleCopyLink = async (file: FileItem) => {
    try {
      const { url } = await getFileURL(file.id);
      await copyToClipboard(url);
      message.success("直链已复制");
    } catch {
      message.error("获取链接失败");
    }
  };

  const handleCopyMarkdown = async (file: FileItem) => {
    try {
      const { markdown, url } = await getFileURL(file.id);
      await copyToClipboard(markdown || `[${file.name}](${url})`);
      message.success("Markdown 已复制");
    } catch {
      message.error("获取 Markdown 失败");
    }
  };

  const handleDelete = (file: FileItem) => {
    const ids = resolveTargetIDs(file);
    const batch = ids.length > 1;
    modal.confirm({
      title: batch ? "确认批量删除" : "确认删除",
      content: batch
        ? `确定要将 ${ids.length} 个项目移入回收站吗？`
        : `确定要将 "${file.name}" 移入回收站吗？`,
      okText: "删除",
      okType: "danger",
      cancelText: "取消",
      onOk: async () => {
        await deleteFiles(ids);
        message.success(
          batch ? `已移入回收站 (${ids.length} 项)` : "已移入回收站",
        );
      },
    });
  };

  const handleContextMenu = (e: React.MouseEvent, file: FileItem | null) => {
    e.preventDefault();
    if (file && !selectedIds.has(file.id)) {
      setSelected([file.id]);
    }
    setContextMenu({ x: e.clientX, y: e.clientY, file });
  };

  const handleSubmitAppeal = async () => {
    if (!appealFile) return;
    const reason = appealReason.trim();
    if (!reason) {
      message.warning("请填写申诉理由");
      return;
    }

    setAppealSubmitting(true);
    try {
      await submitFileAppeal(appealFile.id, reason);
      message.success("申诉工单已提交，请等待管理员审核");
      setAppealFile(null);
      setAppealReason("");
    } catch (error: unknown) {
      const maybeMessage = (
        error as { response?: { data?: { error?: string } } }
      )?.response?.data?.error;
      message.error(maybeMessage || "提交申诉失败");
    } finally {
      setAppealSubmitting(false);
    }
  };

  const fileActionItems = (file: FileItem) =>
    buildFileActionItems(file, {
      onOpen: (target) => navigateTo(target.id, target.name),
      onPreview: (target) => setPreviewFile(target),
      onCopyLink: handleCopyLink,
      onCopyMarkdown: handleCopyMarkdown,
      onShare: (target) => setShareFileId(target.id),
      onAppeal: (target) => setAppealFile(target),
      onRename: openRename,
      onCopy: (target) => {
        const ids = resolveTargetIDs(target);
        setClipboard(ids, "copy");
        message.info(
          ids.length > 1
            ? `已复制 ${ids.length} 个项目`
            : `已复制 "${target.name}"`,
        );
      },
      onCut: (target) => {
        const ids = resolveTargetIDs(target);
        setClipboard(ids, "cut");
        message.info(
          ids.length > 1
            ? `已剪切 ${ids.length} 个项目`
            : `已剪切 "${target.name}"`,
        );
      },
      onDelete: handleDelete,
    });

  const backgroundItems = useMemo<MenuProps["items"]>(() => {
    const items: NonNullable<MenuProps["items"]> = [];

    if (clipboardAction) {
      items.push({
        key: "paste",
        icon: <SnippetsOutlined />,
        label: "粘贴",
        onClick: handlePaste,
      });
    }

    items.push({
      key: "refresh",
      icon: <ReloadOutlined />,
      label: "刷新列表",
      onClick: () => {
        void handleRefresh();
      },
    });
    items.push({
      key: "select-all",
      icon: <SelectOutlined />,
      label: "全选",
      onClick: selectAll,
    });

    return items;
  }, [clipboardAction, handlePaste, handleRefresh, selectAll]);

  return (
    <div
      className="dashboard-page fade-in"
      onContextMenu={(e) => handleContextMenu(e, null)}
    >
      <div className="dashboard-toolbar">
        <div className="dashboard-toolbar-main">
          <Breadcrumb
            items={breadcrumbs.map((b, i) => ({
              title: <a onClick={() => navigateToBreadcrumb(i)}>{b.name}</a>,
            }))}
          />
          {selectedCount > 0 && (
            <div className="selection-pill">
              已选择 {selectedCount} 项
              <button type="button" onClick={clearSelection}>
                清空
              </button>
            </div>
          )}
        </div>

        <Space size={10} wrap>
          {clipboardAction && (
            <Button icon={<SnippetsOutlined />} onClick={handlePaste}>
              粘贴
            </Button>
          )}
          <Button
            icon={<FolderAddOutlined />}
            onClick={() => setMkdirVisible(true)}
          >
            新建文件夹
          </Button>
          <Button
            type="primary"
            icon={<UploadOutlined />}
            onClick={handleUploadClick}
          >
            上传文件
          </Button>
          <Button
            icon={<ReloadOutlined />}
            onClick={() => {
              void handleRefresh();
            }}
          >
            刷新
          </Button>
          <Segmented
            size="small"
            value={viewMode}
            onChange={(v) => setViewMode(v as "table" | "grid")}
            options={[
              { value: "table", icon: <BarsOutlined /> },
              { value: "grid", icon: <AppstoreOutlined /> },
            ]}
          />
        </Space>
      </div>

      {viewMode === "table" ? (
        <FileTable
          files={files}
          loading={loading}
          selectedIds={selectedIds}
          onContextMenu={handleContextMenu}
          onPreview={setPreviewFile}
          actionItems={fileActionItems}
        />
      ) : (
        <FileGrid
          files={files}
          loading={loading}
          selectedIds={selectedIds}
          onContextMenu={handleContextMenu}
          onPreview={setPreviewFile}
          actionItems={fileActionItems}
        />
      )}

      <input
        ref={fileInputRef}
        type="file"
        multiple
        style={{ display: "none" }}
        onChange={handleFileSelect}
      />

      <Modal
        title="新建文件夹"
        open={mkdirVisible}
        onOk={handleMkdir}
        onCancel={() => {
          setMkdirVisible(false);
          setMkdirName("");
        }}
        okText="创建"
        cancelText="取消"
      >
        <Input
          placeholder="文件夹名称"
          value={mkdirName}
          onChange={(e) => setMkdirName(e.target.value)}
          onPressEnter={handleMkdir}
          autoFocus
        />
      </Modal>

      <Modal
        title={renameTarget ? `重命名：${renameTarget.name}` : "重命名"}
        open={!!renameTarget}
        onOk={handleConfirmRename}
        onCancel={() => {
          setRenameTarget(null);
          setRenameName("");
        }}
        okText="保存"
        cancelText="取消"
        confirmLoading={renameSubmitting}
        destroyOnClose
      >
        <Input
          placeholder="输入新名称"
          value={renameName}
          onChange={(e) => setRenameName(e.target.value)}
          onPressEnter={handleConfirmRename}
          autoFocus
        />
      </Modal>

      {contextMenu && (
        <FileContextMenu
          x={contextMenu.x}
          y={contextMenu.y}
          file={contextMenu.file}
          fileItems={contextMenu.file ? fileActionItems(contextMenu.file) : []}
          backgroundItems={backgroundItems}
          onClose={() => setContextMenu(null)}
        />
      )}

      <FilePreview file={previewFile} onClose={() => setPreviewFile(null)} />
      <CreateShareModal
        fileId={shareFileId}
        onClose={() => setShareFileId(null)}
      />

      <Modal
        title={appealFile ? `提交申诉：${appealFile.name}` : "提交申诉"}
        open={!!appealFile}
        onOk={handleSubmitAppeal}
        onCancel={() => {
          setAppealFile(null);
          setAppealReason("");
        }}
        okText="提交工单"
        cancelText="取消"
        confirmLoading={appealSubmitting}
        destroyOnClose
      >
        <Input.TextArea
          value={appealReason}
          onChange={(e) => setAppealReason(e.target.value)}
          placeholder="请说明你认为该文件应恢复访问的理由（例如用途、来源、已整改内容）"
          rows={5}
          maxLength={1000}
          showCount
        />
      </Modal>
    </div>
  );
}
