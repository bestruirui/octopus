package op

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/lingyuins/octopus/internal/db"
	"github.com/lingyuins/octopus/internal/model"
)

func TestNormalizeAIRouteEntriesMergesSameRequestedModel(t *testing.T) {
	routes := []model.AIRouteEntry{
		{
			EndpointType:   model.EndpointTypeAll,
			RequestedModel: " gpt-4o ",
			Items: []model.AIRouteItemSpec{
				{ChannelID: 1, UpstreamModel: "gpt-4o", Priority: 1, Weight: 100},
				{ChannelID: 1, UpstreamModel: "gpt-4o", Priority: 2, Weight: 50},
			},
		},
		{
			EndpointType:   model.EndpointTypeAll,
			RequestedModel: "GPT-4O",
			Items: []model.AIRouteItemSpec{
				{ChannelID: 2, UpstreamModel: "chatgpt-4o-latest", Priority: 1, Weight: 80},
				{ChannelID: 0, UpstreamModel: "ignored", Priority: 1, Weight: 100},
			},
		},
		{
			RequestedModel: "   ",
			Items: []model.AIRouteItemSpec{
				{ChannelID: 3, UpstreamModel: "should-be-skipped", Priority: 1, Weight: 100},
			},
		},
	}

	got := normalizeAIRouteEntries(routes)
	if len(got) != 1 {
		t.Fatalf("normalizeAIRouteEntries() len = %d, want 1", len(got))
	}

	if got[0].RequestedModel != "gpt-4o" {
		t.Fatalf("normalizeAIRouteEntries()[0].RequestedModel = %q, want %q", got[0].RequestedModel, "gpt-4o")
	}

	if len(got[0].Items) != 2 {
		t.Fatalf("normalizeAIRouteEntries()[0].Items len = %d, want 2", len(got[0].Items))
	}

	if got[0].Items[0].ChannelID != 1 || got[0].Items[0].UpstreamModel != "gpt-4o" {
		t.Fatalf("normalizeAIRouteEntries()[0].Items[0] = %+v, want channel 1 / gpt-4o", got[0].Items[0])
	}

	if got[0].Items[1].ChannelID != 2 || got[0].Items[1].UpstreamModel != "chatgpt-4o-latest" {
		t.Fatalf("normalizeAIRouteEntries()[0].Items[1] = %+v, want channel 2 / chatgpt-4o-latest", got[0].Items[1])
	}
}

func TestDedupeAIRouteItemsPreservesFirstOccurrence(t *testing.T) {
	items := []model.AIRouteItemSpec{
		{ChannelID: 1, UpstreamModel: " model-a ", Priority: 5, Weight: 20},
		{ChannelID: 1, UpstreamModel: "MODEL-A", Priority: 1, Weight: 100},
		{ChannelID: 2, UpstreamModel: "model-a", Priority: 1, Weight: 100},
		{ChannelID: 3, UpstreamModel: "", Priority: 1, Weight: 100},
	}

	got := dedupeAIRouteItems(items)
	if len(got) != 2 {
		t.Fatalf("dedupeAIRouteItems() len = %d, want 2", len(got))
	}

	if got[0].ChannelID != 1 || got[0].UpstreamModel != "model-a" || got[0].Priority != 5 || got[0].Weight != 20 {
		t.Fatalf("dedupeAIRouteItems()[0] = %+v, want first trimmed occurrence", got[0])
	}

	if got[1].ChannelID != 2 || got[1].UpstreamModel != "model-a" {
		t.Fatalf("dedupeAIRouteItems()[1] = %+v, want channel 2 / model-a", got[1])
	}
}

func TestNormalizeAIRouteEntriesKeepsDifferentEndpointTypesSeparated(t *testing.T) {
	routes := []model.AIRouteEntry{
		{
			EndpointType:   model.EndpointTypeAll,
			RequestedModel: "common-model",
			Items: []model.AIRouteItemSpec{
				{ChannelID: 1, UpstreamModel: "gpt-4o"},
			},
		},
		{
			EndpointType:   model.EndpointTypeEmbeddings,
			RequestedModel: "common-model",
			Items: []model.AIRouteItemSpec{
				{ChannelID: 2, UpstreamModel: "text-embedding-3-large"},
			},
		},
	}

	got := normalizeAIRouteEntries(routes)
	if len(got) != 2 {
		t.Fatalf("normalizeAIRouteEntries() len = %d, want 2", len(got))
	}

	if got[0].EndpointType == got[1].EndpointType {
		t.Fatalf("normalizeAIRouteEntries() endpoint types should stay separated, got %+v", got)
	}
}

