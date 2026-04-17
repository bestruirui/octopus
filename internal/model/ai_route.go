package model

type AIRouteScope string

const (
	AIRouteScopeGroup AIRouteScope = "group"
	AIRouteScopeTable AIRouteScope = "table"
)

type GenerateAIRouteRequest struct {
	Scope   AIRouteScope `json:"scope,omitempty"`
	GroupID int          `json:"group_id,omitempty"`
}

type GenerateAIRouteResult struct {
	Scope      AIRouteScope `json:"scope,omitempty"`
	GroupID    int          `json:"group_id,omitempty"`
	GroupCount int          `json:"group_count"`
	RouteCount int          `json:"route_count"`
	ItemCount  int          `json:"item_count"`
}

type GenerateAIRouteProgress struct {
	ID      string                 `json:"id"`
	Scope   AIRouteScope           `json:"scope,omitempty"`
	GroupID int                    `json:"group_id,omitempty"`
	Done    bool                   `json:"done"`
	Message string                 `json:"message,omitempty"`
	Result  *GenerateAIRouteResult `json:"result,omitempty"`
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
	EndpointType   string            `json:"endpoint_type,omitempty"`
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
func (GenerateAIRouteProgress) TableName() string {
	return "-"
}
func (AIRouteModelInput) TableName() string      { return "-" }
func (AIRouteResponse) TableName() string        { return "-" }
func (AIRouteEntry) TableName() string           { return "-" }
func (AIRouteItemSpec) TableName() string        { return "-" }
