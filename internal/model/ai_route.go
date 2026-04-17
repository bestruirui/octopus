package model

type GenerateAIRouteRequest struct {
	GroupID int `json:"group_id,omitempty"`
}

type GenerateAIRouteResult struct {
	GroupID    int `json:"group_id"`
	RouteCount int `json:"route_count"`
	ItemCount  int `json:"item_count"`
}

type AIRouteModelInput struct {
	ChannelID   int    `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	Provider    string `json:"provider"`
	Model       string `json:"model"`
}

type AIRouteResponse struct {
	Routes []AIRouteEntry `json:"routes"`
}

type AIRouteEntry struct {
	RequestedModel string            `json:"requested_model"`
	Items          []AIRouteItemSpec `json:"items"`
}

type AIRouteItemSpec struct {
	ChannelID     int    `json:"channel_id"`
	UpstreamModel string `json:"upstream_model"`
	Priority      int    `json:"priority"`
	Weight        int    `json:"weight"`
}

func (GenerateAIRouteRequest) TableName() string { return "-" }
func (GenerateAIRouteResult) TableName() string  { return "-" }
func (AIRouteModelInput) TableName() string      { return "-" }
func (AIRouteResponse) TableName() string        { return "-" }
func (AIRouteEntry) TableName() string           { return "-" }
func (AIRouteItemSpec) TableName() string        { return "-" }
