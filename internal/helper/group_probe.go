package helper

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	appmodel "github.com/lingyuins/octopus/internal/model"
	transmodel "github.com/lingyuins/octopus/internal/transformer/model"
	"github.com/lingyuins/octopus/internal/transformer/outbound"
)

type GroupModelTestRequest struct {
	GroupID int `json:"group_id" binding:"required"`
}

type GroupModelTestResult struct {
	ItemID       int    `json:"item_id"`
	ChannelID    int    `json:"channel_id"`
	ChannelName  string `json:"channel_name"`
	ModelName    string `json:"model_name"`
	Passed       bool   `json:"passed"`
	Attempts     int    `json:"attempts"`
	StatusCode   int    `json:"status_code"`
	ResponseText string `json:"response_text,omitempty"`
	Message      string `json:"message,omitempty"`
}

type GroupModelTestSummary struct {
	Passed    bool                   `json:"passed"`
	Completed int                    `json:"completed"`
	Total     int                    `json:"total"`
	Results   []GroupModelTestResult `json:"results"`
}

type GroupModelTestProgress struct {
	ID        string                 `json:"id"`
	Passed    bool                   `json:"passed"`
	Completed int                    `json:"completed"`
	Total     int                    `json:"total"`
	Done      bool                   `json:"done"`
	Results   []GroupModelTestResult `json:"results"`
	Message   string                 `json:"message,omitempty"`
}

type groupModelTestProgressEntry struct {
	progress  GroupModelTestProgress
	expiresAt time.Time
}

var groupProbeProgress sync.Map

var groupProbeProgressTTL = 10 * time.Minute

func TestGroupModels(ctx context.Context, group *appmodel.Group, channels map[int]appmodel.Channel) (*GroupModelTestSummary, error) {
	progress := &GroupModelTestProgress{Total: len(group.Items)}
	return runGroupModelTest(ctx, group, channels, progress)
}

func StartGroupModelTest(group *appmodel.Group, channels map[int]appmodel.Channel) (*GroupModelTestProgress, error) {
	if group == nil {
		return nil, fmt.Errorf("group is nil")
	}
	if len(group.Items) == 0 {
		return nil, fmt.Errorf("group has no items")
	}

	id := uuid.NewString()
	progress := &GroupModelTestProgress{
		ID:      id,
		Total:   len(group.Items),
		Results: make([]GroupModelTestResult, 0, len(group.Items)),
	}
	storeGroupModelProgress(progress)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				failed := cloneGroupModelProgress(progress)
				failed.Done = true
				failed.Passed = false
				failed.Message = fmt.Sprintf("internal error: %v", r)
				storeGroupModelProgress(&failed)
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		if _, err := runGroupModelTest(ctx, group, channels, progress); err != nil {
			failed := cloneGroupModelProgress(progress)
			failed.Done = true
			failed.Message = err.Error()
			storeGroupModelProgress(&failed)
		}
	}()

	cloned := cloneGroupModelProgress(progress)
	return &cloned, nil
}

func GetGroupModelTestProgress(id string) (*GroupModelTestProgress, bool) {
	cleanupExpiredGroupModelProgress(time.Now())

	value, ok := groupProbeProgress.Load(id)
	if !ok {
		return nil, false
	}

	entry, ok := value.(groupModelTestProgressEntry)
	if !ok {
		return nil, false
	}

	cloned := cloneGroupModelProgress(&entry.progress)
	return &cloned, true
}

func runGroupModelTest(ctx context.Context, group *appmodel.Group, channels map[int]appmodel.Channel, progress *GroupModelTestProgress) (*GroupModelTestSummary, error) {
	if group == nil {
		return nil, fmt.Errorf("group is nil")
	}
	if len(group.Items) == 0 {
		return nil, fmt.Errorf("group has no items")
	}

	summary := &GroupModelTestSummary{Total: len(group.Items), Results: make([]GroupModelTestResult, 0, len(group.Items))}
	for _, item := range group.Items {
		result := GroupModelTestResult{
			ItemID:    item.ID,
			ChannelID: item.ChannelID,
			ModelName: item.ModelName,
			Attempts:  3,
		}

		channel, ok := channels[item.ChannelID]
		if !ok {
			result.Message = "channel not found"
			appendGroupTestResult(summary, progress, result)
			continue
		}
		result.ChannelName = channel.Name
		if !channel.Enabled {
			result.Message = "channel disabled"
			appendGroupTestResult(summary, progress, result)
			continue
		}

		usedKey := channel.GetChannelKey()
		if strings.TrimSpace(usedKey.ChannelKey) == "" {
			result.Message = "no available key"
			appendGroupTestResult(summary, progress, result)
			continue
		}

		outAdapter := outbound.Get(channel.Type)
		if outAdapter == nil {
			result.Message = fmt.Sprintf("unsupported channel type: %d", channel.Type)
			appendGroupTestResult(summary, progress, result)
			continue
		}

		if err := validateGroupProbeChannelType(group.EndpointType, channel.Type); err != nil {
			result.Message = err.Error()
			appendGroupTestResult(summary, progress, result)
			continue
		}

		for attempt := 1; attempt <= 3; attempt++ {
			if attempt > 1 && ctx.Err() != nil {
				result.Message = ctx.Err().Error()
				break
			}
			statusCode, responseText, err := sendGroupProbeRequest(ctx, outAdapter, &channel, usedKey.ChannelKey, group.EndpointType, item.ModelName)
			result.StatusCode = statusCode
			result.ResponseText = responseText
			if err == nil {
				result.Passed = true
				result.Attempts = attempt
				result.Message = "ok"
				summary.Passed = true
				break
			}
			result.Attempts = attempt
			result.Message = err.Error()
		}

		appendGroupTestResult(summary, progress, result)
	}

	if progress != nil {
		finalProgress := cloneGroupModelProgress(progress)
		finalProgress.Done = true
		finalProgress.Passed = summary.Passed
		storeGroupModelProgress(&finalProgress)
	}

	return summary, nil
}

