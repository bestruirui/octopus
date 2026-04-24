package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lingyuins/octopus/internal/db"
	"github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/op"
)

func TestGetStatsAPIKeyIncludesNamesAndFallbacks(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctx := context.Background()
	testName := strings.NewReplacer("/", "-", "\\", "-", " ", "-").Replace(t.Name())
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", testName)

	if err := db.InitDB("sqlite", dsn, false); err != nil {
		t.Fatalf("init db: %v", err)
	}
	if err := op.InitCache(); err != nil {
		t.Fatalf("init cache: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	apiKey := &model.APIKey{
		Name:    "Primary key",
		APIKey:  "sk-octopus-primary",
		Enabled: true,
	}
	if err := op.APIKeyCreate(apiKey, ctx); err != nil {
		t.Fatalf("create api key: %v", err)
	}

	if err := op.StatsAPIKeyUpdate(apiKey.ID, model.StatsMetrics{
		InputToken:     12,
		RequestSuccess: 3,
	}); err != nil {
		t.Fatalf("update named api key stats: %v", err)
	}
	if err := op.StatsAPIKeyUpdate(999, model.StatsMetrics{
		RequestFailed: 1,
	}); err != nil {
		t.Fatalf("update orphan api key stats: %v", err)
	}

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/stats/apikey", nil)

	getStatsAPIKey(c)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var response struct {
		Code    int                   `json:"code"`
		Message string                `json:"message"`
		Data    []apiKeyStatsResponse `json:"data"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Code != http.StatusOK {
		t.Fatalf("response code = %d, want %d", response.Code, http.StatusOK)
	}

	itemsByID := make(map[int]apiKeyStatsResponse, len(response.Data))
	for _, item := range response.Data {
		itemsByID[item.APIKeyID] = item
	}

	named, ok := itemsByID[apiKey.ID]
	if !ok {
		t.Fatalf("missing named api key stats in response: %+v", response.Data)
	}
	if named.Name != "Primary key" {
		t.Fatalf("named api key name = %q, want %q", named.Name, "Primary key")
	}
	if named.RequestSuccess != 3 || named.InputToken != 12 {
		t.Fatalf("named api key stats = %+v, want request_success=3 and input_token=12", named)
	}

	orphan, ok := itemsByID[999]
	if !ok {
		t.Fatalf("missing orphan api key stats in response: %+v", response.Data)
	}
	if orphan.Name != "Key #999" {
		t.Fatalf("orphan api key fallback name = %q, want %q", orphan.Name, "Key #999")
	}
	if orphan.RequestFailed != 1 {
		t.Fatalf("orphan api key stats = %+v, want request_failed=1", orphan)
	}
}