func TestAutoCorrectAIRouteTableRoutesRenamesConflictingEndpointTypes(t *testing.T) {
	routes := []model.AIRouteEntry{
		{
			EndpointType:   model.EndpointTypeAll,
			RequestedModel: "deepseek",
			Items: []model.AIRouteItemSpec{
				{ChannelID: 1, UpstreamModel: "deepseek-chat"},
			},
		},
		{
			EndpointType:   model.EndpointTypeEmbeddings,
			RequestedModel: "deepseek",
			Items: []model.AIRouteItemSpec{
				{ChannelID: 2, UpstreamModel: "deepseek-embedding"},
			},
		},
	}

	got, corrections := autoCorrectAIRouteTableRoutes(routes, nil)
	if len(corrections) != 1 {
		t.Fatalf("autoCorrectAIRouteTableRoutes() corrections len = %d, want 1", len(corrections))
	}

	if got[0].RequestedModel != "deepseek" {
		t.Fatalf("autoCorrectAIRouteTableRoutes()[0].RequestedModel = %q, want %q", got[0].RequestedModel, "deepseek")
	}
	if got[0].MatchRegex != "" {
		t.Fatalf("autoCorrectAIRouteTableRoutes()[0].MatchRegex = %q, want empty", got[0].MatchRegex)
	}

	if got[1].RequestedModel != "deepseek (embeddings)" {
		t.Fatalf("autoCorrectAIRouteTableRoutes()[1].RequestedModel = %q, want %q", got[1].RequestedModel, "deepseek (embeddings)")
	}
	if got[1].MatchRegex != `(?i)^deepseek$` {
		t.Fatalf("autoCorrectAIRouteTableRoutes()[1].MatchRegex = %q, want %q", got[1].MatchRegex, `(?i)^deepseek$`)
	}
}

func TestAutoCorrectAIRouteTableRoutesKeepsExistingCompatibleGroupName(t *testing.T) {
	routes := []model.AIRouteEntry{
		{
			EndpointType:   model.EndpointTypeAll,
			RequestedModel: "deepseek",
			Items: []model.AIRouteItemSpec{
				{ChannelID: 1, UpstreamModel: "deepseek-chat"},
			},
		},
		{
			EndpointType:   model.EndpointTypeEmbeddings,
			RequestedModel: "deepseek",
			Items: []model.AIRouteItemSpec{
				{ChannelID: 2, UpstreamModel: "deepseek-embedding"},
			},
		},
	}
	existing := []model.Group{
		{
			Name:         "deepseek",
			EndpointType: model.EndpointTypeEmbeddings,
		},
	}

	got, corrections := autoCorrectAIRouteTableRoutes(routes, existing)
	if len(corrections) != 1 {
		t.Fatalf("autoCorrectAIRouteTableRoutes() corrections len = %d, want 1", len(corrections))
	}

	if got[0].RequestedModel != "deepseek (chat)" {
		t.Fatalf("autoCorrectAIRouteTableRoutes()[0].RequestedModel = %q, want %q", got[0].RequestedModel, "deepseek (chat)")
	}
	if got[0].MatchRegex != `(?i)^deepseek$` {
		t.Fatalf("autoCorrectAIRouteTableRoutes()[0].MatchRegex = %q, want %q", got[0].MatchRegex, `(?i)^deepseek$`)
	}

	if got[1].RequestedModel != "deepseek" {
		t.Fatalf("autoCorrectAIRouteTableRoutes()[1].RequestedModel = %q, want %q", got[1].RequestedModel, "deepseek")
	}
	if got[1].MatchRegex != "" {
		t.Fatalf("autoCorrectAIRouteTableRoutes()[1].MatchRegex = %q, want empty", got[1].MatchRegex)
	}
}

