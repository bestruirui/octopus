package op

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/dlclark/regexp2"
	"github.com/lingyuins/octopus/internal/db"
	"github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/utils/log"
	"github.com/lingyuins/octopus/internal/utils/xstrings"
)

type autoGroupFamilyRule struct {
	canonical string
	ruleName  string
	re        *regexp.Regexp
}

var autoGroupRepeatedDashRe = regexp.MustCompile(`-+`)
var autoGroupNoiseCharRe = regexp.MustCompile(`[^
\p{L}\p{N}\-\._/]+`)

var autoGroupExplicitAliases = map[string]string{
	"dmxapi-5-4":                    "gpt-5.4",
	"5-4":                           "gpt-5.4",
	"dmxapi-cls-4-6":                "claude-sonnet-4.6",
	"cls-4-6":                       "claude-sonnet-4.6",
	"dmxapi-clo-4-6":                "claude-opus-4.6",
	"clo-4-6":                       "claude-opus-4.6",
	"dmxapi-clh45-20251001":         "claude-haiku-4.5",
	"clh45-20251001":                "claude-haiku-4.5",
	"dmxapl-cls45-0929":             "claude-sonnet-4.5",
	"cls45-0929":                    "claude-sonnet-4.5",
	"dmxapl-cls45-0929-sk":          "claude-sonnet-4.5",
	"cls45-0929-sk":                 "claude-sonnet-4.5",
	"dmxapi-clo-45":                 "claude-opus-4.5",
	"clo-45":                        "claude-opus-4.5",
	"dmxapi-mm-m2.1":                "minimax-m2.1",
	"mm-m2.1":                       "minimax-m2.1",
	"dmxapi-mm-m2":                  "minimax-m2",
	"mm-m2":                         "minimax-m2",
	"dmxapi-g3-flash-preview":       "gemini-3-flash-preview",
	"g3-flash-preview":              "gemini-3-flash-preview",
	"dmxapi-g3-pro-preview":         "gemini-3-pro-preview",
	"g3-pro-preview":                "gemini-3-pro-preview",
	"dmxapi-g2.5-flash":             "gemini-2.5-flash",
	"g2.5-flash":                    "gemini-2.5-flash",
	"dmxapi-g2.5-pro":               "gemini-2.5-pro",
	"g2.5-pro":                      "gemini-2.5-pro",
	"dmxapi-g-l-m-4.7":              "glm-4.7",
	"g-l-m-4.7":                     "glm-4.7",
	"dmxapi-g-l-m-4.6":              "glm-4.6",
	"g-l-m-4.6":                     "glm-4.6",
	"dmxapi-gemini-3.1-pro-preview": "gemini-3.1-pro-preview",
	"gemini-3.1-pro-preview":        "gemini-3.1-pro-preview",
	"dmxapi-minimax-m2.5":           "minimax-m2.5",
	"minimax-m2.5":                  "minimax-m2.5",
}

