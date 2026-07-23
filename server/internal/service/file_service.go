package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/DTMWiki/IdeaSaver/server/internal/config"
	"github.com/DTMWiki/IdeaSaver/server/internal/model"
	"github.com/DTMWiki/IdeaSaver/server/internal/repository"
	"github.com/DTMWiki/IdeaSaver/server/internal/storage"
	"github.com/google/uuid"
)

var ErrFileBanned = errors.New("file is banned")

// FileService handles file management business logic.
type FileService struct {
	cfg   *config.Config
	repos *repository.Repositories
	oss   *storage.OSSClient
}

// GetFileByID returns a file owned by the given user.
func (s *FileService) GetFileByID(ctx context.Context, id, userID uuid.UUID) (*model.File, error) {
	file, err := s.repos.Files.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if file.UserID != userID {
		return nil, ErrPermission
	}
	if file.DeletedAt != nil {
		return nil, fmt.Errorf("file deleted")
	}
	return file, nil
}

func (s *FileService) GetFileForActor(ctx context.Context, id uuid.UUID, actor *model.User) (*model.File, error) {
	file, err := s.repos.Files.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if actor == nil {
		return nil, ErrPermission
	}
	if actor.Role != "admin" && file.UserID != actor.ID {
		return nil, ErrPermission
	}
	if file.DeletedAt != nil {
		return nil, fmt.Errorf("file deleted")
	}
	return file, nil
}

func NewFileService(cfg *config.Config, repos *repository.Repositories, oss *storage.OSSClient) *FileService {
	return &FileService{cfg: cfg, repos: repos, oss: oss}
}

// ListFiles lists files in a directory.
func (s *FileService) ListFiles(ctx context.Context, userID uuid.UUID, parentID *uuid.UUID) ([]model.File, error) {
	return s.repos.Files.ListByParent(ctx, userID, parentID)
}

// CreateDirectory creates a new directory.
func (s *FileService) CreateDirectory(ctx context.Context, userID uuid.UUID, parentID *uuid.UUID, name string) (*model.File, error) {
	if err := s.assertOwnedDirectory(ctx, userID, parentID); err != nil {
		return nil, err
	}
	name, err := ensureUniqueFileName(ctx, s.repos.Files, userID, parentID, name, nil)
	if err != nil {
		return nil, err
	}
	dir := &model.File{
		UserID:           userID,
		ParentID:         parentID,
		Name:             name,
		IsDirectory:      true,
		ModerationStatus: "normal",
	}
	if err := s.repos.Files.Create(ctx, dir); err != nil {
		return nil, err
	}
	s.logAction(ctx, userID, "create_directory", "file", &dir.ID, map[string]any{
		"name":      dir.Name,
		"parent_id": uuidToString(parentID),
	})
	return dir, nil
}

// Rename renames a file or directory.
func (s *FileService) Rename(ctx context.Context, fileID uuid.UUID, userID uuid.UUID, newName string) error {
	file, err := s.repos.Files.FindByID(ctx, fileID)
	if err != nil {
		return err
	}
	if file.UserID != userID {
		return ErrPermission
	}
	newName, err = ensureUniqueFileName(ctx, s.repos.Files, userID, file.ParentID, newName, &fileID)
	if err != nil {
		return err
	}
	if err := s.repos.Files.Rename(ctx, fileID, newName); err != nil {
		return err
	}
	s.logAction(ctx, userID, "rename", "file", &fileID, map[string]any{
		"old_name": file.Name,
		"new_name": newName,
	})
	return nil
}

// Move moves a file to a different directory.
func (s *FileService) Move(ctx context.Context, fileID uuid.UUID, userID uuid.UUID, newParentID *uuid.UUID) error {
	file, err := s.repos.Files.FindByID(ctx, fileID)
	if err != nil {
		return err
	}
	if file.UserID != userID {
		return ErrPermission
	}
	if file.DeletedAt != nil {
		return fmt.Errorf("已删除的文件请先从回收站恢复")
	}
	if err := s.assertOwnedDirectory(ctx, userID, newParentID); err != nil {
		return err
	}
	// Prevent moving a folder into itself or a descendant (cycle).
	if newParentID != nil {
		isAnc, err := s.repos.Files.IsAncestor(ctx, fileID, *newParentID, userID)
		if err != nil {
			return err
		}
		if isAnc {
			return fmt.Errorf("不能将文件夹移动到其自身或子目录中")
		}
	}
	if err := s.repos.Files.Move(ctx, fileID, newParentID); err != nil {
		return err
	}
	s.logAction(ctx, userID, "move", "file", &fileID, map[string]any{
		"name":           file.Name,
		"from_parent_id": uuidToString(file.ParentID),
		"to_parent_id":   uuidToString(newParentID),
	})
	return nil
}