func TestAutoCorrectAIRouteTableRoutesAvoidsReservedNames(t *testing.T) {
	routes := []model.AIRouteEntry{
		{
			EndpointType:   model.EndpointTypeAll,
			RequestedModel: "deepseek",
			Items: []model.AIRouteItemSpec{
				{ChannelID: 1, UpstreamModel: "deepseek-chat"},
			},
		},
		{
			EndpointType:   model.EndpointTypeEmbeddings,
			RequestedModel: "deepseek",
			Items: []model.AIRouteItemSpec{
				{ChannelID: 2, UpstreamModel: "deepseek-embedding"},
			},
		},
	}
	existing := []model.Group{
		{
			Name:         "deepseek (embeddings)",
			EndpointType: model.EndpointTypeEmbeddings,
		},
	}

	got, corrections := autoCorrectAIRouteTableRoutes(routes, existing)
	if len(corrections) != 1 {
		t.Fatalf("autoCorrectAIRouteTableRoutes() corrections len = %d, want 1", len(corrections))
	}

	if got[1].RequestedModel != "deepseek (embeddings) 2" {
		t.Fatalf("autoCorrectAIRouteTableRoutes()[1].RequestedModel = %q, want %q", got[1].RequestedModel, "deepseek (embeddings) 2")
	}
}

func TestBuildAIRoutePromptBucketsSplitsCapabilities(t *testing.T) {
	inputs := []model.AIRouteModelInput{
		{ChannelID: 1, ChannelName: "chat", Provider: "openai", Model: "gpt-4o"},
		{ChannelID: 1, ChannelName: "chat", Provider: "openai", Model: "gpt-4o"},
		{ChannelID: 2, ChannelName: "embed", Provider: "openai", Model: "text-embedding-3-large"},
		{ChannelID: 3, ChannelName: "image", Provider: "openai", Model: "gpt-image-1"},
		{ChannelID: 4, ChannelName: "chat2", Provider: "anthropic", Model: "claude-sonnet-4.5"},
	}

	got := buildAIRoutePromptBuckets(inputs, "")
	if len(got) != 3 {
		t.Fatalf("buildAIRoutePromptBuckets() len = %d, want 3", len(got))
	}

	if got[0].PromptEndpointType != model.EndpointTypeChat || got[0].GroupEndpointType != model.EndpointTypeAll {
		t.Fatalf("buildAIRoutePromptBuckets()[0] = %+v, want chat bucket mapped to *", got[0])
	}
	if len(got[0].ModelInputs) != 2 {
		t.Fatalf("chat bucket len = %d, want 2", len(got[0].ModelInputs))
	}
	if got[1].PromptEndpointType != model.EndpointTypeEmbeddings || got[1].GroupEndpointType != model.EndpointTypeEmbeddings {
		t.Fatalf("buildAIRoutePromptBuckets()[1] = %+v, want embeddings bucket", got[1])
	}
	if len(got[1].ModelInputs) != 1 || got[1].ModelInputs[0].Model != "text-embedding-3-large" {
		t.Fatalf("embeddings bucket = %+v, want text-embedding-3-large", got[1].ModelInputs)
	}
	if got[2].PromptEndpointType != model.EndpointTypeImageGeneration || got[2].GroupEndpointType != model.EndpointTypeImageGeneration {
		t.Fatalf("buildAIRoutePromptBuckets()[2] = %+v, want image bucket", got[2])
	}
}

func TestSummarizeAIRouteErrorBodyCollapsesHTML(t *testing.T) {
	got := summarizeAIRouteErrorBody("<html><body>504 Gateway Time-out</body></html>")
	if got != "upstream returned an HTML error page" {
		t.Fatalf("summarizeAIRouteErrorBody() = %q, want %q", got, "upstream returned an HTML error page")
	}
}

func TestDetectAIRoutePromptEndpointTypeForGroupPrefersNonChatWhenStable(t *testing.T) {
	group := model.Group{
		EndpointType: model.EndpointTypeAll,
		Items: []model.GroupItem{
			{ModelName: "text-embedding-3-large"},
			{ModelName: "text-embedding-3-small"},
		},
	}

	got := detectAIRoutePromptEndpointTypeForGroup(group)
	if got != model.EndpointTypeEmbeddings {
		t.Fatalf("detectAIRoutePromptEndpointTypeForGroup() = %q, want %q", got, model.EndpointTypeEmbeddings)
	}
}

func TestDetectAIRoutePromptEndpointTypeForGroup_PreservesDeepSeekEndpointType(t *testing.T) {
	group := model.Group{
		EndpointType: model.EndpointTypeDeepSeek,
		Items: []model.GroupItem{
			{ModelName: "deepseek-chat"},
		},
	}

	got := detectAIRoutePromptEndpointTypeForGroup(group)
	if got != model.EndpointTypeDeepSeek {
		t.Fatalf("detectAIRoutePromptEndpointTypeForGroup() = %q, want %q", got, model.EndpointTypeDeepSeek)
	}
}

