package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Медиа по умолчанию лежат в бакете (их загружает minio-init из assets/).
const (
	DefaultImageKey = "provider_routers/draft.svg"
	DefaultVideoKey = "videos/draft-loop.mp4"
)

// MediaKind — вид файла маршрутизатора: изображение или короткое видео.
type MediaKind struct {
	Prefix     string            // папка в бакете
	Extensions map[string]string // расширение -> Content-Type
	MaxBytes   int64
}

var (
	ProviderRouterImage = MediaKind{
		Prefix:     "provider_routers",
		Extensions: map[string]string{".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png", ".webp": "image/webp", ".svg": "image/svg+xml"},
		MaxBytes:   10 << 20, // 10 МБ
	}
	ProviderRouterVideo = MediaKind{
		Prefix:     "videos",
		Extensions: map[string]string{".mp4": "video/mp4", ".webm": "video/webm", ".mov": "video/quicktime"},
		MaxBytes:   50 << 20, // 50 МБ
	}
)

// Media — клиент MinIO для файлов маршрутизаторов.
type Media struct {
	client    *minio.Client
	bucket    string
	publicURL string
}

func NewMedia(endpoint, accessKey, secretKey, bucket, publicURL string) (*Media, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
		Region: "us-east-1",
	})
	if err != nil {
		return nil, err
	}
	return &Media{client: client, bucket: bucket, publicURL: publicURL}, nil
}

// EnsureBucket создаёт бакет с публичным чтением, если его ещё нет.
func (m *Media) EnsureBucket(ctx context.Context) error {
	exists, err := m.client.BucketExists(ctx, m.bucket)
	if err != nil || exists {
		return err
	}
	if err := m.client.MakeBucket(ctx, m.bucket, minio.MakeBucketOptions{}); err != nil {
		return err
	}
	policy := fmt.Sprintf(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"AWS":["*"]},"Action":["s3:GetObject"],"Resource":["arn:aws:s3:::%s/*"]}]}`, m.bucket)
	return m.client.SetBucketPolicy(ctx, m.bucket, policy)
}

// ValidateFile проверяет расширение и размер файла до загрузки.
func (kind MediaKind) ValidateFile(header *multipart.FileHeader) (ext, contentType string, err error) {
	ext = strings.ToLower(filepath.Ext(header.Filename))
	contentType, ok := kind.Extensions[ext]
	if !ok {
		allowed := make([]string, 0, len(kind.Extensions))
		for e := range kind.Extensions {
			allowed = append(allowed, e)
		}
		sort.Strings(allowed)
		return "", "", fmt.Errorf("недопустимое расширение %q, разрешены: %s", ext, strings.Join(allowed, ", "))
	}
	if header.Size <= 0 {
		return "", "", fmt.Errorf("файл %q пустой", header.Filename)
	}
	if header.Size > kind.MaxBytes {
		return "", "", fmt.Errorf("файл %q больше %d МБ", header.Filename, kind.MaxBytes>>20)
	}
	return ext, contentType, nil
}

// Upload сохраняет файл в MinIO под новым латинским именем и возвращает ключ,
// например provider_routers/router-20261017-153000-a1b2c3d4.png.
func (m *Media) Upload(ctx context.Context, kind MediaKind, header *multipart.FileHeader) (string, error) {
	ext, contentType, err := kind.ValidateFile(header)
	if err != nil {
		return "", err
	}
	file, err := header.Open()
	if err != nil {
		return "", err
	}
	defer file.Close()
	key := kind.Prefix + "/" + generateObjectName(ext)
	_, err = m.client.PutObject(ctx, m.bucket, key, io.LimitReader(file, header.Size), header.Size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return "", fmt.Errorf("не удалось сохранить файл в MinIO: %w", err)
	}
	return key, nil
}

// Remove удаляет объект (используется для отката, если запись в БД не создалась).
func (m *Media) Remove(ctx context.Context, key string) {
	if key != "" {
		_ = m.client.RemoveObject(ctx, m.bucket, key, minio.RemoveObjectOptions{})
	}
}

// URL строит публичную ссылку на объект; для пустого ключа — файл по умолчанию.
func (m *Media) URL(key, fallback string) string {
	if strings.TrimSpace(key) == "" {
		key = fallback
	}
	return m.publicURL + "/" + key
}

func generateObjectName(ext string) string {
	random := make([]byte, 4)
	_, _ = rand.Read(random)
	return "router-" + time.Now().Format("20060102-150405") + "-" + hex.EncodeToString(random) + ext
}
