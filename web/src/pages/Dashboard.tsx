import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Button, Space, Breadcrumb, Segmented, App } from "antd";
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
import DashboardDialogs from "@/components/DashboardDialogs";
import { useDashboardActions } from "@/hooks/useDashboardActions";
import type { FileItem } from "@/types";
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
    navigateToBreadcrumb,
    setViewMode,
    createDirectory,
    selectAll,
    clearSelection,
    setSelected,
    pasteFiles,
  } = useFileStore();
  const { addFiles } = useUploadStore();
  const { message } = App.useApp();
  const actions = useDashboardActions();

  const fileInputRef = useRef<HTMLInputElement>(null);
  const [mkdirVisible, setMkdirVisible] = useState(false);
  const [mkdirName, setMkdirName] = useState("");
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
    } catch {
      message.error("刷新失败");
    }
  }, [message, refresh]);

  const handleContextMenu = (e: React.MouseEvent, file: FileItem | null) => {
    e.preventDefault();
    if (file && !selectedIds.has(file.id)) {
      setSelected([file.id]);
    }
    setContextMenu({ x: e.clientX, y: e.clientY, file });
  };

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
          onPreview={actions.setPreviewFile}
          actionItems={actions.fileActionItems}
        />
      ) : (
        <FileGrid
          files={files}
          loading={loading}
          selectedIds={selectedIds}
          onContextMenu={handleContextMenu}
          onPreview={actions.setPreviewFile}
          actionItems={actions.fileActionItems}
        />
      )}

      <input
        ref={fileInputRef}
        type="file"
        multiple
        style={{ display: "none" }}
        onChange={handleFileSelect}
      />

      {contextMenu && (
        <FileContextMenu
          x={contextMenu.x}
          y={contextMenu.y}
          file={contextMenu.file}
          fileItems={
            contextMenu.file ? actions.fileActionItems(contextMenu.file) : []
          }
          backgroundItems={backgroundItems}
          onClose={() => setContextMenu(null)}
        />
      )}

      <DashboardDialogs
        mkdirVisible={mkdirVisible}
        mkdirName={mkdirName}
        onMkdirNameChange={setMkdirName}
        onMkdirOk={() => {
          void handleMkdir();
        }}
        onMkdirCancel={() => {
          setMkdirVisible(false);
          setMkdirName("");
        }}
        renameTarget={actions.renameTarget}
        renameName={actions.renameName}
        renameSubmitting={actions.renameSubmitting}
        onRenameNameChange={actions.setRenameName}
        onRenameOk={() => {
          void actions.handleConfirmRename();
        }}
        onRenameCancel={() => {
          actions.setRenameTarget(null);
          actions.setRenameName("");
        }}
        previewFile={actions.previewFile}
        onPreviewClose={() => actions.setPreviewFile(null)}
        shareFileId={actions.shareFileId}
        onShareClose={() => actions.setShareFileId(null)}
        appealFile={actions.appealFile}
        appealReason={actions.appealReason}
        appealSubmitting={actions.appealSubmitting}
        onAppealReasonChange={actions.setAppealReason}
        onAppealOk={() => {
          void actions.handleSubmitAppeal();
        }}
        onAppealCancel={() => {
          actions.setAppealFile(null);
          actions.setAppealReason("");
        }}
      />
    </div>
  );
}
