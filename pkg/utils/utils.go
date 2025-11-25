// Package utils 提供时间/ID（雪花）/hash/serialize/retry/backoff/pagination/errorwrap 等通用工具
package utils

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// SnowflakeID 雪花算法 ID 生成器
type SnowflakeID struct {
	mu        sync.Mutex
	timestamp int64
	sequence  int64
	nodeID    int64
}

// NewSnowflakeID 创建雪花 ID 生成器
func NewSnowflakeID(nodeID int64) *SnowflakeID {
	return &SnowflakeID{
		timestamp: 0,
		sequence:  0,
		nodeID:    nodeID & 0x3FF, // 10 bits
	}
}

// Generate 生成雪花 ID
func (s *SnowflakeID) Generate() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UnixMilli()

	if now == s.timestamp {
		s.sequence = (s.sequence + 1) & 0xFFF // 12 bits
		if s.sequence == 0 {
			// 等待下一毫秒
			for now <= s.timestamp {
				now = time.Now().UnixMilli()
			}
		}
	} else {
		s.sequence = 0
	}

	s.timestamp = now

	// 组合 ID：timestamp(41 bits) + nodeID(10 bits) + sequence(12 bits)
	return (now << 22) | (s.nodeID << 12) | s.sequence
}

// MD5Hash 计算 MD5 哈希
func MD5Hash(data string) string {
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

// SHA256Hash 计算 SHA256 哈希
func SHA256Hash(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// ToJSON 将对象转换为 JSON 字符串
func ToJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(data)
}

// FromJSON 从 JSON 字符串解析对象
func FromJSON(data string, v interface{}) error {
	return json.Unmarshal([]byte(data), v)
}

// Retry 重试函数
func Retry(maxAttempts int, delay time.Duration, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}
		lastErr = err
		if attempt < maxAttempts-1 {
			time.Sleep(delay)
		}
	}
	return lastErr
}

// RetryWithBackoff 带退避的重试
func RetryWithBackoff(maxAttempts int, initialDelay time.Duration, maxDelay time.Duration, fn func() error) error {
	var lastErr error
	delay := initialDelay

	for attempt := 0; attempt < maxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}
		lastErr = err

		if attempt < maxAttempts-1 {
			time.Sleep(delay)
			// 指数退避
			delay = time.Duration(float64(delay) * 1.5)
			if delay > maxDelay {
				delay = maxDelay
			}
		}
	}
	return lastErr
}

// Pagination 分页信息
type Pagination struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
	Pages    int64 `json:"pages"`
}

// NewPagination 创建分页信息
func NewPagination(page, pageSize int, total int64) *Pagination {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 1000 {
		pageSize = 1000
	}

	pages := (total + int64(pageSize) - 1) / int64(pageSize)

	return &Pagination{
		Page:     page,
		PageSize: pageSize,
		Total:    total,
		Pages:    pages,
	}
}

// Offset 获取数据库查询偏移量
func (p *Pagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// Limit 获取数据库查询限制
func (p *Pagination) Limit() int {
	return p.PageSize
}

// ErrorWrapper 错误包装器
type ErrorWrapper struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
	Cause   error       `json:"-"`
}

// NewErrorWrapper 创建错误包装器
func NewErrorWrapper(code, message string, cause error) *ErrorWrapper {
	return &ErrorWrapper{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// WithDetails 添加错误详情
func (ew *ErrorWrapper) WithDetails(details interface{}) *ErrorWrapper {
	ew.Details = details
	return ew
}

// Error 实现 error 接口
func (ew *ErrorWrapper) Error() string {
	if ew.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", ew.Code, ew.Message, ew.Cause)
	}
	return fmt.Sprintf("[%s] %s", ew.Code, ew.Message)
}

// RandString 生成随机字符串
func RandString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// RandInt 生成随机整数
func RandInt(min, max int) int {
	return min + rand.Intn(max-min+1)
}

// RandFloat 生成随机浮点数
func RandFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

// TimeNow 获取当前时间（毫秒）
func TimeNow() int64 {
	return time.Now().UnixMilli()
}

// TimeNowSecond 获取当前时间（秒）
func TimeNowSecond() int64 {
	return time.Now().Unix()
}

// FormatTime 格式化时间
func FormatTime(t time.Time, layout string) string {
	if layout == "" {
		layout = "2006-01-02 15:04:05"
	}
	return t.Format(layout)
}

// ParseTime 解析时间
func ParseTime(timeStr string, layout string) (time.Time, error) {
	if layout == "" {
		layout = "2006-01-02 15:04:05"
	}
	return time.Parse(layout, timeStr)
}

// IsNil 检查值是否为 nil
func IsNil(v interface{}) bool {
	return v == nil
}

// IsEmpty 检查字符串是否为空
func IsEmpty(s string) bool {
	return len(s) == 0
}

// IsNotEmpty 检查字符串是否不为空
func IsNotEmpty(s string) bool {
	return len(s) > 0
}

// StringPtr 返回字符串指针
func StringPtr(s string) *string {
	return &s
}

// IntPtr 返回整数指针
func IntPtr(i int) *int {
	return &i
}

// Int64Ptr 返回 int64 指针
func Int64Ptr(i int64) *int64 {
	return &i
}

// Float64Ptr 返回 float64 指针
func Float64Ptr(f float64) *float64 {
	return &f
}

// BoolPtr 返回 bool 指针
func BoolPtr(b bool) *bool {
	return &b
}

// DerefString 解引用字符串指针
func DerefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// DerefInt 解引用整数指针
func DerefInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

// DerefInt64 解引用 int64 指针
func DerefInt64(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}

// DerefFloat64 解引用 float64 指针
func DerefFloat64(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

// DerefBool 解引用 bool 指针
func DerefBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