// Copy copies a file or folder (recursive) into destParentID.
func (s *FileService) Copy(ctx context.Context, fileID uuid.UUID, userID uuid.UUID, destParentID *uuid.UUID) (*model.File, error) {
	src, err := s.repos.Files.FindByID(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if src.UserID != userID {
		return nil, ErrPermission
	}
	if src.DeletedAt != nil {
		return nil, fmt.Errorf("已删除的文件无法复制")
	}
	if err := s.assertOwnedDirectory(ctx, userID, destParentID); err != nil {
		return nil, err
	}
	// Prevent copying a folder into itself/descendant.
	if src.IsDirectory && destParentID != nil {
		isAnc, err := s.repos.Files.IsAncestor(ctx, fileID, *destParentID, userID)
		if err != nil {
			return nil, err
		}
		if isAnc {
			return nil, fmt.Errorf("不能将文件夹复制到其自身或子目录中")
		}
	}

	if src.IsDirectory {
		return s.copyDirectoryTree(ctx, src, userID, destParentID)
	}
	return s.copySingleFile(ctx, src, userID, destParentID)
}

func (s *FileService) copySingleFile(ctx context.Context, src *model.File, userID uuid.UUID, destParentID *uuid.UUID) (*model.File, error) {
	user, err := s.repos.Users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.StorageUsed+src.Size > user.StorageQuota {
		return nil, fmt.Errorf("存储配额不足，剩余 %d 字节", user.StorageQuota-user.StorageUsed)
	}

	newFile, err := s.copyFileNode(ctx, src, userID, destParentID)
	if err != nil {
		return nil, err
	}
	if src.Size > 0 {
		_ = s.repos.Users.UpdateStorageUsed(ctx, userID, src.Size)
	}
	s.logAction(ctx, userID, "copy", "file", &newFile.ID, map[string]any{
		"source_file_id": src.ID.String(),
		"source_name":    src.Name,
		"copied_name":    newFile.Name,
		"parent_id":      uuidToString(destParentID),
	})
	return newFile, nil
}

func (s *FileService) copyDirectoryTree(ctx context.Context, src *model.File, userID uuid.UUID, destParentID *uuid.UUID) (*model.File, error) {
	nodes, err := s.repos.Files.ListActiveSubtree(ctx, src.ID, userID)
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("源文件夹不存在")
	}

	var needBytes int64
	for i := range nodes {
		if !nodes[i].IsDirectory {
			needBytes += nodes[i].Size
		}
	}
	user, err := s.repos.Users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.StorageUsed+needBytes > user.StorageQuota {
		return nil, fmt.Errorf("存储配额不足，剩余 %d 字节", user.StorageQuota-user.StorageUsed)
	}

	// Map old id -> new id for rewiring parent links.
	idMap := make(map[uuid.UUID]uuid.UUID, len(nodes))
	var rootCopy *model.File

	for i := range nodes {
		n := nodes[i]
		var parent *uuid.UUID
		if n.ID == src.ID {
			parent = destParentID
		} else if n.ParentID != nil {
			if mapped, ok := idMap[*n.ParentID]; ok {
				parent = &mapped
			} else {
				return nil, fmt.Errorf("复制文件夹失败：父节点映射缺失")
			}
		}

		var created *model.File
		if n.IsDirectory {
			name, err := ensureUniqueFileName(ctx, s.repos.Files, userID, parent, n.Name, nil)
			if err != nil {
				return nil, err
			}
			dir := &model.File{
				UserID:           userID,
				ParentID:         parent,
				Name:             name,
				IsDirectory:      true,
				ModerationStatus: "normal",
			}
			if err := s.repos.Files.Create(ctx, dir); err != nil {
				return nil, err
			}
			created = dir
		} else {
			created, err = s.copyFileNode(ctx, &n, userID, parent)
			if err != nil {
				return nil, err
			}
		}
		idMap[n.ID] = created.ID
		if n.ID == src.ID {
			rootCopy = created
		}
	}

	if needBytes > 0 {
		_ = s.repos.Users.UpdateStorageUsed(ctx, userID, needBytes)
	}
	if rootCopy != nil {
		s.logAction(ctx, userID, "copy", "file", &rootCopy.ID, map[string]any{
			"source_file_id": src.ID.String(),
			"source_name":    src.Name,
			"copied_name":    rootCopy.Name,
			"is_directory":   true,
			"nodes":          len(nodes),
			"bytes":          needBytes,
			"parent_id":      uuidToString(destParentID),
		})
	}
	return rootCopy, nil
}