func appendGroupTestResult(summary *GroupModelTestSummary, progress *GroupModelTestProgress, result GroupModelTestResult) {
	summary.Results = append(summary.Results, result)
	summary.Completed = len(summary.Results)
	if result.Passed {
		summary.Passed = true
	}

	if progress == nil {
		return
	}

	next := cloneGroupModelProgress(progress)
	next.Results = append(next.Results, result)
	next.Completed = len(next.Results)
	next.Passed = summary.Passed
	storeGroupModelProgress(&next)
}

func storeGroupModelProgress(progress *GroupModelTestProgress) {
	storeGroupModelProgressAt(progress, time.Now())
}

func storeGroupModelProgressAt(progress *GroupModelTestProgress, now time.Time) {
	if progress == nil || progress.ID == "" {
		return
	}

	cleanupExpiredGroupModelProgress(now)
	groupProbeProgress.Store(progress.ID, groupModelTestProgressEntry{
		progress:  cloneGroupModelProgress(progress),
		expiresAt: now.Add(groupProbeProgressTTL),
	})
}

func cleanupExpiredGroupModelProgress(now time.Time) {
	groupProbeProgress.Range(func(key, value any) bool {
		entry, ok := value.(groupModelTestProgressEntry)
		if !ok || (!entry.expiresAt.IsZero() && !now.Before(entry.expiresAt)) {
			groupProbeProgress.Delete(key)
		}
		return true
	})
}

func cloneGroupModelProgress(progress *GroupModelTestProgress) GroupModelTestProgress {
	if progress == nil {
		return GroupModelTestProgress{}
	}

	cloned := *progress
	if progress.Results != nil {
		cloned.Results = append([]GroupModelTestResult(nil), progress.Results...)
	}
	return cloned
}

func sendGroupProbeRequest(ctx context.Context, outAdapter transmodel.Outbound, channel *appmodel.Channel, key, endpointType, modelName string) (int, string, error) {
	if channel == nil {
		return 0, "", fmt.Errorf("channel is nil")
	}

	httpClient, err := ChannelHttpClient(channel)
	if err != nil {
		return 0, "", err
	}

	probeRequest, err := buildGroupProbeRequest(endpointType, modelName)
	if err != nil {
		return 0, "", err
	}

	req, err := outAdapter.TransformRequest(ctx, probeRequest, channel.GetBaseUrl(), key)
	if err != nil {
		return 0, "", err
	}

	for _, header := range channel.CustomHeader {
		if strings.TrimSpace(header.HeaderKey) != "" {
			req.Header.Set(header.HeaderKey, header.HeaderValue)
		}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	bodyText := strings.TrimSpace(string(body))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if bodyText == "" {
			bodyText = resp.Status
		}
		return resp.StatusCode, bodyText, fmt.Errorf("upstream error: %d", resp.StatusCode)
	}

	if _, err := outAdapter.TransformResponse(ctx, &http.Response{
		Status:        resp.Status,
		StatusCode:    resp.StatusCode,
		Header:        resp.Header.Clone(),
		Body:          io.NopCloser(strings.NewReader(bodyText)),
		ContentLength: int64(len(bodyText)),
	}); err != nil {
		return resp.StatusCode, bodyText, err
	}

	return resp.StatusCode, bodyText, nil
}

func buildGroupProbeRequest(endpointType, modelName string) (*transmodel.InternalLLMRequest, error) {
	stream := false
	normalizedEndpointType := normalizeGroupProbeEndpointType(endpointType)

	switch {
	case normalizedEndpointType == appmodel.EndpointTypeEmbeddings:
		return &transmodel.InternalLLMRequest{
			Model:          modelName,
			EmbeddingInput: &transmodel.EmbeddingInput{Single: stringPtr("hi")},
		}, nil
	case normalizedEndpointType == appmodel.EndpointTypeAll || appmodel.IsConversationEndpointType(normalizedEndpointType):
		return &transmodel.InternalLLMRequest{
			Model: modelName,
			Messages: []transmodel.Message{{
				Role: "user",
				Content: transmodel.MessageContent{
					Content: stringPtr("hi"),
				},
			}},
			Stream: &stream,
		}, nil
	default:
		return nil, fmt.Errorf("group probe does not support endpoint type: %s", normalizedEndpointType)
	}
}

func validateGroupProbeChannelType(endpointType string, channelType outbound.OutboundType) error {
	normalizedEndpointType := normalizeGroupProbeEndpointType(endpointType)

	switch normalizedEndpointType {
	case appmodel.EndpointTypeEmbeddings:
		if !outbound.IsEmbeddingChannelType(channelType) {
			return fmt.Errorf("channel type %d does not support endpoint type %s", channelType, appmodel.EndpointTypeEmbeddings)
		}
	case appmodel.EndpointTypeAll:
		fallthrough
	default:
		if appmodel.IsConversationEndpointType(normalizedEndpointType) {
			if !outbound.IsChatChannelType(channelType) {
				return fmt.Errorf("channel type %d does not support endpoint type %s", channelType, normalizedEndpointType)
			}
			return nil
		}
		return fmt.Errorf("group probe does not support endpoint type: %s", normalizedEndpointType)
	}

	return nil
}

func normalizeGroupProbeEndpointType(endpointType string) string {
	return appmodel.NormalizeEndpointType(endpointType)
}

func stringPtr(value string) *string {
	return &value
}
