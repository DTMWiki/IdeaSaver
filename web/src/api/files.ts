import client from "./client";
import type { FileItem } from "@/types";

export async function listFiles(parentId?: string | null): Promise<FileItem[]> {
  const params: Record<string, string> = {};
  if (parentId) params.parent_id = parentId;
  const { data } = await client.get("/files/list", { params });
  return data.files || [];
}

export async function mkdir(
  name: string,
  parentId?: string | null,
): Promise<FileItem> {
  const { data } = await client.post("/files/mkdir", {
    name,
    parent_id: parentId || undefined,
  });
  return data.directory;
}

export async function renameFile(id: string, name: string): Promise<void> {
  await client.put(`/files/${id}/rename`, { name });
}

export async function moveFile(
  id: string,
  parentId: string | null,
): Promise<void> {
  await client.put(`/files/${id}/move`, { parent_id: parentId });
}

export async function copyFile(
  fileId: string,
  parentId?: string | null,
): Promise<FileItem> {
  const { data } = await client.post("/files/copy", {
    file_id: fileId,
    parent_id: parentId || undefined,
  });
  return data.file;
}

export async function softDelete(id: string): Promise<void> {
  await client.delete(`/files/${id}`);
}

export async function batchDelete(ids: string[]): Promise<void> {
  await client.delete("/files/batch", { data: { ids } });
}

export async function restoreFile(id: string): Promise<void> {
  await client.post(`/files/${id}/restore`);
}

export async function permanentDelete(id: string): Promise<void> {
  await client.delete(`/files/${id}/permanent`);
}

export async function listTrash(): Promise<FileItem[]> {
  const { data } = await client.get("/files/trash");
  return data.files || [];
}

export async function getFileURL(
  id: string,
): Promise<{ url: string; markdown: string }> {
  const { data } = await client.get(`/files/${id}/url`);
  return data;
}

export async function createShare(
  id: string,
  password?: string,
  expiresIn?: number,
): Promise<import("@/types").Share> {
  const { data } = await client.post(`/files/${id}/share`, {
    password: password || undefined,
    expires_in: expiresIn || undefined,
  });
  return data.share;
}

export async function submitFileAppeal(
  id: string,
  reason: string,
): Promise<import("@/types").FileAppeal> {
  const { data } = await client.post(`/files/${id}/appeal`, { reason });
  return data.appeal;
}