var autoGroupFamilyRules = []autoGroupFamilyRule{
	{canonical: "glm-5.1", ruleName: "glm-5.1", re: regexp.MustCompile(`(^|/)(?:glm-5\.1(?:-(?:guan))?)$|(^|/)glm-5\.1(?:-[0-9a-z]+)?$`)},
	{canonical: "glm-5-turbo", ruleName: "glm-5-turbo", re: regexp.MustCompile(`(^|/)glm-5-turbo(?:-[0-9a-z.]+)?$`)},
	{canonical: "glm-5", ruleName: "glm-5", re: regexp.MustCompile(`(^|/)(?:glm-5(?:-(?:fp8(?:-[0-9]+)?))?|glm5)$`)},
	{canonical: "deepseek-v3.2-thinking", ruleName: "deepseek-v3.2-thinking", re: regexp.MustCompile(`(^|/)deepseek-v3\.2(?:-exp)?-thinking(?:-[0-9a-z.]+)?$`)},
	{canonical: "deepseek-v3.2-fast", ruleName: "deepseek-v3.2-fast", re: regexp.MustCompile(`(^|/)deepseek-v3\.2-fast(?:-[0-9a-z.]+)?$`)},
	{canonical: "deepseek-v3.2", ruleName: "deepseek-v3.2", re: regexp.MustCompile(`(^|/)deepseek-v3\.2(?:-exp)?(?:-[0-9a-z.]+)?$`)},
	{canonical: "deepseek-r1-search", ruleName: "deepseek-r1-search", re: regexp.MustCompile(`(^|/)deepseek-r1-search(?:-[0-9a-z.]+)?$`)},
	{canonical: "deepseek-r1", ruleName: "deepseek-r1", re: regexp.MustCompile(`(^|/)(?:deepseek-r1(?:-[0-9a-z.]+)?|huoshan-deepseek-r1-671b-64k)$`)},
	{canonical: "qwen3.5-plus", ruleName: "qwen3.5-plus", re: regexp.MustCompile(`(^|/)qwen3\.5-plus(?:-[0-9]{4}-[0-9]{2}-[0-9]{2})?$`)},
	{canonical: "qwen3.5-flash", ruleName: "qwen3.5-flash", re: regexp.MustCompile(`(^|/)qwen3\.5-flash(?:-[0-9]{4}-[0-9]{2}-[0-9]{2})?$`)},
	{canonical: "qwen3-max", ruleName: "qwen3-max", re: regexp.MustCompile(`(^|/)qwen3-max(?:-[0-9]{4}-[0-9]{2}-[0-9]{2})?$`)},
	{canonical: "qwen-plus", ruleName: "qwen-plus", re: regexp.MustCompile(`(^|/)qwen-plus(?:-[0-9]{4}-[0-9]{2}-[0-9]{2})?$`)},
	{canonical: "qwen-flash", ruleName: "qwen-flash", re: regexp.MustCompile(`(^|/)qwen-flash(?:-[0-9]{4}-[0-9]{2}-[0-9]{2})?$`)},
	{canonical: "gpt-5-chat", ruleName: "gpt-5-chat", re: regexp.MustCompile(`(^|/)gpt-5-chat-latest$|(^|/)gpt-5-chat$`)},
	{canonical: "gpt-5", ruleName: "gpt-5", re: regexp.MustCompile(`(^|/)gpt-5(?:-[0-9]{4}-[0-9]{2}-[0-9]{2})?$`)},
	{canonical: "gpt-4o-mini", ruleName: "gpt-4o-mini", re: regexp.MustCompile(`(^|/)gpt-4o-mini(?:-[0-9]{4}-[0-9]{2}-[0-9]{2})?$`)},
	{canonical: "gpt-4o", ruleName: "gpt-4o", re: regexp.MustCompile(`(^|/)(?:gpt-4o|chatgpt-4o-latest)(?:-[0-9]{4}-[0-9]{2}-[0-9]{2})?$`)},
	{canonical: "gpt-4.1", ruleName: "gpt-4.1", re: regexp.MustCompile(`(^|/)gpt-4\.1(?:-[0-9]{4}-[0-9]{2}-[0-9]{2})?$`)},
	{canonical: "claude-sonnet-4.6-thinking", ruleName: "claude-sonnet-4.6-thinking", re: regexp.MustCompile(`(^|/)claude-sonnet-4(?:[.-])6-thinking(?:-[0-9]{8})?$`)},
	{canonical: "claude-opus-4.6-thinking", ruleName: "claude-opus-4.6-thinking", re: regexp.MustCompile(`(^|/)claude-opus-4(?:[.-])6-thinking(?:-[0-9]{8})?$`)},
	{canonical: "claude-sonnet-4.6", ruleName: "claude-sonnet-4.6", re: regexp.MustCompile(`(^|/)claude-sonnet-4(?:[.-])6(?:-[0-9]{8})?$`)},
	{canonical: "claude-opus-4.6", ruleName: "claude-opus-4.6", re: regexp.MustCompile(`(^|/)claude-opus-4(?:[.-])6(?:-[0-9]{8})?$`)},
	{canonical: "claude-sonnet-4.5", ruleName: "claude-sonnet-4.5", re: regexp.MustCompile(`(^|/)claude-sonnet-4(?:[.-])5(?:-[0-9]{8})?$`)},
	{canonical: "claude-haiku-4.5", ruleName: "claude-haiku-4.5", re: regexp.MustCompile(`(^|/)claude-haiku-4(?:[.-])5(?:-[0-9]{8})?$`)},
	{canonical: "claude-opus-4.5", ruleName: "claude-opus-4.5", re: regexp.MustCompile(`(^|/)claude-opus-4(?:[.-])5(?:-[0-9]{8})?$`)},
	{canonical: "gemini-3.1-pro-preview", ruleName: "gemini-3.1-pro-preview", re: regexp.MustCompile(`(^|/)gemini-3\.1-pro-preview$`)},
	{canonical: "gemini-3-pro-preview", ruleName: "gemini-3-pro-preview", re: regexp.MustCompile(`(^|/)gemini-3-pro-preview$`)},
	{canonical: "gemini-3-flash-preview", ruleName: "gemini-3-flash-preview", re: regexp.MustCompile(`(^|/)gemini-3-flash-preview$`)},
	{canonical: "gemini-2.5-pro", ruleName: "gemini-2.5-pro", re: regexp.MustCompile(`(^|/)gemini-2\.5-pro$`)},
	{canonical: "gemini-2.5-flash", ruleName: "gemini-2.5-flash", re: regexp.MustCompile(`(^|/)gemini-2\.5-flash$`)},
	{canonical: "minimax-m2.7", ruleName: "minimax-m2.7", re: regexp.MustCompile(`(^|/)minimax-m2\.7$`)},
	{canonical: "minimax-m2.5", ruleName: "minimax-m2.5", re: regexp.MustCompile(`(^|/)minimax-m2\.5$`)},
	{canonical: "minimax-m2.1", ruleName: "minimax-m2.1", re: regexp.MustCompile(`(^|/)minimax-m2\.1$`)},
	{canonical: "minimax-m2", ruleName: "minimax-m2", re: regexp.MustCompile(`(^|/)minimax-m2$`)},
	{canonical: "doubao-seed-2.0-pro", ruleName: "doubao-seed-2.0-pro", re: regexp.MustCompile(`(^|/)doubao-seed-2(?:[.-])0-pro(?:-[0-9]{6})?$`)},
	{canonical: "doubao-seed-1.6-flash", ruleName: "doubao-seed-1.6-flash", re: regexp.MustCompile(`(^|/)doubao-seed-1(?:[.-])6-flash(?:-[0-9]{6})?$`)},
	{canonical: "doubao-seed-1.6", ruleName: "doubao-seed-1.6", re: regexp.MustCompile(`(^|/)doubao-seed-1(?:[.-])6(?:-[0-9]{6})?$`)},
}

