package model

import (
	"fmt"
	"time"
)

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

type AIRouteTaskStatus string

const (
	AIRouteTaskStatusQueued    AIRouteTaskStatus = "queued"
	AIRouteTaskStatusRunning   AIRouteTaskStatus = "running"
	AIRouteTaskStatusCompleted AIRouteTaskStatus = "completed"
	AIRouteTaskStatusFailed    AIRouteTaskStatus = "failed"
	AIRouteTaskStatusTimeout   AIRouteTaskStatus = "timeout"
)

type AIRouteTaskStep string

const (
	AIRouteTaskStepQueued           AIRouteTaskStep = "queued"
	AIRouteTaskStepCollectingModels AIRouteTaskStep = "collecting_models"
	AIRouteTaskStepBuildingBatches  AIRouteTaskStep = "building_batches"
	AIRouteTaskStepAnalyzingBatches AIRouteTaskStep = "analyzing_batches"
	AIRouteTaskStepParsingResponse  AIRouteTaskStep = "parsing_response"
	AIRouteTaskStepValidatingRoutes AIRouteTaskStep = "validating_routes"
	AIRouteTaskStepWritingGroups    AIRouteTaskStep = "writing_groups"
	AIRouteTaskStepFinalizing       AIRouteTaskStep = "finalizing"
	AIRouteTaskStepCompleted        AIRouteTaskStep = "completed"
	AIRouteTaskStepFailed           AIRouteTaskStep = "failed"
	AIRouteTaskStepTimeout          AIRouteTaskStep = "timeout"
)

type AIRouteChannelStatus string

const (
	AIRouteChannelStatusPending   AIRouteChannelStatus = "pending"
	AIRouteChannelStatusRunning   AIRouteChannelStatus = "running"
	AIRouteChannelStatusCompleted AIRouteChannelStatus = "completed"
	AIRouteChannelStatusFailed    AIRouteChannelStatus = "failed"
)

type GenerateAIRouteProgressSummary struct {
	TotalChannels     int `json:"total_channels"`
	CompletedChannels int `json:"completed_channels"`
	RunningChannels   int `json:"running_channels"`
	PendingChannels   int `json:"pending_channels"`
	FailedChannels    int `json:"failed_channels"`
	TotalModels       int `json:"total_models"`
	CompletedModels   int `json:"completed_models"`
}

type GenerateAIRouteCurrentBatch struct {
	Index        int      `json:"index"`
	Total        int      `json:"total"`
	EndpointType string   `json:"endpoint_type,omitempty"`
	ModelCount   int      `json:"model_count"`
	ChannelIDs   []int    `json:"channel_ids,omitempty"`
	ChannelNames []string `json:"channel_names,omitempty"`
}

type GenerateAIRouteChannelProgress struct {
	ChannelID       int                  `json:"channel_id"`
	ChannelName     string               `json:"channel_name,omitempty"`
	Provider        string               `json:"provider,omitempty"`
	Status          AIRouteChannelStatus `json:"status,omitempty"`
	TotalModels     int                  `json:"total_models"`
	ProcessedModels int                  `json:"processed_models"`
	Message         string               `json:"message,omitempty"`
}

type GenerateAIRouteProgress struct {
	ID               string                           `json:"id"`
	Scope            AIRouteScope                     `json:"scope,omitempty"`
	GroupID          int                              `json:"group_id,omitempty"`
	Status           AIRouteTaskStatus                `json:"status,omitempty"`
	CurrentStep      AIRouteTaskStep                  `json:"current_step,omitempty"`
	ProgressPercent  int                              `json:"progress_percent"`
	TotalBatches     int                              `json:"total_batches"`
	CompletedBatches int                              `json:"completed_batches"`
	Done             bool                             `json:"done"`
	ResultReady      bool                             `json:"result_ready"`
	Message          string                           `json:"message,omitempty"`
	ErrorReason      string                           `json:"error_reason,omitempty"`
	StartedAt        *time.Time                       `json:"started_at,omitempty"`
	UpdatedAt        *time.Time                       `json:"updated_at,omitempty"`
	HeartbeatAt      *time.Time                       `json:"heartbeat_at,omitempty"`
	FinishedAt       *time.Time                       `json:"finished_at,omitempty"`
	EventSequence    int64                            `json:"event_sequence"`
	Summary          *GenerateAIRouteProgressSummary  `json:"summary,omitempty"`
	CurrentBatch     *GenerateAIRouteCurrentBatch     `json:"current_batch,omitempty"`
	Channels         []GenerateAIRouteChannelProgress `json:"channels,omitempty"`
	Result           *GenerateAIRouteResult           `json:"result,omitempty"`
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

func (s AIRouteScope) Valid() bool {
	switch s {
	case AIRouteScopeGroup, AIRouteScopeTable:
		return true
	default:
		return false
	}
}

func (req GenerateAIRouteRequest) Validate() error {
	if !req.Scope.Valid() {
		return fmt.Errorf("invalid scope")
	}
	if req.GroupID < 0 {
		return fmt.Errorf("invalid group_id")
	}
	return nil
}

func (GenerateAIRouteRequest) TableName() string { return "-" }
func (GenerateAIRouteResult) TableName() string  { return "-" }
func (GenerateAIRouteProgressSummary) TableName() string {
	return "-"
}
func (GenerateAIRouteCurrentBatch) TableName() string {
	return "-"
}
func (GenerateAIRouteChannelProgress) TableName() string {
	return "-"
}
func (GenerateAIRouteProgress) TableName() string {
	return "-"
}
func (AIRouteModelInput) TableName() string { return "-" }
func (AIRouteResponse) TableName() string   { return "-" }
func (AIRouteEntry) TableName() string      { return "-" }
func (AIRouteItemSpec) TableName() string   { return "-" }
