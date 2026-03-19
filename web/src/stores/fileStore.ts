import { create } from "zustand";
import type { FileItem, BreadcrumbItem } from "@/types";
import * as filesApi from "@/api/files";

type ViewMode = "table" | "grid";

interface FileState {
  files: FileItem[];
  loading: boolean;
  currentParentId: string | null;
  breadcrumbs: BreadcrumbItem[];
  selectedIds: Set<string>;
  viewMode: ViewMode;
  clipboardIds: string[];
  clipboardAction: "copy" | "cut" | null;

  fetchFiles: (parentId?: string | null) => Promise<void>;
  navigateTo: (parentId: string | null, name?: string) => void;
  navigateToBreadcrumb: (index: number) => void;
  setViewMode: (mode: ViewMode) => void;
  toggleSelect: (id: string) => void;
  setSelected: (ids: string[]) => void;
  selectAll: () => void;
  clearSelection: () => void;
  setClipboard: (ids: string[], action: "copy" | "cut") => void;
  clearClipboard: () => void;
  createDirectory: (name: string) => Promise<void>;
  renameFile: (id: string, name: string) => Promise<void>;
  deleteFiles: (ids: string[]) => Promise<void>;
  deleteSelected: () => Promise<void>;
  pasteFiles: () => Promise<void>;
  refresh: () => Promise<void>;
}

export const useFileStore = create<FileState>((set, get) => ({
  files: [],
  loading: false,
  currentParentId: null,
  breadcrumbs: [{ id: null, name: "全部文件" }],
  selectedIds: new Set(),
  viewMode: (localStorage.getItem("viewMode") as ViewMode) || "table",
  clipboardIds: [],
  clipboardAction: null,

  fetchFiles: async (parentId?: string | null) => {
    const pid = parentId !== undefined ? parentId : get().currentParentId;
    set({ loading: true });
    try {
      const files = await filesApi.listFiles(pid);
      // Directories first, then by name
      files.sort((a, b) => {
        if (a.is_directory !== b.is_directory) return a.is_directory ? -1 : 1;
        return a.name.localeCompare(b.name);
      });
      set({ files, loading: false, currentParentId: pid });
    } catch {
      set({ loading: false });
    }
  },

  navigateTo: (parentId: string | null, name?: string) => {
    const { breadcrumbs } = get();
    if (parentId === null) {
      set({ breadcrumbs: [{ id: null, name: "全部文件" }] });
    } else {
      set({
        breadcrumbs: [...breadcrumbs, { id: parentId, name: name || "文件夹" }],
      });
    }
    set({ selectedIds: new Set(), currentParentId: parentId });
    get().fetchFiles(parentId);
  },

  navigateToBreadcrumb: (index: number) => {
    const { breadcrumbs } = get();
    const newBreadcrumbs = breadcrumbs.slice(0, index + 1);
    const targetId = newBreadcrumbs[newBreadcrumbs.length - 1].id;
    set({
      breadcrumbs: newBreadcrumbs,
      selectedIds: new Set(),
      currentParentId: targetId,
    });
    get().fetchFiles(targetId);
  },

  setViewMode: (mode: ViewMode) => {
    localStorage.setItem("viewMode", mode);
    set({ viewMode: mode });
  },

  toggleSelect: (id: string) => {
    const selectedIds = new Set(get().selectedIds);
    if (selectedIds.has(id)) {
      selectedIds.delete(id);
    } else {
      selectedIds.add(id);
    }
    set({ selectedIds });
  },

  setSelected: (ids: string[]) => {
    set({ selectedIds: new Set(ids) });
  },

  selectAll: () => {
    set({ selectedIds: new Set(get().files.map((f) => f.id)) });
  },

  clearSelection: () => {
    set({ selectedIds: new Set() });
  },

  setClipboard: (ids: string[], action: "copy" | "cut") => {
    set({ clipboardIds: ids, clipboardAction: action });
  },

  clearClipboard: () => {
    set({ clipboardIds: [], clipboardAction: null });
  },

  createDirectory: async (name: string) => {
    await filesApi.mkdir(name, get().currentParentId);
    await get().fetchFiles();
  },

  renameFile: async (id: string, name: string) => {
    await filesApi.renameFile(id, name);
    await get().fetchFiles();
  },

  deleteFiles: async (ids: string[]) => {
    if (ids.length === 0) return;
    if (ids.length === 1) {
      await filesApi.softDelete(ids[0]);
    } else {
      await filesApi.batchDelete(ids);
    }
    set({ selectedIds: new Set() });
    await get().fetchFiles();
  },

  deleteSelected: async () => {
    const ids = Array.from(get().selectedIds);
    await get().deleteFiles(ids);
  },

  pasteFiles: async () => {
    const { clipboardIds, clipboardAction, currentParentId } = get();
    if (!clipboardAction || clipboardIds.length === 0) return;

    for (const id of clipboardIds) {
      if (clipboardAction === "copy") {
        await filesApi.copyFile(id, currentParentId);
      } else {
        await filesApi.moveFile(id, currentParentId);
      }
    }

    set({ clipboardIds: [], clipboardAction: null });
    await get().fetchFiles();
  },

  refresh: async () => {
    await get().fetchFiles();
  },
}));