func AutoGroupModels(ctx context.Context) (*model.AutoGroupResult, error) {
	channelRefs, totalChannels, err := collectChannelModelRefs(ctx)
	if err != nil {
		return nil, err
	}

	result := &model.AutoGroupResult{
		TotalChannels:   totalChannels,
		TotalModelsSeen: len(channelRefs),
	}

	rawSeen := make(map[string]struct{}, len(channelRefs))
	candidateMap := make(map[string]*model.CandidateGroup)
	for _, ref := range channelRefs {
		rawSeen[strings.ToLower(strings.TrimSpace(ref.RawModel))] = struct{}{}
		identity := NormalizeModelIdentity(ref.RawModel)
		key := identity.EndpointType + "::" + identity.Canonical
		candidate := candidateMap[key]
		if candidate == nil {
			candidate = &model.CandidateGroup{
				EndpointType: identity.EndpointType,
				Canonical:    identity.Canonical,
			}
			candidateMap[key] = candidate
		}
		candidate.RawModels = appendUniqueString(candidate.RawModels, strings.TrimSpace(ref.RawModel))
		candidate.ChannelIDs = appendUniqueInt(candidate.ChannelIDs, ref.ChannelID)
		candidate.Refs = append(candidate.Refs, ref)
	}

	result.TotalDistinctRawModels = len(rawSeen)
	result.TotalCandidates = len(candidateMap)

	existingGroups, err := GroupList(ctx)
	if err != nil {
		return nil, fmt.Errorf("list groups failed: %w", err)
	}

	groupNameSet := make(map[string]model.Group, len(existingGroups))
	for _, group := range existingGroups {
		groupNameSet[strings.ToLower(strings.TrimSpace(group.Name))] = group
	}

	candidates := make([]model.CandidateGroup, 0, len(candidateMap))
	for _, candidate := range candidateMap {
		candidate.MatchRegex = buildCandidateMatchRegex(*candidate)
		candidates = append(candidates, *candidate)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].EndpointType != candidates[j].EndpointType {
			return candidates[i].EndpointType < candidates[j].EndpointType
		}
		return candidates[i].Canonical < candidates[j].Canonical
	})

	for _, candidate := range candidates {
		covered, reason := IsCandidateCoveredByExistingGroups(candidate, existingGroups)
		if covered {
			if reason == "same canonical group already exists" || reason == "group name already exists" {
				result.SkippedExistingGroups++
			} else {
				result.SkippedCoveredModels += len(candidate.RawModels)
			}
			result.Skipped = append(result.Skipped, model.AutoGroupSkippedItem{
				Name:         candidate.Canonical,
				EndpointType: candidate.EndpointType,
				Reason:       reason,
			})
			continue
		}

		if conflict, ok := groupNameSet[strings.ToLower(candidate.Canonical)]; ok {
			result.SkippedExistingGroups++
			result.Skipped = append(result.Skipped, model.AutoGroupSkippedItem{
				Name:         candidate.Canonical,
				EndpointType: candidate.EndpointType,
				Reason:       fmt.Sprintf("group name already exists with endpoint %s", conflict.EndpointType),
			})
			continue
		}

		if err := createAutoGroupCandidate(candidate, ctx); err != nil {
			result.FailedGroups++
			result.Skipped = append(result.Skipped, model.AutoGroupSkippedItem{
				Name:         candidate.Canonical,
				EndpointType: candidate.EndpointType,
				Reason:       err.Error(),
			})
			continue
		}

		groupNameSet[strings.ToLower(candidate.Canonical)] = model.Group{Name: candidate.Canonical, EndpointType: candidate.EndpointType}
		existingGroups = append(existingGroups, model.Group{Name: candidate.Canonical, EndpointType: candidate.EndpointType, MatchRegex: candidate.MatchRegex})
		result.CreatedGroups++
		result.Created = append(result.Created, model.AutoGroupCreatedItem{
			Name:          candidate.Canonical,
			EndpointType:  candidate.EndpointType,
			MatchedModels: candidate.RawModels,
		})
	}

	log.Infof("auto group finished: channels=%d seen_models=%d distinct_raw=%d candidates=%d created=%d skipped_existing=%d skipped_covered_models=%d failed=%d",
		result.TotalChannels,
		result.TotalModelsSeen,
		result.TotalDistinctRawModels,
		result.TotalCandidates,
		result.CreatedGroups,
		result.SkippedExistingGroups,
		result.SkippedCoveredModels,
		result.FailedGroups,
	)

	return result, nil
}

