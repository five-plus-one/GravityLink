package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gravitylink/backend/internal/model"
	"image"
	"image/color"
	"image/png"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const storageConfigKey = "storage.s3.encrypted"

var ErrStorage = errors.New("图片存储操作失败，请检查配置和存储桶权限")

// ErrStorageUndecryptable 表示已存密文无法用当前 wechat.key 解开（常见于只迁库未迁数据目录）。
var ErrStorageUndecryptable = errors.New("已保存的存储配置无法解密，请填写 Secret Access Key 后重新保存")

type StorageConfig struct {
	Enabled          bool   `json:"enabled"`
	Endpoint         string `json:"endpoint"`
	Region           string `json:"region"`
	Bucket           string `json:"bucket"`
	Prefix           string `json:"prefix"`
	CDN              string `json:"cdn"`
	AccessKey        string `json:"access_key"`
	SecretKey        string `json:"secret_key,omitempty"`
	PathStyle        bool   `json:"path_style"`
	SecretConfigured bool   `json:"secret_configured"`
}
type StorageService struct {
	db  *gorm.DB
	dir string
}

func NewStorageService(db *gorm.DB, dir string) *StorageService {
	return &StorageService{db: db, dir: dir}
}
func (s *StorageService) config(ctx context.Context) (StorageConfig, error) {
	var result StorageConfig
	var row model.SystemConfig
	err := s.db.WithContext(ctx).Where("key_name = ?", storageConfigKey).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	key, err := NewWechatService(s.db, s.dir).key()
	if err != nil {
		return result, ErrStorageUndecryptable
	}
	plain, err := openSecret(key, row.Value)
	if err != nil {
		return result, ErrStorageUndecryptable
	}
	if err = json.Unmarshal([]byte(plain), &result); err != nil {
		return StorageConfig{}, ErrStorageUndecryptable
	}
	return result, nil
}

// configForOverwrite: 旧密文解不开时，仅当表单已带 Secret 才允许覆盖。
func configForOverwrite(old StorageConfig, err error, secret string) (StorageConfig, error) {
	if err == nil {
		return old, nil
	}
	if !errors.Is(err, ErrStorageUndecryptable) {
		return StorageConfig{}, err
	}
	if strings.TrimSpace(secret) == "" {
		return StorageConfig{}, ErrStorageUndecryptable
	}
	return StorageConfig{}, nil
}

