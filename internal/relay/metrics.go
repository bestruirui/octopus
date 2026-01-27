package relay

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/op"
	"github.com/bestruirui/octopus/internal/price"
	transformerModel "github.com/bestruirui/octopus/internal/transformer/model"
	"github.com/bestruirui/octopus/internal/utils/log"
	"github.com/bestruirui/octopus/internal/utils/tokenizer"
)

// RelayMetrics 统一管理请求的日志记录和统计信息
type RelayMetrics struct {
	// 基础信息
	ChannelID      int
	APIKeyID       int
	ChannelName    string // 渠道名称
	RequestModel   string // 请求的模型名称
	ActualModel    string // 实际使用的模型名称
	StartTime      time.Time
	FirstTokenTime time.Time // 首个 Token 时间（流式场景）

	// 请求和响应内容
	InternalRequest  *transformerModel.InternalLLMRequest
	InternalResponse *transformerModel.InternalLLMResponse

	// 统计指标
	Stats model.StatsMetrics

	// 预估成本（用于严格计费）
	EstimatedCost    float64
	CostDeducted     bool
	ActualCostSaved  bool
}

// NewRelayMetrics 创建新的 RelayMetrics
func NewRelayMetrics(requestModel string) *RelayMetrics {
	return &RelayMetrics{
		RequestModel: requestModel,
		StartTime:    time.Now(),
	}
}

func (m *RelayMetrics) SetAPIKeyID(apiKeyID int) {
	m.APIKeyID = apiKeyID
}

// SetChannel 设置通道信息
func (m *RelayMetrics) SetChannel(channelID int, channelName string, actualModel string) {
	m.ChannelID = channelID
	m.ChannelName = channelName
	m.ActualModel = actualModel
}

// EstimateAndDeductCost 在请求发送前估算并扣除成本
// 这确保了即使请求被中断，也会扣除相应的费用
func (m *RelayMetrics) EstimateAndDeductCost(ctx context.Context) {
	if m.CostDeducted {
		return // 已经扣除过，避免重复扣除
	}

	modelPrice := price.GetLLMPrice(m.ActualModel)
	if modelPrice == nil {
		// 没有定价信息，使用默认最小成本
		m.EstimatedCost = 0.0001 // $0.0001 作为最小成本
	} else if modelPrice.Type == "request" {
		// 按请求计费的模型，使用固定成本
		m.EstimatedCost = modelPrice.Request
	} else {
		// 按 token 计费的模型，估算合理的最小成本
		// 使用更合理的估算：假设最少 100 个输入 token 和 50 个输出 token
		// 这样可以减少大部分请求的成本调整幅度
		estimatedInputTokens := 100.0
		estimatedOutputTokens := 50.0
		m.EstimatedCost = (estimatedInputTokens*modelPrice.Input + estimatedOutputTokens*modelPrice.Output) * 1e-6
		
		// 如果估算成本太小，使用最小成本
		if m.EstimatedCost < 0.0001 {
			m.EstimatedCost = 0.0001
		}
	}

	// 立即扣除估算成本
	m.Stats.InputCost = m.EstimatedCost
	m.Stats.OutputCost = 0

	// 更新统计信息（标记为请求开始）
	op.StatsChannelUpdate(m.ChannelID, m.Stats)
	op.StatsTotalUpdate(m.Stats)
	op.StatsHourlyUpdate(m.Stats)
	op.StatsDailyUpdate(ctx, m.Stats)
	op.StatsAPIKeyUpdate(m.APIKeyID, m.Stats)

	m.CostDeducted = true

	log.Debugf("Upfront cost deducted: channel %d, model %s, estimated cost: %f", 
		m.ChannelID, m.ActualModel, m.EstimatedCost)
}

// SetFirstTokenTime 设置首个 Token 时间
func (m *RelayMetrics) SetFirstTokenTime(t time.Time) {
	m.FirstTokenTime = t
}

// SetInternalRequest 设置内部请求
func (m *RelayMetrics) SetInternalRequest(req *transformerModel.InternalLLMRequest) {
	m.InternalRequest = req
}

