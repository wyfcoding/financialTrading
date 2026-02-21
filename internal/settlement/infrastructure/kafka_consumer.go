//go:build settlement_experimental
// +build settlement_experimental

package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/wyfcoding/financialtrading/internal/settlement/domain"
)

// KafkaConsumer Kafka消费者
type KafkaConsumer struct {
	reader    *kafka.Reader
	processor *SettlementProcessor
	shutdown  chan struct{}
	wg        sync.WaitGroup
	mu        sync.RWMutex
	config    *KafkaConfig
	metrics   *ConsumerMetrics
}

// KafkaConfig Kafka配置
type KafkaConfig struct {
	Brokers         []string      `json:"brokers"`
	Topic           string        `json:"topic"`
	DLQTopic        string        `json:"dlq_topic"`
	GroupID         string        `json:"group_id"`
	MaxBytes        int           `json:"max_bytes"`
	MinBytes        int           `json:"min_bytes"`
	MaxWait         time.Duration `json:"max_wait"`
	CommitInterval  time.Duration `json:"commit_interval"`
	Concurrency     int           `json:"concurrency"`
	AutoOffsetReset string        `json:"auto_offset_reset"` // earliest, latest
	EnableMetrics   bool          `json:"enable_metrics"`
	EnableDLQ       bool          `json:"enable_dlq"`
}

// ConsumerMetrics 消费者指标
type ConsumerMetrics struct {
	MessagesConsumed    int64         `json:"messages_consumed"`
	MessagesFailed      int64         `json:"messages_failed"`
	LastConsumedAt      time.Time     `json:"last_consumed_at"`
	AvgProcessingTime   time.Duration `json:"avg_processing_time"`
	TotalProcessingTime time.Duration `json:"total_processing_time"`
	mu                  sync.RWMutex
}

// NewKafkaConsumer 创建Kafka消费者
func NewKafkaConsumer(config *KafkaConfig, processor *SettlementProcessor) *KafkaConsumer {
	if config == nil {
		config = &KafkaConfig{
			Brokers:         []string{"localhost:9092"},
			Topic:           "settlement-instructions",
			DLQTopic:        "settlement-instructions-dlq",
			GroupID:         "settlement-group",
			MaxBytes:        10e6, // 10MB
			MinBytes:        10e3, // 10KB
			MaxWait:         1 * time.Second,
			CommitInterval:  5 * time.Second,
			Concurrency:     10,
			AutoOffsetReset: "latest",
			EnableMetrics:   true,
			EnableDLQ:       true,
		}
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        config.Brokers,
		Topic:          config.Topic,
		GroupID:        config.GroupID,
		MaxBytes:       config.MaxBytes,
		MinBytes:       config.MinBytes,
		MaxWait:        config.MaxWait,
		CommitInterval: config.CommitInterval,
		StartOffset:    kafka.LastOffset, // 默认从最新开始
	})

	// 根据配置设置起始偏移量
	if config.AutoOffsetReset == "earliest" {
		reader.SetOffset(kafka.FirstOffset)
	}

	return &KafkaConsumer{
		reader:    reader,
		processor: processor,
		shutdown:  make(chan struct{}),
		config:    config,
		metrics: &ConsumerMetrics{
			LastConsumedAt: time.Now(),
		},
	}
}

// Start 启动消费者
func (kc *KafkaConsumer) Start(ctx context.Context) error {
	kc.mu.Lock()
	defer kc.mu.Unlock()

	// 启动多个消费者协程
	for i := 0; i < kc.config.Concurrency; i++ {
		kc.wg.Add(1)
		go kc.consumeMessages(ctx, i)
	}

	// 启动指标收集
	if kc.config.EnableMetrics {
		go kc.collectMetrics(ctx)
	}

	fmt.Printf("Kafka consumer started with %d workers\n", kc.config.Concurrency)
	return nil
}