func (s *StorageService) Status(ctx context.Context) (StorageConfig, error) {
	c, err := s.config(ctx)
	if errors.Is(err, ErrStorageUndecryptable) {
		return StorageConfig{}, nil
	}
	if err != nil {
		return c, err
	}
	c.SecretConfigured = c.SecretKey != ""
	c.SecretKey = ""
	return c, nil
}
func normalizeStorage(c StorageConfig) StorageConfig {
	c.Endpoint = strings.TrimRight(strings.TrimSpace(c.Endpoint), "/")
	c.Region = strings.TrimSpace(c.Region)
	c.Bucket = strings.TrimSpace(c.Bucket)
	u, err := url.Parse(c.Endpoint)
	if err == nil && strings.HasSuffix(u.Hostname(), ".aliyuncs.com") {
		host := u.Hostname()
		if strings.HasPrefix(host, "oss-") {
			u.Host = "s3." + host
			c.Endpoint = u.String()
		}
		if strings.HasPrefix(u.Hostname(), "s3.oss-") {
			c.PathStyle = false
			c.Region = strings.TrimPrefix(c.Region, "oss-")
		}
	}
	return c
}
func validateStorage(c StorageConfig) error {
	if !c.Enabled {
		return nil
	}
	for _, raw := range []string{c.Endpoint, c.CDN} {
		u, e := url.Parse(raw)
		if e != nil || u.Host == "" || u.User != nil || (u.Scheme != "https" && u.Scheme != "http") || u.RawQuery != "" || u.Fragment != "" {
			return ErrStorage
		}
	}
	u, _ := url.Parse(c.Endpoint)
	if strings.HasPrefix(u.Hostname(), "s3.oss-") && strings.HasSuffix(u.Hostname(), ".aliyuncs.com") {
		region := strings.TrimSuffix(strings.TrimPrefix(u.Hostname(), "s3.oss-"), ".aliyuncs.com")
		region = strings.TrimSuffix(region, "-internal")
		if region != "accelerate" && c.Region != region {
			return fmt.Errorf("地域应为 %s，请检查 Region 与存储桶所在地", region)
		}
	}
	if strings.Trim(u.Path, "/") != "" {
		return ErrStorage
	}
	if c.Bucket == "" || strings.ContainsAny(c.Bucket, "/\\ .") || c.Region == "" || c.AccessKey == "" || c.SecretKey == "" || strings.Contains(c.Prefix, "..") || strings.ContainsAny(c.Prefix, "\\?#") {
		return ErrStorage
	}
	return nil
}
func (s *StorageService) Save(ctx context.Context, c StorageConfig) error {
	c = normalizeStorage(c)
	old, err := s.config(ctx)
	old, err = configForOverwrite(old, err, c.SecretKey)
	if err != nil {
		return err
	}
	if c.SecretKey == "" {
		c.SecretKey = old.SecretKey
	}
	c.Endpoint = strings.TrimRight(strings.TrimSpace(c.Endpoint), "/")
	c.CDN = strings.TrimRight(strings.TrimSpace(c.CDN), "/")
	c.Prefix = strings.Trim(c.Prefix, "/")
	if err := validateStorage(c); err != nil {
		return err
	}
	key, err := NewWechatService(s.db, s.dir).key()
	if err != nil {
		return ErrStorage
	}
	plain, err := json.Marshal(c)
	if err != nil {
		return err
	}
	sealed, err := sealSecret(key, string(plain))
	if err != nil {
		return ErrStorage
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "key_name"}}, DoUpdates: clause.AssignmentColumns([]string{"value"})}).Create(&model.SystemConfig{KeyName: storageConfigKey, Value: sealed}).Error
}
func storagePut(ctx context.Context, c StorageConfig, name string, b []byte) (string, error) {
	c = normalizeStorage(c)
	if err := validateStorage(c); err != nil {
		return "", err
	}
	endpoint, _ := url.Parse(c.Endpoint)
	lookup := minio.BucketLookupDNS
	if c.PathStyle {
		lookup = minio.BucketLookupPath
	}
	client, err := minio.New(endpoint.Host, &minio.Options{Creds: credentials.NewStaticV4(c.AccessKey, c.SecretKey, ""), Secure: endpoint.Scheme == "https", Region: c.Region, BucketLookup: lookup, Transport: http.DefaultTransport})
	if err != nil {
		return "", ErrStorage
	}
	key := path.Join(c.Prefix, name)
	u, _ := url.Parse(c.CDN)
	u.Path = strings.TrimRight(u.Path, "/") + "/" + key
	if len(u.String()) > 255 {
		return "", ErrStorage
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	_, err = client.PutObject(ctx, c.Bucket, key, bytes.NewReader(b), int64(len(b)), minio.PutObjectOptions{ContentType: http.DetectContentType(b), DisableMultipart: true, DisableContentSha256: true, SendContentMd5: true})
	if err != nil {
		return "", storageError(err)
	}
	return u.String(), nil
}
func (s *StorageService) Upload(ctx context.Context, name string, b []byte) (string, error) {
	c, err := s.config(ctx)
	if err != nil {
		return "", err
	}
	if c.Enabled {
		return storagePut(ctx, c, name, b)
	}
	dir := filepath.Join(s.dir, "uploads")
	if err = os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	if err = os.WriteFile(filepath.Join(dir, name), b, 0600); err != nil {
		return "", err
	}
	return "/uploads/" + name, nil
}
func (s *StorageService) Migrate(ctx context.Context, id uint64) (model.Material, error) {
	var item model.Material
	c, err := s.config(ctx)
	if err != nil || !c.Enabled {
		return item, ErrStorage
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&item, id).Error; err != nil {
			return err
		}
		if !strings.HasPrefix(item.Path, "/uploads/") {
			return nil
		}
		name := strings.TrimPrefix(item.Path, "/uploads/")
		if filepath.Base(name) != name || strings.Contains(name, "..") {
			return ErrStorage
		}
		b, err := os.ReadFile(filepath.Join(s.dir, "uploads", name))
		if err != nil {
			return fmt.Errorf("本地图片无法读取")
		}
		address, err := storagePut(ctx, c, name, b)
		if err != nil {
			return err
		}
		item.Path = address
		return tx.Model(&item).Update("path", address).Error
	})
	return item, err
}

func storageError(err error) error {
	r := minio.ToErrorResponse(err)
	switch r.Code {
	case "NoSuchBucket":
		return errors.New("存储桶不存在，请检查 Bucket 名称和地域")
	case "InvalidAccessKeyId":
		return errors.New("访问密钥 ID 无效，请检查 Access Key ID")
	case "AccessDenied":
		return errors.New("存储桶拒绝写入，请检查访问密钥的上传权限")
	case "SignatureDoesNotMatch", "InvalidSecurity":
		return errors.New("签名校验失败，请检查密钥、地域与 S3 Endpoint")
	case "AuthorizationHeaderMalformed", "PermanentRedirect", "IncorrectEndpoint":
		return errors.New("存储地址或地域不匹配，请检查 Endpoint 和 Region")
	case "RequestTimeTooSkewed":
		return errors.New("服务器时间偏差过大，请同步系统时间")
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return errors.New("无法连接存储服务，请检查地址与网络")
	}
	return ErrStorage
}