func (s *FileService) copyFileNode(ctx context.Context, src *model.File, userID uuid.UUID, destParentID *uuid.UUID) (*model.File, error) {
	newKey := generateStorageKey(userID.String(), src.Name)
	if src.StorageKey != "" {
		if s.oss == nil {
			return nil, fmt.Errorf("storage unavailable")
		}
		if err := s.oss.CopyObject(ctx, src.StorageKey, newKey); err != nil {
			return nil, fmt.Errorf("failed to copy file in storage: %w", err)
		}
	}

	newThumb := ""
	if src.ThumbnailKey != "" && s.oss != nil {
		newThumb = generateStorageKey(userID.String(), "thumb-"+src.Name)
		if err := s.oss.CopyObject(ctx, src.ThumbnailKey, newThumb); err != nil {
			newThumb = ""
		}
	}

	publicURL := fmt.Sprintf("%s/s/%s/%s", s.cfg.PublicBaseURL, userID.String(), filepath.Base(newKey))
	name, err := ensureUniqueFileName(ctx, s.repos.Files, userID, destParentID, src.Name, nil)
	if err != nil {
		return nil, err
	}

	newFile := &model.File{
		UserID:           userID,
		ParentID:         destParentID,
		Name:             name,
		StorageKey:       newKey,
		IsDirectory:      false,
		MimeType:         src.MimeType,
		Size:             src.Size,
		PublicURL:        publicURL,
		ThumbnailKey:     newThumb,
		ModerationStatus: "normal",
	}
	if err := s.repos.Files.Create(ctx, newFile); err != nil {
		return nil, err
	}
	return newFile, nil
}

// SoftDelete moves a file or folder (recursive) to trash.
// Soft-delete of the tree is one SQL statement (atomic); quota update follows.
func (s *FileService) SoftDelete(ctx context.Context, fileID uuid.UUID, userID uuid.UUID) error {
	file, err := s.repos.Files.FindByID(ctx, fileID)
	if err != nil {
		return err
	}
	if file.UserID != userID {
		return ErrPermission
	}
	if file.DeletedAt != nil {
		return nil
	}

	// Atomic subtree soft-delete + size sum in one statement.
	freed, err := s.repos.Files.SoftDeleteSubtree(ctx, fileID, userID)
	if err != nil {
		return err
	}
	if freed > 0 {
		if err := s.repos.Users.UpdateStorageUsed(ctx, userID, -freed); err != nil {
			// Soft-delete already committed; best-effort recalc for this user.
			_ = s.repos.Users.RecalcStorageUsed(ctx, userID)
		}
	}
	s.logAction(ctx, userID, "delete", "file", &fileID, map[string]any{
		"name":         file.Name,
		"is_directory": file.IsDirectory,
		"freed_bytes":  freed,
	})
	return nil
}

// Restore restores a file or folder from trash (with unique rename if needed).
func (s *FileService) Restore(ctx context.Context, fileID uuid.UUID, userID uuid.UUID) error {
	file, err := s.repos.Files.FindByID(ctx, fileID)
	if err != nil {
		return err
	}
	if file.UserID != userID {
		return ErrPermission
	}
	if file.DeletedAt == nil {
		return nil
	}

	needBytes, err := s.repos.Files.SumDeletedSubtreeSize(ctx, fileID, userID)
	if err != nil {
		return err
	}
	if needBytes > 0 {
		user, uerr := s.repos.Users.FindByID(ctx, userID)
		if uerr != nil {
			return uerr
		}
		if user.StorageUsed+needBytes > user.StorageQuota {
			return fmt.Errorf("存储配额不足，无法从回收站恢复")
		}
	}

	// Resolve name conflicts in the parent before restoring.
	uniqueName, err := ensureUniqueFileName(ctx, s.repos.Files, userID, file.ParentID, file.Name, &fileID)
	if err != nil {
		return err
	}

	// Single-statement restore of the whole deleted subtree (+ optional rename of root).
	if err := s.repos.Files.RestoreSubtree(ctx, fileID, userID, uniqueName); err != nil {
		return err
	}

	if needBytes > 0 {
		if err := s.repos.Users.UpdateStorageUsed(ctx, userID, needBytes); err != nil {
			_ = s.repos.Users.RecalcStorageUsed(ctx, userID)
		}
	}
	s.logAction(ctx, userID, "restore", "file", &fileID, map[string]any{
		"name":           uniqueName,
		"restored_bytes": needBytes,
	})
	return nil
}