// Stop 停止消费者
func (kc *KafkaConsumer) Stop() error {
	kc.mu.Lock()
	defer kc.mu.Unlock()

	// 发送关闭信号
	close(kc.shutdown)

	// 等待所有协程完成
	kc.wg.Wait()

	// 关闭Kafka reader
	err := kc.reader.Close()
	if err != nil {
		return fmt.Errorf("failed to close Kafka reader: %w", err)
	}

	fmt.Println("Kafka consumer stopped")
	return nil
}

// consumeMessages 消费消息
func (kc *KafkaConsumer) consumeMessages(ctx context.Context, workerID int) {
	defer kc.wg.Done()

	fmt.Printf("Consumer worker %d started\n", workerID)

	for {
		select {
		case <-kc.shutdown:
			fmt.Printf("Consumer worker %d shutting down\n", workerID)
			return

		default:
			// 从Kafka读取消息
			msg, err := kc.reader.FetchMessage(ctx)
			if err != nil {
				if err == context.Canceled {
					fmt.Printf("Consumer worker %d context canceled\n", workerID)
					return
				}
				fmt.Printf("Worker %d failed to fetch message: %v\n", workerID, err)
				time.Sleep(1 * time.Second)
				continue
			}

			// 处理消息
			startTime := time.Now()
			err = kc.processMessage(ctx, msg)
			processingTime := time.Since(startTime)

			// 更新指标
			kc.updateMetrics(err, processingTime)

			if err != nil {
				fmt.Printf("Worker %d failed to process message: %v\n", workerID, err)
				// 可以考虑将失败的消息发送到死信队列
				kc.sendToDLQ(ctx, msg, err)
			}

			// 提交偏移量
			err = kc.reader.CommitMessages(ctx, msg)
			if err != nil {
				fmt.Printf("Worker %d failed to commit offset: %v\n", workerID, err)
			}
		}
	}
}

// processMessage 处理消息
func (kc *KafkaConsumer) processMessage(ctx context.Context, msg kafka.Message) error {
	// 解析消息
	var instruction domain.SettlementInstruction
	err := json.Unmarshal(msg.Value, &instruction)
	if err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// 验证消息
	if err := kc.validateMessage(&instruction); err != nil {
		return fmt.Errorf("message validation failed: %w", err)
	}

	// 根据消息类型处理
	switch instruction.SettlementType {
	case domain.SettlementTrade:
		return kc.processTradeInstruction(ctx, &instruction)

	case domain.SettlementFee:
		return kc.processFeeInstruction(ctx, &instruction)

	case domain.SettlementTax:
		return kc.processTaxInstruction(ctx, &instruction)

	case domain.SettlementDividend:
		return kc.processDividendInstruction(ctx, &instruction)

	case domain.SettlementInterest:
		return kc.processInterestInstruction(ctx, &instruction)

	default:
		return fmt.Errorf("unsupported settlement type: %s", instruction.SettlementType)
	}
}

// validateMessage 验证消息
func (kc *KafkaConsumer) validateMessage(instruction *domain.SettlementInstruction) error {
	if instruction.ID == "" {
		return fmt.Errorf("instruction ID is required")
	}

	if instruction.Symbol == "" {
		return fmt.Errorf("symbol is required")
	}

	if instruction.Quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}

	if instruction.Price <= 0 {
		return fmt.Errorf("price must be positive")
	}

	if instruction.Amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}

	return nil
}

// processTradeInstruction 处理交易结算指令
func (kc *KafkaConsumer) processTradeInstruction(ctx context.Context, instruction *domain.SettlementInstruction) error {
	// 创建结算指令
	err := kc.processor.CreateInstruction(ctx, instruction)
	if err != nil {
		return fmt.Errorf("failed to create settlement instruction: %w", err)
	}

	// 添加到批次
	err = kc.processor.AddToBatch(ctx, instruction)
	if err != nil {
		return fmt.Errorf("failed to add instruction to batch: %w", err)
	}

	return nil
}