func collectChannelModelRefs(ctx context.Context) ([]model.ChannelModelRef, int, error) {
	channels, err := ChannelList(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("list channels failed: %w", err)
	}

	refs := make([]model.ChannelModelRef, 0)
	for _, channel := range channels {
		for _, rawModel := range xstrings.SplitTrimCompact(",", channel.Model, channel.CustomModel) {
			refs = append(refs, model.ChannelModelRef{
				ChannelID:   channel.ID,
				ChannelName: channel.Name,
				RawModel:    rawModel,
			})
		}
	}
	log.Infof("auto group scan: channels=%d model_refs=%d", len(channels), len(refs))
	return refs, len(channels), nil
}

func CleanModelName(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	s = strings.ReplaceAll(s, "_", "-")
	s = autoGroupNoiseCharRe.ReplaceAllString(s, "")
	parts := strings.Split(s, "/")
	for i := range parts {
		parts[i] = cleanModelSegment(parts[i])
	}
	s = strings.Join(parts, "/")
	s = autoGroupRepeatedDashRe.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-./")
	return s
}

func cleanModelSegment(segment string) string {
	segment = strings.TrimSpace(segment)
	segment = strings.ReplaceAll(segment, "g-p-t", "gpt")
	segment = strings.ReplaceAll(segment, "g-l-m", "glm")
	segment = strings.TrimPrefix(segment, "dmxapi-")
	segment = strings.TrimPrefix(segment, "dmxapl-")
	for _, suffix := range []string{"-free", "-cc", "-ssvip", "-sk"} {
		segment = strings.TrimSuffix(segment, suffix)
	}
	segment = autoGroupRepeatedDashRe.ReplaceAllString(segment, "-")
	return strings.Trim(segment, "-")
}

