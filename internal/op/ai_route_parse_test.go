package op

import (
	"testing"

	"github.com/lingyuins/octopus/internal/model"
)

func TestParseAIRouteResponseContentAcceptsWrappedJSON(t *testing.T) {
	content := "先说明一下，示例占位符 {ignore_me} 不属于结果。\n```json\n{\"routes\":[{\"requested_model\":\"gpt-4o\",\"items\":[{\"channel_id\":1,\"upstream_model\":\"gpt-4o\",\"priority\":1,\"weight\":100}]}]}\n```"

	resp, err := parseAIRouteResponseContent(content)
	if err != nil {
		t.Fatalf("parseAIRouteResponseContent(wrapped) error = %v, want nil", err)
	}
	if len(resp.Routes) != 1 {
		t.Fatalf("parseAIRouteResponseContent(wrapped) routes len = %d, want 1", len(resp.Routes))
	}
	if resp.Routes[0].RequestedModel != "gpt-4o" {
		t.Fatalf("parseAIRouteResponseContent(wrapped) requested_model = %q, want %q", resp.Routes[0].RequestedModel, "gpt-4o")
	}
}

func TestParseAIRouteResponseContentAcceptsTopLevelArray(t *testing.T) {
	content := `[{"requested_model":"gpt-4o","items":[{"channel_id":1,"upstream_model":"gpt-4o","priority":1,"weight":100}]}]`

	resp, err := parseAIRouteResponseContent(content)
	if err != nil {
		t.Fatalf("parseAIRouteResponseContent(array) error = %v, want nil", err)
	}
	if len(resp.Routes) != 1 {
		t.Fatalf("parseAIRouteResponseContent(array) routes len = %d, want 1", len(resp.Routes))
	}
}

func TestParseAIRouteResponseContentAcceptsNestedResult(t *testing.T) {
	content := `{"result":{"routes":[{"requested_model":"gpt-4o","items":[{"channel_id":1,"upstream_model":"gpt-4o","priority":1,"weight":100}]}]}}`

	resp, err := parseAIRouteResponseContent(content)
	if err != nil {
		t.Fatalf("parseAIRouteResponseContent(result) error = %v, want nil", err)
	}
	if len(resp.Routes) != 1 {
		t.Fatalf("parseAIRouteResponseContent(result) routes len = %d, want 1", len(resp.Routes))
	}
}

func TestParseAIRouteResponseContentAcceptsSingleRouteObject(t *testing.T) {
	content := `{"requested_model":"gpt-4o","items":[{"channel_id":1,"upstream_model":"gpt-4o","priority":1,"weight":100}]}`

	resp, err := parseAIRouteResponseContent(content)
	if err != nil {
		t.Fatalf("parseAIRouteResponseContent(singleRoute) error = %v, want nil", err)
	}
	if len(resp.Routes) != 1 {
		t.Fatalf("parseAIRouteResponseContent(singleRoute) routes len = %d, want 1", len(resp.Routes))
	}
}

func TestParseAIRouteResponseContentRejectsInvalidContent(t *testing.T) {
	content := `不是 JSON，也没有结果`

	if _, err := parseAIRouteResponseContent(content); err == nil {
		t.Fatal("parseAIRouteResponseContent(invalid) error = nil, want non-nil")
	}
}

func TestDecodeAIRouteResponseCandidatePrefersRouteShape(t *testing.T) {
	candidate := `{"routes":[{"requested_model":"gpt-4o","items":[{"channel_id":1,"upstream_model":"gpt-4o","priority":1,"weight":100}]}],"note":"ok"}`

	resp, ok := decodeAIRouteResponseCandidate(candidate)
	if !ok {
		t.Fatal("decodeAIRouteResponseCandidate(routeShape) ok = false, want true")
	}
	want := model.AIRouteResponse{
		Routes: []model.AIRouteEntry{{
			RequestedModel: "gpt-4o",
			Items: []model.AIRouteItemSpec{{
				ChannelID:     1,
				UpstreamModel: "gpt-4o",
				Priority:      1,
				Weight:        100,
			}},
		}},
	}

	if len(resp.Routes) != len(want.Routes) || resp.Routes[0].RequestedModel != want.Routes[0].RequestedModel {
		t.Fatalf("decodeAIRouteResponseCandidate(routeShape) = %+v, want %+v", resp, want)
	}
}
