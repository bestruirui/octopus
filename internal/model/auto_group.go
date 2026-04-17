package model

type ChannelModelRef struct {
	ChannelID   int    `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	RawModel    string `json:"raw_model"`
}

type ModelIdentity struct {
	Raw          string `json:"raw"`
	Cleaned      string `json:"cleaned"`
	Canonical    string `json:"canonical"`
	EndpointType string `json:"endpoint_type"`
	Confidence   string `json:"confidence"`
	MatchedRule  string `json:"matched_rule"`
}

type CandidateGroup struct {
	EndpointType string            `json:"endpoint_type"`
	Canonical    string            `json:"canonical"`
	RawModels    []string          `json:"raw_models"`
	ChannelIDs   []int             `json:"channel_ids"`
	Refs         []ChannelModelRef `json:"refs"`
	MatchRegex   string            `json:"match_regex"`
}

type AutoGroupResult struct {
	TotalChannels          int                    `json:"total_channels"`
	TotalModelsSeen        int                    `json:"total_models_seen"`
	TotalDistinctRawModels int                    `json:"total_distinct_raw_models"`
	TotalCandidates        int                    `json:"total_candidates"`
	CreatedGroups          int                    `json:"created_groups"`
	SkippedExistingGroups  int                    `json:"skipped_existing_groups"`
	SkippedCoveredModels   int                    `json:"skipped_covered_models"`
	FailedGroups           int                    `json:"failed_groups"`
	Created                []AutoGroupCreatedItem `json:"created"`
	Skipped                []AutoGroupSkippedItem `json:"skipped"`
}

type AutoGroupCreatedItem struct {
	Name          string   `json:"name"`
	EndpointType  string   `json:"endpoint_type"`
	MatchedModels []string `json:"matched_models"`
}

type AutoGroupSkippedItem struct {
	Name         string `json:"name"`
	EndpointType string `json:"endpoint_type"`
	Reason       string `json:"reason"`
}