func TestBuildAIRoutePromptBucketsSplitsLargeBucket(t *testing.T) {
	inputs := make([]model.AIRouteModelInput, 0, aiRouteMaxModelsPerRequest+5)
	for i := 0; i < aiRouteMaxModelsPerRequest+5; i++ {
		inputs = append(inputs, model.AIRouteModelInput{
			ChannelID:   i + 1,
			ChannelName: "chat",
			Provider:    "openai",
			Model:       fmt.Sprintf("gpt-4o-variant-%03d", i),
		})
	}

	got := buildAIRoutePromptBuckets(inputs, model.EndpointTypeChat)
	if len(got) != 2 {
		t.Fatalf("buildAIRoutePromptBuckets() len = %d, want 2", len(got))
	}

	total := 0
	for _, bucket := range got {
		if len(bucket.ModelInputs) > aiRouteMaxModelsPerRequest {
			t.Fatalf("bucket len = %d, want <= %d", len(bucket.ModelInputs), aiRouteMaxModelsPerRequest)
		}
		total += len(bucket.ModelInputs)
	}

	if total != len(inputs) {
		t.Fatalf("buildAIRoutePromptBuckets() total = %d, want %d", total, len(inputs))
	}
}

func TestBuildAIRoutePromptBuckets_PreservesDeepSeekGroupEndpointTypeForChatPrompts(t *testing.T) {
	inputs := []model.AIRouteModelInput{
		{ChannelID: 1, ChannelName: "chat", Provider: "openai", Model: "deepseek-chat"},
	}

	got := buildAIRoutePromptBuckets(inputs, model.EndpointTypeDeepSeek)
	if len(got) != 1 {
		t.Fatalf("buildAIRoutePromptBuckets() len = %d, want 1", len(got))
	}
	if got[0].PromptEndpointType != model.EndpointTypeChat {
		t.Fatalf("prompt endpoint type = %q, want %q", got[0].PromptEndpointType, model.EndpointTypeChat)
	}
	if got[0].GroupEndpointType != model.EndpointTypeDeepSeek {
		t.Fatalf("group endpoint type = %q, want %q", got[0].GroupEndpointType, model.EndpointTypeDeepSeek)
	}
}

func TestFormatAIRouteTimeout(t *testing.T) {
	got := formatAIRouteTimeout(3 * time.Minute)
	if got != "180s" {
		t.Fatalf("formatAIRouteTimeout() = %q, want %q", got, "180s")
	}
}

func TestIsAIRouteTimeoutErrorDetectsContextDeadlineExceeded(t *testing.T) {
	if !isAIRouteTimeoutError(context.DeadlineExceeded) {
		t.Fatal("isAIRouteTimeoutError() = false, want true")
	}
}

func TestSyncGroupItemsWithAIRouteReplacesStaleItemsAndUpdatesExisting(t *testing.T) {
	ctx := context.Background()
	if err := db.InitDB("sqlite", "file:ai-route-sync-group-items?mode=memory&cache=shared", false); err != nil {
		t.Fatalf("init db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		groupCache.Clear()
		groupMap.Clear()
		rebuildGroupIndexesFromCache()
	})

	groupCache.Clear()
	groupMap.Clear()
	rebuildGroupIndexesFromCache()

	group := &model.Group{
		Name:         "sync-target",
		EndpointType: model.EndpointTypeAll,
		Mode:         model.GroupModeRoundRobin,
		Items: []model.GroupItem{
			{ChannelID: 1, ModelName: "legacy-a", Priority: 1, Weight: 10},
			{ChannelID: 2, ModelName: "keep-b", Priority: 2, Weight: 20},
		},
	}
	if err := GroupCreate(group, ctx); err != nil {
		t.Fatalf("create group: %v", err)
	}

	writtenCount, err := syncGroupItemsWithAIRoute(ctx, group.ID, model.EndpointTypeChat, []model.GroupItem{
		{ChannelID: 2, ModelName: "keep-b", Priority: 1, Weight: 99},
		{ChannelID: 3, ModelName: " new-c ", Priority: 2, Weight: 50},
	})
	if err != nil {
		t.Fatalf("syncGroupItemsWithAIRoute() error = %v", err)
	}
	if writtenCount != 2 {
		t.Fatalf("syncGroupItemsWithAIRoute() writtenCount = %d, want 2", writtenCount)
	}

	updatedGroup, err := GroupGet(group.ID, ctx)
	if err != nil {
		t.Fatalf("get group after sync: %v", err)
	}
	if updatedGroup.EndpointType != model.EndpointTypeChat {
		t.Fatalf("group endpoint_type = %q, want %q", updatedGroup.EndpointType, model.EndpointTypeChat)
	}

	items, err := GroupItemList(group.ID, ctx)
	if err != nil {
		t.Fatalf("list group items after sync: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 persisted items after sync, got %d", len(items))
	}

	if items[0].ChannelID != 2 || items[0].ModelName != "keep-b" || items[0].Priority != 1 || items[0].Weight != 99 {
		t.Fatalf("unexpected first item after sync: %+v", items[0])
	}
	if items[1].ChannelID != 3 || items[1].ModelName != "new-c" || items[1].Priority != 2 || items[1].Weight != 50 {
		t.Fatalf("unexpected second item after sync: %+v", items[1])
	}
}