// processFeeInstruction 处理费用结算指令
func (kc *KafkaConsumer) processFeeInstruction(ctx context.Context, instruction *domain.SettlementInstruction) error {
	// 处理费用结算
	err := kc.processor.ProcessFee(ctx, instruction)
	if err != nil {
		return fmt.Errorf("failed to process fee: %w", err)
	}

	return nil
}

// processTaxInstruction 处理税费结算指令
func (kc *KafkaConsumer) processTaxInstruction(ctx context.Context, instruction *domain.SettlementInstruction) error {
	// 处理税费结算
	err := kc.processor.ProcessTax(ctx, instruction)
	if err != nil {
		return fmt.Errorf("failed to process tax: %w", err)
	}

	return nil
}

// processDividendInstruction 处理股息结算指令
func (kc *KafkaConsumer) processDividendInstruction(ctx context.Context, instruction *domain.SettlementInstruction) error {
	// 处理股息结算
	err := kc.processor.ProcessDividend(ctx, instruction)
	if err != nil {
		return fmt.Errorf("failed to process dividend: %w", err)
	}

	return nil
}

// processInterestInstruction 处理利息结算指令
func (kc *KafkaConsumer) processInterestInstruction(ctx context.Context, instruction *domain.SettlementInstruction) error {
	// 处理利息结算
	err := kc.processor.ProcessInterest(ctx, instruction)
	if err != nil {
		return fmt.Errorf("failed to process interest: %w", err)
	}

	return nil
}

// updateMetrics 更新指标
func (kc *KafkaConsumer) updateMetrics(err error, processingTime time.Duration) {
	kc.metrics.mu.Lock()
	defer kc.metrics.mu.Unlock()

	kc.metrics.MessagesConsumed++
	if err != nil {
		kc.metrics.MessagesFailed++
	}

	kc.metrics.LastConsumedAt = time.Now()
	kc.metrics.TotalProcessingTime += processingTime
	kc.metrics.AvgProcessingTime = kc.metrics.TotalProcessingTime / time.Duration(kc.metrics.MessagesConsumed)
}

// collectMetrics 收集指标
func (kc *KafkaConsumer) collectMetrics(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-kc.shutdown:
			return

		case <-ticker.C:
			kc.logMetrics()
		}
	}
}

// logMetrics 记录指标
func (kc *KafkaConsumer) logMetrics() {
	kc.metrics.mu.RLock()
	defer kc.metrics.mu.RUnlock()

	fmt.Printf("Consumer Metrics - Consumed: %d, Failed: %d, Avg Processing: %v\n",
		kc.metrics.MessagesConsumed,
		kc.metrics.MessagesFailed,
		kc.metrics.AvgProcessingTime)
}

// sendToDLQ 发送到死信队列
func (kc *KafkaConsumer) sendToDLQ(ctx context.Context, msg kafka.Message, err error) {
	if !kc.config.EnableDLQ {
		return
	}
	if kc.config.DLQTopic == "" {
		kc.config.DLQTopic = kc.config.Topic + "-dlq"
	}

	// 创建死信消息
	dlqMsg := &DeadLetterMessage{
		OriginalTopic:     msg.Topic,
		OriginalPartition: msg.Partition,
		OriginalOffset:    msg.Offset,
		OriginalKey:       string(msg.Key),
		OriginalValue:     string(msg.Value),
		Error:             err.Error(),
		Timestamp:         time.Now(),
	}

	// 序列化死信消息
	data, err := json.Marshal(dlqMsg)
	if err != nil {
		fmt.Printf("Failed to marshal DLQ message: %v\n", err)
		return
	}

	writer := &kafka.Writer{
		Addr:     kafka.TCP(kc.config.Brokers...),
		Topic:    kc.config.DLQTopic,
		Balancer: &kafka.LeastBytes{},
	}
	defer writer.Close()

	writeErr := writer.WriteMessages(ctx, kafka.Message{
		Key:   msg.Key,
		Value: data,
		Headers: append(msg.Headers, kafka.Header{
			Key:   "x-settlement-dlq-error",
			Value: []byte(dlqMsg.Error),
		}),
		Time: time.Now(),
	})
	if writeErr != nil {
		fmt.Printf("Failed to write DLQ message: topic=%s err=%v\n", kc.config.DLQTopic, writeErr)
		return
	}

	fmt.Printf("Message sent to DLQ topic=%s offset=%d\n", kc.config.DLQTopic, msg.Offset)
}

