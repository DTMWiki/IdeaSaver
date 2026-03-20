import client from "./client";
import type { FileItem, Video, AuditLog, User, FileAppeal } from "@/types";

export interface ListLogsParams {
  action?: string;
  user?: string;
  keyword?: string;
  startAt?: string;
  endAt?: string;
  offset?: number;
  limit?: number;
}

export async function listAllFiles(
  offset = 0,
  limit = 50,
): Promise<{ files: FileItem[]; total: number }> {
  const { data } = await client.get("/admin/files", {
    params: { offset, limit },
  });
  return { files: data.files || [], total: data.total || 0 };
}

export async function adminDeleteFile(id: string): Promise<void> {
  await client.delete(`/admin/files/${id}`);
}

export async function adminBanFile(id: string, reason: string): Promise<void> {
  await client.put(`/admin/files/${id}/ban`, { reason });
}

export async function adminUnbanFile(
  id: string,
  comment?: string,
): Promise<void> {
  await client.put(`/admin/files/${id}/unban`, { comment });
}

export async function listAllVideos(
  offset = 0,
  limit = 50,
): Promise<{ videos: Video[]; total: number }> {
  const { data } = await client.get("/admin/videos", {
    params: { offset, limit },
  });
  return { videos: data.videos || [], total: data.total || 0 };
}

export async function adminDeleteVideo(id: string): Promise<void> {
  await client.delete(`/admin/videos/${id}`);
}

export async function adminGetVideoPlayInfo(id: string): Promise<{
  ready: boolean;
  play_url?: string;
  vcode?: string;
  player_user_id?: string;
  play_count: number;
  transcode_status: string;
  message?: string;
}> {
  const { data } = await client.get(`/admin/videos/${id}/play-info`);
  return data;
}

export async function adminSetVideoStatus(
  id: string,
  status: number,
): Promise<void> {
  await client.put(`/admin/videos/${id}/status`, { status });
}

export async function listLogs(
  params: ListLogsParams = {},
): Promise<{ logs: AuditLog[]; total: number }> {
  const query: Record<string, string | number> = {
    offset: params.offset ?? 0,
    limit: params.limit ?? 50,
  };
  if (params.action) query.action = params.action;
  if (params.user) query.user = params.user;
  if (params.keyword) query.keyword = params.keyword;
  if (params.startAt) query.start_at = params.startAt;
  if (params.endAt) query.end_at = params.endAt;
  const { data } = await client.get("/admin/logs", { params: query });
  return { logs: data.logs || [], total: data.total || 0 };
}

export async function listUsers(
  offset = 0,
  limit = 50,
): Promise<{ users: User[]; total: number }> {
  const { data } = await client.get("/admin/users", {
    params: { offset, limit },
  });
  return { users: data.users || [], total: data.total || 0 };
}

export async function updateUserQuota(
  id: string,
  quota: number,
): Promise<void> {
  await client.put(`/admin/users/${id}/quota`, { quota });
}

export async function cleanupTrash(): Promise<number> {
  const { data } = await client.delete("/admin/trash/cleanup");
  return data.cleaned;
}

export async function listAppeals(
  status?: string,
  offset = 0,
  limit = 50,
): Promise<{ appeals: FileAppeal[]; total: number }> {
  const params: Record<string, string | number> = { offset, limit };
  if (status) params.status = status;
  const { data } = await client.get("/admin/appeals", { params });
  return { appeals: data.appeals || [], total: data.total || 0 };
}

export async function reviewAppeal(
  id: string,
  decision: "approve" | "delete",
  comment?: string,
): Promise<void> {
  await client.put(`/admin/appeals/${id}/review`, { decision, comment });
}