func InferEndpointTypeFromModelName(raw string, cleaned string) string {
	s := strings.ToLower(raw + " " + cleaned)
	if containsAny(s, "rerank", "reranker", "colbert") {
		return model.EndpointTypeRerank
	}
	if containsAny(s, "embedding", "embeddings", "bge-m3", "jina-clip") {
		return model.EndpointTypeEmbeddings
	}
	if containsAny(s, "tts", "speech", "voice") {
		return model.EndpointTypeAudioSpeech
	}
	if containsAny(s, "transcribe", "transcription", "whisper") {
		return model.EndpointTypeAudioTranscription
	}
	if containsAny(s, "deepsearch", "deep-research", "deepresearch", "sonar", "search") {
		return model.EndpointTypeSearch
	}
	if containsAny(s, "video-generation", "seendance", "kling", "vidu", "paiwo", "ttv", "itv", "i2v", "t2v", "r2v", "hailuo") {
		return model.EndpointTypeVideoGeneration
	}
	if containsAny(s, "gpt-image", "dall-e", "qwen-image", "imagen", "seedream") ||
		regexp.MustCompile(`wan.*(?:image|t2i)`).MatchString(s) || strings.Contains(s, "sd-wan") {
		return model.EndpointTypeImageGeneration
	}
	if containsAny(s, "moderation", "moderations") {
		return model.EndpointTypeModerations
	}
	return model.EndpointTypeChat
}

func NormalizeModelIdentity(raw string) model.ModelIdentity {
	cleaned := CleanModelName(raw)
	endpointType := InferEndpointTypeFromModelName(raw, cleaned)
	for _, key := range identityAliasKeys(raw, cleaned) {
		if canonical, ok := autoGroupExplicitAliases[key]; ok {
			return model.ModelIdentity{
				Raw:          raw,
				Cleaned:      cleaned,
				Canonical:    canonical,
				EndpointType: endpointType,
				Confidence:   "alias",
				MatchedRule:  key,
			}
		}
	}
	for _, sample := range identityMatchSamples(raw, cleaned) {
		for _, rule := range autoGroupFamilyRules {
			if rule.re.MatchString(sample) {
				return model.ModelIdentity{
					Raw:          raw,
					Cleaned:      cleaned,
					Canonical:    rule.canonical,
					EndpointType: endpointType,
					Confidence:   "regex",
					MatchedRule:  rule.ruleName,
				}
			}
		}
	}
	return model.ModelIdentity{
		Raw:          raw,
		Cleaned:      cleaned,
		Canonical:    cleaned,
		EndpointType: endpointType,
		Confidence:   "fallback",
		MatchedRule:  "cleaned",
	}
}

func IsCandidateCoveredByExistingGroups(candidate model.CandidateGroup, existingGroups []model.Group) (bool, string) {
	for _, group := range existingGroups {
		if !sameEndpointType(group.EndpointType, candidate.EndpointType) {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(group.Name), candidate.Canonical) {
			return true, "same canonical group already exists"
		}
		for _, item := range group.Items {
			for _, rawModel := range candidate.RawModels {
				if strings.EqualFold(strings.TrimSpace(item.ModelName), strings.TrimSpace(rawModel)) {
					return true, "already covered by existing group item"
				}
			}
		}
		regex := strings.TrimSpace(group.MatchRegex)
		if regex == "" {
			continue
		}
		re, err := regexp2.Compile(regex, regexp2.ECMAScript)
		if err != nil {
			continue
		}
		for _, rawModel := range candidate.RawModels {
			identity := NormalizeModelIdentity(rawModel)
			for _, sample := range []string{rawModel, identity.Cleaned, candidate.Canonical} {
				matched, matchErr := re.MatchString(sample)
				if matchErr == nil && matched {
					return true, "already covered by existing match regex"
				}
			}
		}
	}
	return false, ""
}