// SetInternalResponse 设置内部响应并计算费用
func (m *RelayMetrics) SetInternalResponse(resp *transformerModel.InternalLLMResponse) {
	m.InternalResponse = resp

	// 从响应中提取 Usage 并计算费用
	if resp == nil || resp.Usage == nil {
		return
	}

	usage := resp.Usage
	m.Stats.InputToken = usage.PromptTokens
	m.Stats.OutputToken = usage.CompletionTokens

	// 计算实际费用
	modelPrice := price.GetLLMPrice(m.ActualModel)
	if modelPrice == nil {
		return
	}

	var actualInputCost, actualOutputCost float64

	if modelPrice.Type == "request" {
		actualInputCost = modelPrice.Request
		actualOutputCost = 0
	} else {
		if usage.PromptTokensDetails == nil {
			usage.PromptTokensDetails = &transformerModel.PromptTokensDetails{
				CachedTokens: 0,
			}
		}
		if usage.AnthropicUsage {
			actualInputCost = (float64(usage.PromptTokensDetails.CachedTokens)*modelPrice.CacheRead +
				float64(usage.PromptTokens)*modelPrice.Input +
				float64(usage.CacheCreationInputTokens)*modelPrice.CacheWrite) * 1e-6
		} else {
			actualInputCost = (float64(usage.PromptTokensDetails.CachedTokens)*modelPrice.CacheRead + float64(usage.PromptTokens-usage.PromptTokensDetails.CachedTokens)*modelPrice.Input) * 1e-6
		}
		actualOutputCost = float64(usage.CompletionTokens) * modelPrice.Output * 1e-6
	}

	// 如果已经扣除过预估成本，需要调整差额
	if m.CostDeducted {
		// 计算差额（实际成本 - 已扣除的预估成本）
		costDifference := (actualInputCost + actualOutputCost) - m.EstimatedCost
		
		// 更新为实际成本
		m.Stats.InputCost = actualInputCost
		m.Stats.OutputCost = actualOutputCost
		
		// 只更新差额部分到统计
		// 将差额按照实际成本的比例分配到 InputCost 和 OutputCost
		if costDifference != 0 {
			totalActualCost := actualInputCost + actualOutputCost
			var inputDiff, outputDiff float64
			
			if totalActualCost > 0 {
				// 按实际成本比例分配差额
				inputDiff = costDifference * (actualInputCost / totalActualCost)
				outputDiff = costDifference * (actualOutputCost / totalActualCost)
			} else {
				// 如果实际成本为0，将差额全部计入 InputCost（退回预估成本）
				inputDiff = costDifference
				outputDiff = 0
			}
			
			adjustmentMetrics := model.StatsMetrics{
				InputCost:  inputDiff,
				OutputCost: outputDiff,
			}
			op.StatsChannelUpdate(m.ChannelID, adjustmentMetrics)
			op.StatsTotalUpdate(adjustmentMetrics)
			op.StatsHourlyUpdate(adjustmentMetrics)
			op.StatsDailyUpdate(context.Background(), adjustmentMetrics)
			op.StatsAPIKeyUpdate(m.APIKeyID, adjustmentMetrics)
		}
	} else {
		// 如果之前没有扣除预估成本，直接设置实际成本
		m.Stats.InputCost = actualInputCost
		m.Stats.OutputCost = actualOutputCost
	}
}

// Save 保存日志和统计信息
// success: 请求是否成功
// err: 失败时的错误信息，成功时为 nil
func (m *RelayMetrics) Save(ctx context.Context, success bool, err error) {
	duration := time.Since(m.StartTime)

	// Ensure stats are calculated even if usage info was missing
	m.resolveMissingStats()

	// 保存统计信息
	m.saveStats(success, duration)

	// 保存日志
	m.saveLog(ctx, err, duration)
}