// PermanentDelete permanently deletes a file/folder tree and OSS objects.
func (s *FileService) PermanentDelete(ctx context.Context, fileID uuid.UUID, userID uuid.UUID) error {
	file, err := s.repos.Files.FindByID(ctx, fileID)
	if err != nil {
		return err
	}
	if file.UserID != userID {
		return ErrPermission
	}

	subtree, err := s.repos.Files.ListSubtree(ctx, fileID, userID)
	if err != nil {
		return err
	}

	var activeBytes int64
	for i := range subtree {
		f := subtree[i]
		if f.DeletedAt == nil && !f.IsDirectory && f.Size > 0 {
			activeBytes += f.Size
		}
		if s.oss != nil {
			if f.StorageKey != "" {
				_ = s.oss.DeleteObject(ctx, f.StorageKey)
			}
			if f.ThumbnailKey != "" {
				_ = s.oss.DeleteObject(ctx, f.ThumbnailKey)
			}
		}
	}

	// Delete root (CASCADE removes descendants in DB).
	if err := s.repos.Files.PermanentDeleteSubtree(ctx, fileID, userID); err != nil {
		return err
	}
	if activeBytes > 0 {
		_ = s.repos.Users.UpdateStorageUsed(ctx, userID, -activeBytes)
	}
	s.logAction(ctx, userID, "permanent_delete", "file", &fileID, map[string]any{
		"name":         file.Name,
		"nodes":        len(subtree),
		"freed_bytes":  activeBytes,
	})
	return nil
}

// ListTrash lists files in the user's trash.
func (s *FileService) ListTrash(ctx context.Context, userID uuid.UUID) ([]model.File, error) {
	return s.repos.Files.ListTrash(ctx, userID)
}

// CleanupTrash removes files that have been in trash for too long (DB + OSS).
// Quota is not adjusted for soft-deleted trees (already released).
func (s *FileService) CleanupTrash(ctx context.Context) (int64, error) {
	files, err := s.repos.Files.ListExpiredTrash(ctx, s.cfg.TrashRetentionDays)
	if err != nil {
		return 0, err
	}
	var n int64
	for i := range files {
		f := files[i]
		// Skip children that will be cleaned with a parent still in trash list —
		// ListExpiredTrash returns all expired rows; permanent-delete each root-like node safely.
		if err := s.permanentDeleteInternal(ctx, f.ID, f.UserID); err != nil {
			continue
		}
		n++
	}
	return n, nil
}

// permanentDeleteInternal is PermanentDelete without ownership re-check by caller identity.
func (s *FileService) permanentDeleteInternal(ctx context.Context, fileID, ownerID uuid.UUID) error {
	file, err := s.repos.Files.FindByID(ctx, fileID)
	if err != nil {
		return err
	}
	// Already gone
	if file == nil {
		return nil
	}
	subtree, err := s.repos.Files.ListSubtree(ctx, fileID, ownerID)
	if err != nil {
		return err
	}
	var activeBytes int64
	for i := range subtree {
		f := subtree[i]
		if f.DeletedAt == nil && !f.IsDirectory && f.Size > 0 {
			activeBytes += f.Size
		}
		if s.oss != nil {
			if f.StorageKey != "" {
				_ = s.oss.DeleteObject(ctx, f.StorageKey)
			}
			if f.ThumbnailKey != "" {
				_ = s.oss.DeleteObject(ctx, f.ThumbnailKey)
			}
		}
	}
	if err := s.repos.Files.PermanentDeleteSubtree(ctx, fileID, ownerID); err != nil {
		return err
	}
	if activeBytes > 0 {
		_ = s.repos.Users.UpdateStorageUsed(ctx, ownerID, -activeBytes)
	}
	return nil
}

// assertOwnedDirectory ensures parentID is nil (root) or an owned non-deleted directory.
func (s *FileService) assertOwnedDirectory(ctx context.Context, userID uuid.UUID, parentID *uuid.UUID) error {
	if parentID == nil {
		return nil
	}
	parent, err := s.repos.Files.FindByID(ctx, *parentID)
	if err != nil {
		return fmt.Errorf("目标文件夹不存在")
	}
	if parent.UserID != userID {
		return ErrPermission
	}
	if parent.DeletedAt != nil {
		return fmt.Errorf("目标文件夹已在回收站中")
	}
	if !parent.IsDirectory {
		return fmt.Errorf("目标必须是文件夹")
	}
	return nil
}