// GetMetrics 获取指标
func (kc *KafkaConsumer) GetMetrics() *ConsumerMetrics {
	kc.metrics.mu.RLock()
	defer kc.metrics.mu.RUnlock()

	// 返回副本
	return &ConsumerMetrics{
		MessagesConsumed:    kc.metrics.MessagesConsumed,
		MessagesFailed:      kc.metrics.MessagesFailed,
		LastConsumedAt:      kc.metrics.LastConsumedAt,
		AvgProcessingTime:   kc.metrics.AvgProcessingTime,
		TotalProcessingTime: kc.metrics.TotalProcessingTime,
	}
}

// SettlementProcessor 结算处理器
type SettlementProcessor struct {
	instructionRepo domain.SettlementInstructionRepository
	batchRepo       domain.SettlementBatchRepository
	accountRepo     domain.AccountRepository
	paymentGateway  domain.PaymentGateway
	mu              sync.RWMutex
}

// NewSettlementProcessor 创建结算处理器
func NewSettlementProcessor(instructionRepo domain.SettlementInstructionRepository,
	batchRepo domain.SettlementBatchRepository,
	accountRepo domain.AccountRepository,
	paymentGateway domain.PaymentGateway) *SettlementProcessor {

	return &SettlementProcessor{
		instructionRepo: instructionRepo,
		batchRepo:       batchRepo,
		accountRepo:     accountRepo,
		paymentGateway:  paymentGateway,
	}
}

// CreateInstruction 创建结算指令
func (sp *SettlementProcessor) CreateInstruction(ctx context.Context, instruction *domain.SettlementInstruction) error {
	// 保存指令
	err := sp.instructionRepo.SaveInstruction(ctx, instruction)
	if err != nil {
		return fmt.Errorf("failed to save instruction: %w", err)
	}

	return nil
}

// AddToBatch 添加到批次
func (sp *SettlementProcessor) AddToBatch(ctx context.Context, instruction *domain.SettlementInstruction) error {
	// 查找或创建批次
	batch, err := sp.findOrCreateBatch(ctx, instruction.SettlementDate, domain.CycleT1)
	if err != nil {
		return fmt.Errorf("failed to find or create batch: %w", err)
	}

	// 更新指令批次ID
	instruction.BatchID = batch.ID

	// 更新指令
	err = sp.instructionRepo.UpdateInstruction(ctx, instruction)
	if err != nil {
		return fmt.Errorf("failed to update instruction: %w", err)
	}

	// 更新批次统计
	batch.TotalInstructions++
	batch.TotalAmount += instruction.NetAmount
	batch.PendingCount++
	batch.UpdatedAt = time.Now()

	err = sp.batchRepo.UpdateBatch(ctx, batch)
	if err != nil {
		return fmt.Errorf("failed to update batch: %w", err)
	}

	return nil
}