type StorageCheck struct {
	Uploaded   bool   `json:"uploaded"`
	Accessible bool   `json:"accessible"`
	Message    string `json:"message"`
	URL        string `json:"url,omitempty"`
}

func (s *StorageService) Check(ctx context.Context, c StorageConfig) (StorageCheck, error) {
	old, err := s.config(ctx)
	old, err = configForOverwrite(old, err, c.SecretKey)
	if err != nil {
		return StorageCheck{}, err
	}
	if c.SecretKey == "" {
		c.SecretKey = old.SecretKey
	}
	c.Enabled = true
	picture := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			shade := color.RGBA{15, 118, 110, 255}
			if x >= 14 && x < 50 && y >= 14 && y < 50 {
				shade = color.RGBA{240, 253, 250, 255}
			}
			picture.Set(x, y, shade)
		}
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, picture); err != nil {
		return StorageCheck{}, ErrStorage
	}
	data := buffer.Bytes()
	id := make([]byte, 16)
	if _, err = rand.Read(id); err != nil {
		return StorageCheck{}, ErrStorage
	}
	name := "check-" + hex.EncodeToString(id) + ".png"
	address, err := storagePut(ctx, c, name, data)
	if err != nil {
		return StorageCheck{}, err
	}
	if err = s.db.WithContext(ctx).Create(&model.Material{Name: "上传检测.png", Path: address}).Error; err != nil {
		return StorageCheck{}, ErrStorage
	}
	result := StorageCheck{Uploaded: true, URL: address, Message: "上传成功，CDN 暂时无法读取，请检查域名回源和访问权限"}
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return result, nil
	}
	resp, err := client.Do(req)
	if err != nil {
		return result, nil
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err == nil && resp.StatusCode == 200 && bytes.Equal(b, data) {
		result.Accessible = true
		result.Message = "上传和 CDN 访问均正常"
	}
	return result, nil
}

// DeleteMaterial removes the stored object before removing its library record.
func (s *StorageService) DeleteMaterial(ctx context.Context, id uint64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item model.Material
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&item, id).Error; err != nil {
			return errors.New("素材不存在")
		}
		name := ""
		if strings.HasPrefix(item.Path, "/uploads/") {
			name = strings.TrimPrefix(item.Path, "/uploads/")
		} else {
			c, err := s.config(ctx)
			if err != nil {
				return ErrStorage
			}
			c = normalizeStorage(c)
			root, err := url.Parse(c.CDN)
			if err != nil || root.Host == "" {
				return errors.New("请先配置图片所属的存储服务")
			}
			target, err := url.Parse(item.Path)
			if err != nil || target.Scheme != root.Scheme || target.Host != root.Host || target.RawQuery != "" {
				return errors.New("图片属于其他存储服务，请切换到对应配置后删除")
			}
			prefix := strings.TrimRight(root.Path, "/") + "/"
			if !strings.HasPrefix(target.Path, prefix) {
				return errors.New("图片不属于当前存储目录")
			}
			key := strings.TrimPrefix(target.Path, prefix)
			objectPrefix := strings.Trim(c.Prefix, "/")
			if objectPrefix != "" && !strings.HasPrefix(key, objectPrefix+"/") {
				return errors.New("图片不属于当前存储目录")
			}
			if key == "" || path.Clean(key) != key || strings.Contains(key, "..") {
				return errors.New("图片地址无效")
			}
			c.Enabled = true
			if err := validateStorage(c); err != nil {
				return err
			}
			endpoint, _ := url.Parse(c.Endpoint)
			lookup := minio.BucketLookupDNS
			if c.PathStyle {
				lookup = minio.BucketLookupPath
			}
			client, err := minio.New(endpoint.Host, &minio.Options{Creds: credentials.NewStaticV4(c.AccessKey, c.SecretKey, ""), Secure: endpoint.Scheme == "https", Region: c.Region, BucketLookup: lookup, Transport: http.DefaultTransport})
			if err != nil {
				return ErrStorage
			}
			timeout, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			if err = client.RemoveObject(timeout, c.Bucket, key, minio.RemoveObjectOptions{}); err != nil {
				return fmt.Errorf("删除存储文件失败，请检查存储桶删除权限")
			}
			name = path.Base(key)
		}
		if name == "" || filepath.Base(name) != name || strings.Contains(name, "..") || strings.ContainsAny(name, "/\\") {
			return errors.New("图片地址无效")
		}
		directory, err := filepath.Abs(filepath.Join(s.dir, "uploads"))
		if err != nil {
			return err
		}
		file := filepath.Join(directory, name)
		if filepath.Dir(file) != directory {
			return errors.New("图片地址无效")
		}
		if err = os.Remove(file); err != nil && !os.IsNotExist(err) {
			return errors.New("本地图片删除失败，请检查文件权限")
		}
		return tx.Delete(&item).Error
	})
}
