// Package config 提供 TOML 配置加载、环境变量覆盖、配置热更与 schema 校验
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config 基础配置结构
type Config struct {
	// 服务名称
	ServiceName string `mapstructure:"service_name"`
	// 服务版本
	Version string `mapstructure:"version"`
	// 环境：dev, staging, prod
	Environment string `mapstructure:"environment"`
	// HTTP 服务配置
	HTTP HTTPConfig `mapstructure:"http"`
	// gRPC 服务配置
	GRPC GRPCConfig `mapstructure:"grpc"`
	// 数据库配置
	Database DatabaseConfig `mapstructure:"database"`
	// Redis 配置
	Redis RedisConfig `mapstructure:"redis"`
	// Kafka 配置
	Kafka KafkaConfig `mapstructure:"kafka"`
	// 日志配置
	Logger LoggerConfig `mapstructure:"logger"`
	// 追踪配置
	Tracing TracingConfig `mapstructure:"tracing"`
	// 指标配置
	Metrics MetricsConfig `mapstructure:"metrics"`
}

// HTTPConfig HTTP 服务配置
type HTTPConfig struct {
	// 监听地址
	Host string `mapstructure:"host" default:"0.0.0.0"`
	// 监听端口
	Port int `mapstructure:"port" default:"8080"`
	// 读超时（秒）
	ReadTimeout int `mapstructure:"read_timeout" default:"30"`
	// 写超时（秒）
	WriteTimeout int `mapstructure:"write_timeout" default:"30"`
	// 最大连接数
	MaxConnections int `mapstructure:"max_connections" default:"1000"`
}

// GRPCConfig gRPC 服务配置
type GRPCConfig struct {
	// 监听地址
	Host string `mapstructure:"host" default:"0.0.0.0"`
	// 监听端口
	Port int `mapstructure:"port" default:"50051"`
	// 最大并发流数
	MaxConcurrentStreams int `mapstructure:"max_concurrent_streams" default:"1000"`
	// 连接空闲超时（秒）
	IdleTimeout int `mapstructure:"idle_timeout" default:"300"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	// 驱动：mysql, postgres, sqlite
	Driver string `mapstructure:"driver" default:"mysql"`
	// 数据源名称
	DSN string `mapstructure:"dsn"`
	// 最大连接数
	MaxOpenConns int `mapstructure:"max_open_conns" default:"25"`
	// 最大空闲连接数
	MaxIdleConns int `mapstructure:"max_idle_conns" default:"5"`
	// 连接最大生命周期（秒）
	ConnMaxLifetime int `mapstructure:"conn_max_lifetime" default:"300"`
	// 是否启用日志
	LogEnabled bool `mapstructure:"log_enabled" default:"false"`
	// 慢查询阈值（毫秒）
	SlowQueryThreshold int `mapstructure:"slow_query_threshold" default:"1000"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	// 主机地址
	Host string `mapstructure:"host" default:"localhost"`
	// 端口
	Port int `mapstructure:"port" default:"6379"`
	// 密码
	Password string `mapstructure:"password"`
	// 数据库编号
	DB int `mapstructure:"db" default:"0"`
	// 最大连接数
	MaxPoolSize int `mapstructure:"max_pool_size" default:"10"`
	// 连接超时（秒）
	ConnTimeout int `mapstructure:"conn_timeout" default:"5"`
	// 读超时（秒）
	ReadTimeout int `mapstructure:"read_timeout" default:"3"`
	// 写超时（秒）
	WriteTimeout int `mapstructure:"write_timeout" default:"3"`
}

// KafkaConfig Kafka 配置
type KafkaConfig struct {
	// Broker 地址列表
	Brokers []string `mapstructure:"brokers"`
	// Consumer Group ID
	GroupID string `mapstructure:"group_id"`
	// 分区数
	Partitions int `mapstructure:"partitions" default:"3"`
	// 副本数
	Replication int `mapstructure:"replication" default:"1"`
	// 消费者超时（秒）
	SessionTimeout int `mapstructure:"session_timeout" default:"10"`
}

// LoggerConfig 日志配置
type LoggerConfig struct {
	// 日志级别
	Level string `mapstructure:"level" default:"info"`
	// 输出格式
	Format string `mapstructure:"format" default:"json"`
	// 输出目标
	Output string `mapstructure:"output" default:"stdout"`
	// 文件路径
	FilePath string `mapstructure:"file_path" default:"logs/app.log"`
	// 最大文件大小（MB）
	MaxSize int `mapstructure:"max_size" default:"100"`
	// 最大备份文件数
	MaxBackups int `mapstructure:"max_backups" default:"10"`
	// 最大保留天数
	MaxAge int `mapstructure:"max_age" default:"30"`
	// 是否压缩
	Compress bool `mapstructure:"compress" default:"true"`
	// 是否输出调用者信息
	WithCaller bool `mapstructure:"with_caller" default:"true"`
	// 是否输出堆栈跟踪
	WithStacktrace bool `mapstructure:"with_stacktrace" default:"false"`
}