// GetFileURL returns the direct link URL and markdown reference for a file.
func (s *FileService) GetFileURL(ctx context.Context, fileID uuid.UUID, userID uuid.UUID) (string, string, error) {
	file, err := s.repos.Files.FindByID(ctx, fileID)
	if err != nil {
		return "", "", err
	}
	if file.UserID != userID {
		return "", "", ErrPermission
	}

	return buildFileURLResponse(file)
}

func (s *FileService) GetFileURLForActor(ctx context.Context, fileID uuid.UUID, actor *model.User) (string, string, error) {
	file, err := s.repos.Files.FindByID(ctx, fileID)
	if err != nil {
		return "", "", err
	}
	if actor == nil {
		return "", "", ErrPermission
	}
	if actor.Role != "admin" && file.UserID != actor.ID {
		return "", "", ErrPermission
	}

	return buildFileURLResponse(file)
}

// ProxyFile returns a reader for the file content from OSS.
func (s *FileService) ProxyFile(ctx context.Context, storageKey string) (io.ReadCloser, string, int64, error) {
	return s.oss.GetObject(ctx, storageKey)
}

func (s *FileService) ProxyThumbnail(ctx context.Context, storageKey, style string) (io.ReadCloser, string, int64, error) {
	return s.oss.GetStyledObject(ctx, storageKey, style)
}

// ResolvePublicFile returns a file record by direct-link path segments.
func (s *FileService) ResolvePublicFile(ctx context.Context, userIDStr, filename string) (*model.File, error) {
	return s.repos.Files.FindByStorageKey(ctx, userIDStr+"/"+filename)
}

// SubmitAppeal creates an appeal ticket for a banned file.
func (s *FileService) SubmitAppeal(ctx context.Context, fileID, userID uuid.UUID, reason string) (*model.FileAppeal, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, fmt.Errorf("申诉理由不能为空")
	}
	if len([]rune(reason)) > 1000 {
		return nil, fmt.Errorf("申诉理由不能超过 1000 字")
	}

	file, err := s.repos.Files.FindByID(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if file.UserID != userID {
		return nil, ErrPermission
	}
	if file.ModerationStatus != "banned" {
		return nil, fmt.Errorf("该文件当前未被封禁")
	}

	if _, err := s.repos.FileAppeals.FindPendingByFileID(ctx, fileID); err == nil {
		return nil, fmt.Errorf("该文件已有待处理申诉，请勿重复提交")
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	appeal := &model.FileAppeal{
		FileID: fileID,
		UserID: userID,
		Status: "pending",
		Reason: reason,
	}
	if err := s.repos.FileAppeals.Create(ctx, appeal); err != nil {
		return nil, err
	}

	_ = s.repos.AuditLogs.Create(ctx, &model.AuditLog{
		UserID:     userID,
		Action:     "file_appeal_submitted",
		Resource:   "file",
		ResourceID: &fileID,
		Details: map[string]any{
			"reason": reason,
		},
	})

	return appeal, nil
}

// generateStorageKey creates a random storage key for a file.
func generateStorageKey(userID, originalName string) string {
	randBytes := make([]byte, 16)
	_, _ = rand.Read(randBytes)
	randomName := hex.EncodeToString(randBytes)
	ext := filepath.Ext(originalName)
	return userID + "/" + randomName + ext
}

func ensureUniqueFileName(ctx context.Context, repo *repository.FileRepository, userID uuid.UUID, parentID *uuid.UUID, desired string, excludeID *uuid.UUID) (string, error) {
	name := strings.TrimSpace(desired)
	if name == "" {
		return "", fmt.Errorf("名称不能为空")
	}

	base := strings.TrimSuffix(name, filepath.Ext(name))
	ext := filepath.Ext(name)
	if base == "" {
		base = name
		ext = ""
	}

	candidate := name
	index := 2
	for {
		exists, err := repo.ExistsByName(ctx, userID, parentID, candidate, excludeID)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s (%d)%s", base, index, ext)
		index++
	}
}

func (s *FileService) logAction(ctx context.Context, userID uuid.UUID, action, resource string, resourceID *uuid.UUID, details map[string]any) {
	if s == nil || s.repos == nil || s.repos.AuditLogs == nil {
		return
	}
	_ = s.repos.AuditLogs.Create(ctx, &model.AuditLog{
		UserID:     userID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Details:    details,
	})
}

func uuidToString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

func buildFileURLResponse(file *model.File) (string, string, error) {
	if file == nil {
		return "", "", fmt.Errorf("file is nil")
	}
	url := file.PublicURL
	markdown := fmt.Sprintf("[%s](%s)", file.Name, url)

	if strings.HasPrefix(file.MimeType, "image/") {
		markdown = fmt.Sprintf("![%s](%s)", file.Name, url)
	}

	return url, markdown, nil
}