func TestSelectAIRouteForGroupRejectsMismatchedSingleRoute(t *testing.T) {
	group := model.Group{Name: "gpt-4o"}
	routes := []model.AIRouteEntry{
		{
			RequestedModel: "gpt-4.1",
			Items: []model.AIRouteItemSpec{
				{ChannelID: 1, UpstreamModel: "gpt-4.1"},
			},
		},
	}

	_, err := selectAIRouteForGroup(group, routes)
	if err == nil {
		t.Fatal("selectAIRouteForGroup() error = nil, want non-nil")
	}
}

func TestValidateAIRouteTableRoutesRejectsSameNameDifferentEndpointTypes(t *testing.T) {
	routes := []model.AIRouteEntry{
		{
			EndpointType:   model.EndpointTypeAll,
			RequestedModel: "shared-model",
			Items: []model.AIRouteItemSpec{
				{ChannelID: 1, UpstreamModel: "gpt-4o"},
			},
		},
		{
			EndpointType:   model.EndpointTypeEmbeddings,
			RequestedModel: "shared-model",
			Items: []model.AIRouteItemSpec{
				{ChannelID: 2, UpstreamModel: "text-embedding-3-large"},
			},
		},
	}

	err := validateAIRouteTableRoutes(routes)
	if err == nil {
		t.Fatal("validateAIRouteTableRoutes() error = nil, want non-nil")
	}
}

func TestValidateAIRouteTableRoutesAllowsUniqueNames(t *testing.T) {
	routes := []model.AIRouteEntry{
		{
			EndpointType:   model.EndpointTypeAll,
			RequestedModel: "shared-model",
			Items: []model.AIRouteItemSpec{
				{ChannelID: 1, UpstreamModel: "gpt-4o"},
			},
		},
		{
			EndpointType:   model.EndpointTypeAll,
			RequestedModel: "shared-model-2",
			Items: []model.AIRouteItemSpec{
				{ChannelID: 2, UpstreamModel: "gpt-4.1"},
			},
		},
	}

	if err := validateAIRouteTableRoutes(routes); err != nil {
		t.Fatalf("validateAIRouteTableRoutes() error = %v, want nil", err)
	}
}

func TestSummarizeAIRouteBucketFailures(t *testing.T) {
	err := summarizeAIRouteBucketFailures([]aiRouteBucketFailure{
		{Index: 2, Total: 3, Err: errors.New("service unavailable")},
		{Index: 1, Total: 3, Err: errors.New("timeout")},
	})
	if err == nil {
		t.Fatal("summarizeAIRouteBucketFailures() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "2 个 AI 分析批次失败") {
		t.Fatalf("summarizeAIRouteBucketFailures() = %q, want failure count", err.Error())
	}
	if !strings.Contains(err.Error(), "第 1/3 批") {
		t.Fatalf("summarizeAIRouteBucketFailures() = %q, want first batch details", err.Error())
	}
}

func TestNewAIRouteTablePartialFailureError(t *testing.T) {
	cause := errors.New("第 1/2 批 AI 分析失败：timeout")
	err := newAIRouteTablePartialFailureError(3, cause)
	if err == nil {
		t.Fatal("newAIRouteTablePartialFailureError() error = nil, want non-nil")
	}
	if !errors.Is(err, cause) {
		t.Fatal("newAIRouteTablePartialFailureError() should wrap original cause")
	}
	if !strings.Contains(err.Error(), "已保留成功写入的 3 个分组") {
		t.Fatalf("newAIRouteTablePartialFailureError() = %q, want preserved groups message", err.Error())
	}
}