func (m *RelayMetrics) resolveMissingStats() {
	// If InputToken is 0, try to calculate it from request
	if m.Stats.InputToken == 0 && m.InternalRequest != nil {
		m.Stats.InputToken = int64(countRequestTokens(m.InternalRequest, m.ActualModel))
	}

	// If OutputToken is 0, try to calculate it from response (if available)
	if m.Stats.OutputToken == 0 && m.InternalResponse != nil {
		m.Stats.OutputToken = int64(countResponseTokens(m.InternalResponse, m.ActualModel))
	}

	// Recalculate cost if needed (if cost is 0 but tokens are > 0)
	modelPrice := price.GetLLMPrice(m.ActualModel)
	if modelPrice == nil {
		return
	}

	if modelPrice.Type == "request" {
		m.Stats.InputCost = modelPrice.Request
		m.Stats.OutputCost = 0
	} else {
		// Simple recalculation based on tokens
		// We don't have detailed usage (cache read/write) here, so assume standard input/output
		if m.Stats.InputCost == 0 && m.Stats.InputToken > 0 {
			m.Stats.InputCost = float64(m.Stats.InputToken) * modelPrice.Input * 1e-6
		}
		if m.Stats.OutputCost == 0 && m.Stats.OutputToken > 0 {
			m.Stats.OutputCost = float64(m.Stats.OutputToken) * modelPrice.Output * 1e-6
		}
	}
}

func countRequestTokens(req *transformerModel.InternalLLMRequest, modelName string) int {
	if req == nil {
		return 0
	}
	text := ""
	if req.EmbeddingInput != nil {
		if req.EmbeddingInput.Single != nil {
			text += *req.EmbeddingInput.Single
		}
		for _, s := range req.EmbeddingInput.Multiple {
			text += s
		}
	}
	for _, msg := range req.Messages {
		if msg.Content.Content != nil {
			text += *msg.Content.Content
		}
		for _, part := range msg.Content.MultipleContent {
			if part.Text != nil {
				text += *part.Text
			}
		}
	}
	return tokenizer.CountTokens(text, modelName)
}

func countResponseTokens(resp *transformerModel.InternalLLMResponse, modelName string) int {
	if resp == nil {
		return 0
	}
	text := ""
	for _, choice := range resp.Choices {
		if choice.Message != nil {
			if choice.Message.Content.Content != nil {
				text += *choice.Message.Content.Content
			}
			for _, part := range choice.Message.Content.MultipleContent {
				if part.Text != nil {
					text += *part.Text
				}
			}
		}
	}
	return tokenizer.CountTokens(text, modelName)
}

// saveStats 保存统计信息
func (m *RelayMetrics) saveStats(success bool, duration time.Duration) {
	// 创建用于更新的指标（不包括成本，因为成本已经在前面扣除了）
	updateMetrics := model.StatsMetrics{
		InputToken:  m.Stats.InputToken,
		OutputToken: m.Stats.OutputToken,
		WaitTime:    duration.Milliseconds(),
	}

	if success {
		updateMetrics.RequestSuccess = 1
	} else {
		updateMetrics.RequestFailed = 1
	}

	op.StatsChannelUpdate(m.ChannelID, updateMetrics)
	op.StatsTotalUpdate(updateMetrics)
	op.StatsHourlyUpdate(updateMetrics)
	op.StatsDailyUpdate(context.Background(), updateMetrics)
	op.StatsAPIKeyUpdate(m.APIKeyID, updateMetrics)

	m.ActualCostSaved = true

	log.Infof("channel: %d, model: %s, success: %t, wait time: %d, input token: %d, output token: %d, input cost: %f, output cost: %f total cost: %f",
		m.ChannelID, m.ActualModel, success, updateMetrics.WaitTime,
		m.Stats.InputToken, m.Stats.OutputToken,
		m.Stats.InputCost, m.Stats.OutputCost, m.Stats.InputCost+m.Stats.OutputCost)
}

