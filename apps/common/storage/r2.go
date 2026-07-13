package storage

import (
	"context"
	"fmt"
	"log/slog"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gbootcms/apps/admin/model"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// R2Storage Cloudflare R2 雲存儲實現
type R2Storage struct {
	client    *minio.Client
	bucket    string
	publicURL string
}

// NewR2Storage 建立新的 R2 存儲實例
func NewR2Storage() (*R2Storage, error) {
	endpoint := model.GetConfigValue("r2_endpoint", "")
	accessKey := model.GetConfigValue("r2_access_key", "")
	secretKey := model.GetConfigValue("r2_secret_key", "")
	bucket := model.GetConfigValue("r2_bucket", "")
	publicURL := model.GetConfigValue("r2_public_url", "")

	if endpoint == "" || accessKey == "" || secretKey == "" || bucket == "" {
		return nil, fmt.Errorf("R2 配置不完整：需要 endpoint, access_key, secret_key, bucket")
	}

	// 移除 endpoint 的協議前綴（minio-go 需要 host:port 格式）
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")

	client, err := minio.New(endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(accessKey, secretKey, ""),
		Region:       "auto",
		Secure:       true, // R2 使用 HTTPS
		BucketLookup: minio.BucketLookupAuto,
	})
	if err != nil {
		return nil, fmt.Errorf("R2 客戶端建立失敗: %w", err)
	}

	// 確保 bucket 存在（不重複建立）
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, fmt.Errorf("R2 連接失敗: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("R2 bucket 不存在: %s", bucket)
	}

	// 正規化 publicURL
	publicURL = strings.TrimSuffix(publicURL, "/")

	slog.Info("R2 存儲連接成功", "endpoint", endpoint, "bucket", bucket)

	return &R2Storage{
		client:    client,
		bucket:    bucket,
		publicURL: publicURL,
	}, nil
}

// Upload 上傳本地檔案到 R2，返回公開 URL
func (r *R2Storage) Upload(localPath, objectKey string) (string, error) {
	// 檢查快取
	if cached := cache.getCached(objectKey); cached != nil && cached.exists {
		return cached.url, nil
	}

	// 開啟本地檔案
	file, err := os.Open(localPath)
	if err != nil {
		return "", fmt.Errorf("開啟本地檔案失敗: %w", err)
	}
	defer file.Close()

	// 取得檔案資訊
	stat, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("取得檔案資訊失敗: %w", err)
	}

	// 推斷 Content-Type
	contentType := mime.TypeByExtension(filepath.Ext(localPath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// 上傳到 R2
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// objectKey 是相對路徑（如 static/upload/202607/xxx.jpg）
	// 在 R2 中，我們使用去掉 "static/" 前綴的路徑作為 object key
	r2Key := strings.TrimPrefix(objectKey, "static/")

	_, err = r.client.PutObject(ctx, r.bucket, r2Key, file, stat.Size(),
		minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return "", fmt.Errorf("R2 上傳失敗: %w", err)
	}

	// 構建公開 URL
	publicURL := r.GetURL(objectKey)

	// 更新快取
	cache.setCached(objectKey, publicURL, true)

	slog.Info("R2 上傳成功", "key", r2Key, "url", publicURL)
	return publicURL, nil
}

// Delete 刪除 R2 上的檔案
func (r *R2Storage) Delete(objectKey string) error {
	r2Key := strings.TrimPrefix(objectKey, "static/")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := r.client.RemoveObject(ctx, r.bucket, r2Key, minio.RemoveObjectOptions{})
	if err != nil {
		slog.Warn("R2 刪除失敗", "key", r2Key, "error", err)
		return err
	}

	// 使快取失效
	cache.invalidate(objectKey)
	return nil
}

// GetURL 取得 R2 檔案的公開 URL
func (r *R2Storage) GetURL(objectKey string) string {
	r2Key := strings.TrimPrefix(objectKey, "static/")
	if r.publicURL != "" {
		return r.publicURL + "/" + r2Key
	}
	// 如果沒有配置 publicURL，使用 R2 預設 URL 格式
	return fmt.Sprintf("https://%s/%s/%s", strings.TrimPrefix(model.GetConfigValue("r2_endpoint", ""), "https://"), r.bucket, r2Key)
}

// Exists 檢查檔案是否存在於 R2
func (r *R2Storage) Exists(objectKey string) bool {
	// 先檢查快取
	if cached := cache.getCached(objectKey); cached != nil {
		return cached.exists
	}

	r2Key := strings.TrimPrefix(objectKey, "static/")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.client.StatObject(ctx, r.bucket, r2Key, minio.StatObjectOptions{})
	exists := err == nil

	// 更新快取
	cache.setCached(objectKey, r.GetURL(objectKey), exists)
	return exists
}

// IsEnabled R2 存儲是否啟用
func (r *R2Storage) IsEnabled() bool {
	return r.client != nil
}