// findOrCreateBatch 查找或创建批次
func (sp *SettlementProcessor) findOrCreateBatch(ctx context.Context, settlementDate time.Time, cycle domain.SettlementCycle) (*domain.SettlementBatch, error) {
	// 查找现有批次
	batch, err := sp.batchRepo.GetBatchByDate(ctx, settlementDate, cycle)
	if err != nil {
		return nil, fmt.Errorf("failed to get batch: %w", err)
	}

	if batch != nil {
		return batch, nil
	}

	// 创建新批次
	batch = &domain.SettlementBatch{
		ID:                generateBatchID(),
		BatchNo:           generateBatchNo(),
		SettlementDate:    settlementDate,
		Cycle:             cycle,
		Status:            "PENDING",
		TotalInstructions: 0,
		TotalAmount:       0,
		SuccessCount:      0,
		FailedCount:       0,
		PendingCount:      0,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	err = sp.batchRepo.SaveBatch(ctx, batch)
	if err != nil {
		return nil, fmt.Errorf("failed to save batch: %w", err)
	}

	return batch, nil
}

// ProcessFee 处理费用
func (sp *SettlementProcessor) ProcessFee(ctx context.Context, instruction *domain.SettlementInstruction) error {
	instruction.SettlementType = domain.SettlementFee
	instruction.Status = domain.SettlementPending
	if instruction.NetAmount <= 0 {
		instruction.NetAmount = instruction.Amount
	}
	if len(instruction.Fees) == 0 {
		instruction.Fees = []*domain.FeeDetail{
			{
				FeeType:     "PROCESSING_FEE",
				FeeAmount:   instruction.Amount,
				FeeRate:     0,
				Currency:    instruction.Currency,
				Description: "Fee settlement entry",
			},
		}
	}
	instruction.UpdatedAt = time.Now()

	if err := sp.CreateInstruction(ctx, instruction); err != nil {
		return err
	}
	if err := sp.AddToBatch(ctx, instruction); err != nil {
		return err
	}
	return nil
}

// ProcessTax 处理税费
func (sp *SettlementProcessor) ProcessTax(ctx context.Context, instruction *domain.SettlementInstruction) error {
	instruction.SettlementType = domain.SettlementTax
	instruction.Status = domain.SettlementPending
	if instruction.NetAmount <= 0 {
		instruction.NetAmount = instruction.Amount
	}
	if len(instruction.Taxes) == 0 {
		instruction.Taxes = []*domain.TaxDetail{
			{
				TaxType:     "WITHHOLDING_TAX",
				TaxAmount:   instruction.Amount,
				TaxRate:     0,
				Currency:    instruction.Currency,
				Description: "Tax settlement entry",
			},
		}
	}
	instruction.UpdatedAt = time.Now()

	if err := sp.CreateInstruction(ctx, instruction); err != nil {
		return err
	}
	if err := sp.AddToBatch(ctx, instruction); err != nil {
		return err
	}
	return nil
}

// ProcessDividend 处理股息
func (sp *SettlementProcessor) ProcessDividend(ctx context.Context, instruction *domain.SettlementInstruction) error {
	instruction.SettlementType = domain.SettlementDividend
	instruction.Status = domain.SettlementPending
	if instruction.NetAmount <= 0 {
		instruction.NetAmount = instruction.Amount
	}
	instruction.UpdatedAt = time.Now()

	if err := sp.CreateInstruction(ctx, instruction); err != nil {
		return err
	}
	if err := sp.AddToBatch(ctx, instruction); err != nil {
		return err
	}
	return nil
}

// ProcessInterest 处理利息
func (sp *SettlementProcessor) ProcessInterest(ctx context.Context, instruction *domain.SettlementInstruction) error {
	instruction.SettlementType = domain.SettlementInterest
	instruction.Status = domain.SettlementPending
	if instruction.NetAmount <= 0 {
		instruction.NetAmount = instruction.Amount
	}
	instruction.UpdatedAt = time.Now()

	if err := sp.CreateInstruction(ctx, instruction); err != nil {
		return err
	}
	if err := sp.AddToBatch(ctx, instruction); err != nil {
		return err
	}
	return nil
}

// Data structures

type DeadLetterMessage struct {
	OriginalTopic     string    `json:"original_topic"`
	OriginalPartition int       `json:"original_partition"`
	OriginalOffset    int64     `json:"original_offset"`
	OriginalKey       string    `json:"original_key"`
	OriginalValue     string    `json:"original_value"`
	Error             string    `json:"error"`
	Timestamp         time.Time `json:"timestamp"`
}

// Helper functions

func generateBatchID() string {
	return fmt.Sprintf("BATCH_%d", time.Now().UnixNano())
}

func generateBatchNo() string {
	return fmt.Sprintf("BATCH%d", time.Now().UnixNano())
}
