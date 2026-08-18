package helper

import (
	"testing"

	"github.com/bestruirui/octopus/internal/model"
)

// TestRunExtractor 验证完整脚本(请求配置 + 提取函数)在成功与失败响应下的行为。
func TestRunExtractor(t *testing.T) {
	script := `({
  request: {
    url: "https://example.com/api/subscription/self",
    method: "GET",
    headers: { "Authorization": "Bearer test" }
  },
  extractor: function (response) {
    if (response.success && response.data) {
      var subs = (response.data.subscriptions || []).filter(function (s) {
        return s && s.subscription && s.subscription.status === "active";
      });
      var total = 0, used = 0;
      subs.forEach(function (s) {
        total += Number(s.subscription.amount_total) || 0;
        used += Number(s.subscription.amount_used) || 0;
      });
      if (subs.length === 0) {
        return { isValid: false, invalidMessage: "无生效中的订阅" };
      }
      return {
        planName: "订阅套餐 x" + subs.length,
        remaining: (total - used) / 500000,
        used: used / 500000,
        total: total / 500000,
        unit: "CNY"
      };
    }
    return { isValid: false, invalidMessage: response.message || "查询失败" };
  }
})`

	successBody := []byte(`{"success":true,"data":{"subscriptions":[{"subscription":{"status":"active","amount_total":100000000,"amount_used":21544797}}]}}`)
	balance, err := runExtractorForTest(script, successBody)
	if err != nil {
		t.Fatalf("success case: %v", err)
	}
	t.Logf("success: total=%v used=%v remaining=%v unit=%v plan=%q",
		balance.Total, balance.Used, balance.Remaining, balance.Unit, balance.PlanName)
	if balance.Total != 200 || balance.Used != 43.089594 {
		t.Fatalf("unexpected values: %+v", balance)
	}

	failBody := []byte(`{"success":false,"message":"Unauthorized"}`)
	_, err = runExtractorForTest(script, failBody)
	if err == nil {
		t.Fatalf("fail case: expected error")
	}
	if err.Error() != "Unauthorized" {
		t.Fatalf("fail case: unexpected error %v", err)
	}
}

// runExtractorForTest 编译脚本后直接用提取函数解析响应体,跳过网络请求。
func runExtractorForTest(scriptText string, responseBody []byte) (*model.Balance, error) {
	channel := &model.Channel{BaseUrls: []model.BaseUrl{{URL: "https://example.com/v1"}}}
	script, err := compileBalanceScript(scriptText, channel)
	if err != nil {
		return nil, err
	}
	return runExtractor(script, responseBody)
}