// TracingConfig 追踪配置
type TracingConfig struct {
	// 是否启用
	Enabled bool `mapstructure:"enabled" default:"true"`
	// 追踪器类型：jaeger, otlp
	Type string `mapstructure:"type" default:"otlp"`
	// OTel 收集器端点
	CollectorEndpoint string `mapstructure:"collector_endpoint" default:"localhost:4317"`
	// 采样率
	SamplingRate float64 `mapstructure:"sampling_rate" default:"1.0"`
}

// MetricsConfig 指标配置
type MetricsConfig struct {
	// 是否启用
	Enabled bool `mapstructure:"enabled" default:"true"`
	// Prometheus 监听端口
	Port int `mapstructure:"port" default:"9090"`
	// 指标路径
	Path string `mapstructure:"path" default:"/metrics"`
}

// Load 从 TOML 文件加载配置，支持环境变量覆盖
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// 设置配置文件
	v.SetConfigFile(configPath)
	v.SetConfigType("toml")

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// 设置环境变量前缀
	v.SetEnvPrefix("APP")
	// 自动绑定环境变量（使用 _ 替代 .）
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 解析配置
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// LoadWithDefaults 从 TOML 文件加载配置，使用默认值
func LoadWithDefaults(configPath string) (*Config, error) {
	v := viper.New()

	// 设置默认值
	setDefaults(v)

	// 设置配置文件
	v.SetConfigFile(configPath)
	v.SetConfigType("toml")

	// 读取配置文件（如果不存在则忽略）
	_ = v.ReadInConfig()

	// 设置环境变量前缀
	v.SetEnvPrefix("APP")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 解析配置
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// Validate 验证配置的有效性
func (c *Config) Validate() error {
	if c.ServiceName == "" {
		return fmt.Errorf("service_name is required")
	}
	if c.Environment == "" {
		c.Environment = "dev"
	}
	if c.HTTP.Port <= 0 || c.HTTP.Port > 65535 {
		return fmt.Errorf("invalid HTTP port: %d", c.HTTP.Port)
	}
	if c.GRPC.Port <= 0 || c.GRPC.Port > 65535 {
		return fmt.Errorf("invalid gRPC port: %d", c.GRPC.Port)
	}
	if c.Database.DSN == "" && c.Database.Driver != "sqlite" {
		return fmt.Errorf("database DSN is required for %s driver", c.Database.Driver)
	}
	return nil
}

// setDefaults 设置默认值
func setDefaults(v *viper.Viper) {
	v.SetDefault("http.host", "0.0.0.0")
	v.SetDefault("http.port", 8080)
	v.SetDefault("http.read_timeout", 30)
	v.SetDefault("http.write_timeout", 30)
	v.SetDefault("http.max_connections", 1000)

	v.SetDefault("grpc.host", "0.0.0.0")
	v.SetDefault("grpc.port", 50051)
	v.SetDefault("grpc.max_concurrent_streams", 1000)
	v.SetDefault("grpc.idle_timeout", 300)

	v.SetDefault("database.driver", "mysql")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", 300)
	v.SetDefault("database.log_enabled", false)
	v.SetDefault("database.slow_query_threshold", 1000)

	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.max_pool_size", 10)
	v.SetDefault("redis.conn_timeout", 5)
	v.SetDefault("redis.read_timeout", 3)
	v.SetDefault("redis.write_timeout", 3)

	v.SetDefault("logger.level", "info")
	v.SetDefault("logger.format", "json")
	v.SetDefault("logger.output", "stdout")
	v.SetDefault("logger.file_path", "logs/app.log")
	v.SetDefault("logger.max_size", 100)
	v.SetDefault("logger.max_backups", 10)
	v.SetDefault("logger.max_age", 30)
	v.SetDefault("logger.compress", true)
	v.SetDefault("logger.with_caller", true)
	v.SetDefault("logger.with_stacktrace", false)

	v.SetDefault("tracing.enabled", true)
	v.SetDefault("tracing.type", "otlp")
	v.SetDefault("tracing.collector_endpoint", "localhost:4317")
	v.SetDefault("tracing.sampling_rate", 1.0)

	v.SetDefault("metrics.enabled", true)
	v.SetDefault("metrics.port", 9090)
	v.SetDefault("metrics.path", "/metrics")
}

// GetEnv 获取环境变量，支持默认值
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