// saveLog 保存日志
func (m *RelayMetrics) saveLog(ctx context.Context, err error, duration time.Duration) {
	relayLog := model.RelayLog{
		Time:             m.StartTime.Unix(),
		RequestModelName: m.RequestModel,
		ChannelName:      m.ChannelName,
		ChannelId:        m.ChannelID,
		ActualModelName:  m.ActualModel,
		UseTime:          int(duration.Milliseconds()),
	}

	// 设置首字时间（流式场景）
	if !m.FirstTokenTime.IsZero() {
		relayLog.Ftut = int(m.FirstTokenTime.Sub(m.StartTime).Milliseconds())
	}

	// 设置 Usage 信息
	if m.InternalResponse != nil && m.InternalResponse.Usage != nil {
		relayLog.InputTokens = int(m.InternalResponse.Usage.PromptTokens)
		relayLog.OutputTokens = int(m.InternalResponse.Usage.CompletionTokens)
		relayLog.Cost = m.Stats.InputCost + m.Stats.OutputCost
	}

	// 设置请求内容
	if m.InternalRequest != nil {
		if reqJSON, jsonErr := json.Marshal(m.InternalRequest); jsonErr == nil {
			relayLog.RequestContent = string(reqJSON)
		}
	}

	// 设置响应内容
	if m.InternalResponse != nil {
		// 创建响应的浅拷贝，过滤掉 images 字段以减少存储压力
		respForLog := m.filterResponseForLog(m.InternalResponse)
		if respJSON, jsonErr := json.Marshal(respForLog); jsonErr == nil {
			// 如果是 Anthropic 响应，补充 cache_creation_input_tokens 字段
			if m.InternalResponse.Usage != nil && m.InternalResponse.Usage.AnthropicUsage {
				respStr := string(respJSON)
				old := `"usage":{`
				insert := fmt.Sprintf(`"usage":{"cache_creation_input_tokens":%d,`, m.InternalResponse.Usage.CacheCreationInputTokens)
				respJSON = []byte(strings.Replace(respStr, old, insert, 1))
			}
			relayLog.ResponseContent = string(respJSON)
		}
	}

	// 设置错误信息
	if err != nil {
		relayLog.Error = err.Error()
	}

	if logErr := op.RelayLogAdd(ctx, relayLog); logErr != nil {
		log.Warnf("failed to save relay log: %v", logErr)
	}
}

// filterResponseForLog 创建响应的浅拷贝，过滤掉 images 和 MultipleContent 中的图片数据以减少存储压力
func (m *RelayMetrics) filterResponseForLog(resp *transformerModel.InternalLLMResponse) *transformerModel.InternalLLMResponse {
	if resp == nil {
		return nil
	}

	// 创建浅拷贝
	filtered := *resp
	filtered.Choices = make([]transformerModel.Choice, len(resp.Choices))

	for i, choice := range resp.Choices {
		filtered.Choices[i] = choice

		// 处理 Message
		if choice.Message != nil {
			msgCopy := *choice.Message
			// 清除 Images 字段
			if len(msgCopy.Images) > 0 {
				msgCopy.Images = nil
			}
			// 过滤 MultipleContent 中的图片数据
			if len(msgCopy.Content.MultipleContent) > 0 {
				msgCopy.Content = m.filterMessageContent(msgCopy.Content)
			}
			filtered.Choices[i].Message = &msgCopy
		}

		// 处理 Delta
		if choice.Delta != nil {
			deltaCopy := *choice.Delta
			// 清除 Images 字段
			if len(deltaCopy.Images) > 0 {
				deltaCopy.Images = nil
			}
			// 过滤 MultipleContent 中的图片数据
			if len(deltaCopy.Content.MultipleContent) > 0 {
				deltaCopy.Content = m.filterMessageContent(deltaCopy.Content)
			}
			filtered.Choices[i].Delta = &deltaCopy
		}
	}

	return &filtered
}

// filterMessageContent 过滤 MessageContent 中的图片数据
func (m *RelayMetrics) filterMessageContent(content transformerModel.MessageContent) transformerModel.MessageContent {
	if len(content.MultipleContent) == 0 {
		return content
	}

	filteredParts := make([]transformerModel.MessageContentPart, 0, len(content.MultipleContent))
	for _, part := range content.MultipleContent {
		if part.Type == "image_url" && part.ImageURL != nil {
			// 用占位符替换图片数据
			filteredParts = append(filteredParts, transformerModel.MessageContentPart{
				Type: "image_url",
				ImageURL: &transformerModel.ImageURL{
					URL: "[image data omitted for storage]",
				},
			})
		} else {
			filteredParts = append(filteredParts, part)
		}
	}

	return transformerModel.MessageContent{
		Content:         content.Content,
		MultipleContent: filteredParts,
	}
}
