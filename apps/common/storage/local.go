package storage

import (
	"os"
	"path/filepath"
	"strings"
)

// LocalStorage 本地檔案系統存儲（預設實現）
type LocalStorage struct{}

// Upload 本地存儲直接返回相對路徑（檔案已保存在本地）
func (ls *LocalStorage) Upload(localPath, objectKey string) (string, error) {
	// 本地存儲無需上傳，檔案已在本地
	relPath := filepath.ToSlash(localPath)
	return relPath, nil
}

// Delete 刪除本地檔案
func (ls *LocalStorage) Delete(objectKey string) error {
	// objectKey 即為相對路徑
	if err := os.Remove(objectKey); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// GetURL 取得本地檔案 URL（相對路徑）
func (ls *LocalStorage) GetURL(objectKey string) string {
	if !strings.HasPrefix(objectKey, "/") {
		return "/" + objectKey
	}
	return objectKey
}

// Exists 檢查本地檔案是否存在
func (ls *LocalStorage) Exists(objectKey string) bool {
	_, err := os.Stat(objectKey)
	return err == nil
}

// IsEnabled 本地存儲始終啟用
func (ls *LocalStorage) IsEnabled() bool {
	return true
}