func createAutoGroupCandidate(candidate model.CandidateGroup, ctx context.Context) error {
	tx := db.GetDB().WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	group := model.Group{
		Name:              candidate.Canonical,
		EndpointType:      model.NormalizeEndpointType(candidate.EndpointType),
		Mode:              model.GroupModeRoundRobin,
		MatchRegex:        candidate.MatchRegex,
		FirstTokenTimeOut: 0,
		SessionKeepTime:   0,
	}
	if err := tx.Create(&group).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("create group failed: %w", err)
	}

	items := make([]model.GroupItem, 0, len(candidate.Refs))
	seen := make(map[string]struct{}, len(candidate.Refs))
	priority := 1
	for _, ref := range candidate.Refs {
		k := fmt.Sprintf("%d|%s", ref.ChannelID, strings.TrimSpace(ref.RawModel))
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		items = append(items, model.GroupItem{
			GroupID:   group.ID,
			ChannelID: ref.ChannelID,
			ModelName: strings.TrimSpace(ref.RawModel),
			Priority:  priority,
			Weight:    1,
		})
		priority++
	}
	if len(items) > 0 {
		if err := tx.Create(&items).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("create group items failed: %w", err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("commit auto group failed: %w", err)
	}
	return groupRefreshCacheByID(group.ID, ctx)
}

func buildCandidateMatchRegex(candidate model.CandidateGroup) string {
	patterns := make([]string, 0, len(candidate.RawModels)*2+1)
	seen := make(map[string]struct{})
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		patterns = append(patterns, regexp.QuoteMeta(value))
	}
	add(candidate.Canonical)
	for _, rawModel := range candidate.RawModels {
		add(rawModel)
		add(CleanModelName(rawModel))
		if tail := lastModelSegment(CleanModelName(rawModel)); tail != "" {
			add(tail)
		}
	}
	sort.Strings(patterns)
	if len(patterns) == 0 {
		return ""
	}
	return `(?i)^(?:` + strings.Join(patterns, "|") + `)$`
}

func identityAliasKeys(raw string, cleaned string) []string {
	keys := []string{
		strings.ToLower(strings.TrimSpace(raw)),
		cleaned,
		lastModelSegment(cleaned),
	}
	for _, item := range []string{strings.ToLower(strings.TrimSpace(raw)), cleaned, lastModelSegment(cleaned)} {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		keys = append(keys, strings.TrimPrefix(item, "dmxapi-"), strings.TrimPrefix(item, "dmxapl-"))
	}
	return uniqueStrings(keys)
}

func identityMatchSamples(raw string, cleaned string) []string {
	samples := []string{
		strings.ToLower(strings.TrimSpace(raw)),
		cleaned,
		lastModelSegment(cleaned),
	}
	return uniqueStrings(samples)
}

func appendUniqueString(items []string, value string) []string {
	value = strings.TrimSpace(value)
	for _, item := range items {
		if strings.EqualFold(item, value) {
			return items
		}
	}
	return append(items, value)
}

func appendUniqueInt(items []int, value int) []int {
	for _, item := range items {
		if item == value {
			return items
		}
	}
	return append(items, value)
}

func uniqueStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		key := strings.ToLower(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}

func lastModelSegment(name string) string {
	if idx := strings.LastIndex(name, "/"); idx >= 0 && idx < len(name)-1 {
		return name[idx+1:]
	}
	return name
}

func sameEndpointType(existing string, target string) bool {
	existing = model.NormalizeEndpointType(existing)
	target = model.NormalizeEndpointType(target)
	return existing == target || existing == model.EndpointTypeAll || target == model.EndpointTypeAll
}

func containsAny(s string, values ...string) bool {
	for _, value := range values {
		if strings.Contains(s, value) {
			return true
		}
	}
	return false
}
