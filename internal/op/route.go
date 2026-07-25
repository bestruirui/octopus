package op

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/utils/log"
	"github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/utils/cache"
	"gorm.io/gorm/clause"
)

var routeCache = cache.New[int, model.Route](16)

func RouteList(ctx context.Context) ([]model.Route, error) {
	routes := make([]model.Route, 0, routeCache.Len())
	for _, route := range routeCache.GetAll() {
		routes = append(routes, route)
	}
	return routes, nil
}

func RouteGet(id int, ctx context.Context) (*model.Route, error) {
	route, ok := routeCache.Get(id)
	if !ok {
		return nil, fmt.Errorf("route not found")
	}
	return &route, nil
}

func RouteGetByName(name string, ctx context.Context) (*model.Route, error) {
	for _, route := range routeCache.GetAll() {
		if route.Name == name {
			return &route, nil
		}
	}
	return nil, fmt.Errorf("route not found")
}

func RouteCreate(req *model.RouteCreateRequest, ctx context.Context) (*model.Route, error) {
	// 验证主分组存在
	if _, err := GroupGet(req.PrimaryGroupID, ctx); err != nil {
		return nil, fmt.Errorf("primary group not found: %w", err)
	}
	// 验证分派分组存在
	if req.DispatchGroupID != nil {
		if _, err := GroupGet(*req.DispatchGroupID, ctx); err != nil {
			return nil, fmt.Errorf("dispatch group not found: %w", err)
		}
	}

	route := &model.Route{
		Name:            req.Name,
		PrimaryGroupID:  req.PrimaryGroupID,
		DispatchGroupID: req.DispatchGroupID,
		Description:     req.Description,
	}

	tx := db.GetDB().WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Create(route).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create route: %w", err)
	}

	// 创建工作分组
	if len(req.WorkGroups) > 0 {
		workGroups := make([]model.RouteGroup, 0, len(req.WorkGroups))
		for _, wg := range req.WorkGroups {
			if _, err := GroupGet(wg.GroupID, ctx); err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("work group %d not found: %w", wg.GroupID, err)
			}
			workGroups = append(workGroups, model.RouteGroup{
				RouteID:     route.ID,
				GroupID:     wg.GroupID,
				Category:    wg.Category,
				Description: wg.Description,
				Keywords:    wg.Keywords,
			})
		}
		if err := tx.Create(&workGroups).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to create work groups: %w", err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	// 刷新缓存
	if err := routeRefreshCacheByID(route.ID, ctx); err != nil {
		return nil, err
	}

	r, _ := routeCache.Get(route.ID)
	return &r, nil
}

func RouteUpdate(req *model.RouteUpdateRequest, ctx context.Context) (*model.Route, error) {
	if _, ok := routeCache.Get(req.ID); !ok {
		return nil, fmt.Errorf("route not found")
	}

	tx := db.GetDB().WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var selectFields []string
	updates := model.Route{ID: req.ID}

	if req.Name != nil {
		selectFields = append(selectFields, "name")
		updates.Name = *req.Name
	}
	if req.PrimaryGroupID != nil {
		if _, err := GroupGet(*req.PrimaryGroupID, ctx); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("primary group not found: %w", err)
		}
		selectFields = append(selectFields, "primary_group_id")
		updates.PrimaryGroupID = *req.PrimaryGroupID
	}
	if req.DispatchGroupID != nil {
		if *req.DispatchGroupID > 0 {
			if _, err := GroupGet(*req.DispatchGroupID, ctx); err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("dispatch group not found: %w", err)
			}
		}
		selectFields = append(selectFields, "dispatch_group_id")
		updates.DispatchGroupID = req.DispatchGroupID
	}
	if req.Description != nil {
		selectFields = append(selectFields, "description")
		updates.Description = *req.Description
	}

	if len(selectFields) > 0 {
		if err := tx.Model(&model.Route{}).Where("id = ?", req.ID).Select(selectFields).Updates(&updates).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to update route: %w", err)
		}
	}

	// 删除工作分组
	if len(req.WorkGroupsToDelete) > 0 {
		if err := tx.Where("id IN ? AND route_id = ?", req.WorkGroupsToDelete, req.ID).Delete(&model.RouteGroup{}).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to delete work groups: %w", err)
		}
	}

	// 更新工作分组
	if len(req.WorkGroupsToUpdate) > 0 {
		for _, wg := range req.WorkGroupsToUpdate {
			updates := map[string]interface{}{}
			if wg.GroupID > 0 {
				if _, err := GroupGet(wg.GroupID, ctx); err != nil {
					tx.Rollback()
					return nil, fmt.Errorf("work group %d not found: %w", wg.GroupID, err)
				}
				updates["group_id"] = wg.GroupID
			}
			if wg.Category != "" {
				updates["category"] = wg.Category
			}
			if wg.Description != "" {
				updates["description"] = wg.Description
			}
			if wg.Keywords != "" {
				updates["keywords"] = wg.Keywords
			}
			if len(updates) > 0 {
				if err := tx.Model(&model.RouteGroup{}).Where("id = ? AND route_id = ?", wg.ID, req.ID).Updates(updates).Error; err != nil {
					tx.Rollback()
					return nil, fmt.Errorf("failed to update work group: %w", err)
				}
			}
		}
	}

	// 新增工作分组
	if len(req.WorkGroupsToAdd) > 0 {
		newGroups := make([]model.RouteGroup, 0, len(req.WorkGroupsToAdd))
		for _, wg := range req.WorkGroupsToAdd {
			if _, err := GroupGet(wg.GroupID, ctx); err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("work group %d not found: %w", wg.GroupID, err)
			}
			newGroups = append(newGroups, model.RouteGroup{
				RouteID:     req.ID,
				GroupID:     wg.GroupID,
				Category:    wg.Category,
				Description: wg.Description,
				Keywords:    wg.Keywords,
			})
		}
		if err := tx.Create(&newGroups).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to create work groups: %w", err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	// 刷新缓存
	if err := routeRefreshCacheByID(req.ID, ctx); err != nil {
		return nil, err
	}

	r, _ := routeCache.Get(req.ID)
	return &r, nil
}

func RouteDel(id int, ctx context.Context) error {
	route, ok := routeCache.Get(id)
	if !ok {
		return fmt.Errorf("route not found")
	}

	tx := db.GetDB().WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 检查是否有 API Key 绑定此路由（通过 SupportedRoutes）
	var apiKeys []model.APIKey
	if err := tx.Where("supported_routes LIKE ?", "%"+route.Name+"%").Find(&apiKeys).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to check api keys: %w", err)
	}
	// 精确匹配路由名称
	boundCount := 0
	for _, key := range apiKeys {
		for _, name := range strings.Split(key.SupportedRoutes, ",") {
			if strings.TrimSpace(name) == route.Name {
				boundCount++
				break
			}
		}
	}
	if boundCount > 0 {
		tx.Rollback()
		return fmt.Errorf("route is bound to %d API keys, unbind first", boundCount)
	}

	if err := tx.Where("route_id = ?", id).Delete(&model.RouteGroup{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete route groups: %w", err)
	}

	if err := tx.Delete(&model.Route{}, id).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete route: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	routeCache.Del(id)
	return nil
}

// RouteResult 路由解析结果，携带分组和分派信息
type RouteResult struct {
	Group          model.Group
	DispatchGroup  string           // 分派分组名（空=关键词匹配，非空=dispatch LLM 或兜底）
	DispatchResult string           // 分派结果描述（如 "keyword: Analysis(16)" 或 "llm: coding(15)"）
	DispatchCall   DispatchCallInfo // dispatch LLM 调用详情（供调用方写日志）
}

// dispatchCacheEntry 分派结果缓存
type dispatchCacheEntry struct {
	result    RouteResult
	cachedAt  time.Time
}

var dispatchCacheTTL = 5 * time.Minute
var dispatchCacheData sync.Map // key: "routeID:requestModel:contentHash" → dispatchCacheEntry

// contentCacheKey 为 dispatch LLM 结果生成包含内容哈希的缓存键
func contentCacheKey(routeID int, requestModel string, requestContent string) string {
	h := sha256.Sum256([]byte(requestContent))
	return fmt.Sprintf("%d:%s:%x", routeID, requestModel, h[:8])
}

// RouteResolve 根据路由解析应该使用的分组
// 优先级：关键词匹配 → 分派分组 LLM 分析 → 主分组兜底
func RouteResolve(routeID int, requestModel string, requestContent string, ctx context.Context) (RouteResult, error) {
	route, ok := routeCache.Get(routeID)
	if !ok {
		return RouteResult{}, fmt.Errorf("route not found")
	}

	// 1. 关键词匹配工作分组（快速路径，不缓存，直接匹配用户消息内容）
	if requestContent != "" {
		if match := matchWorkGroup(route.WorkGroups, requestContent); match != nil {
			result := RouteResult{
				DispatchGroup:  "",
				DispatchResult: fmt.Sprintf("keyword: %s(%d)", match.Category, match.GroupID),
			}
			group, err := GroupGetEnabled(match.GroupID, ctx)
			if err == nil && len(group.Items) > 0 {
				result.Group = group
				log.Infof("route resolve: keyword matched group=%s (group_id=%d)", match.Category, match.GroupID)
				return result, nil
			}
		}
	}

	// 2. 分派分组 LLM 分析（当配置了分派分组且关键词未命中时，调用 LLM 选择工作分组）
	if route.DispatchGroupID != nil && *route.DispatchGroupID > 0 {
		// 检查 dispatch LLM 缓存（基于内容哈希，避免相同内容重复调 LLM）
		cacheKey := contentCacheKey(routeID, requestModel, requestContent)
		if cached, ok := dispatchCacheData.Load(cacheKey); ok {
			entry := cached.(dispatchCacheEntry)
			if time.Since(entry.cachedAt) < dispatchCacheTTL {
				log.Infof("route resolve: dispatch cache hit for route=%s → %s", route.Name, entry.result.DispatchResult)
				return entry.result, nil
			}
			dispatchCacheData.Delete(cacheKey)
		}

		log.Infof("route resolve: dispatch LLM starting for route=%s, model=%s", route.Name, requestModel)
		group, dispatchGroup, dispatchDesc, callInfo, err := RouteDispatch(&route, requestModel, requestContent, ctx)
		if err == nil {
			result := RouteResult{
				Group:          group,
				DispatchGroup:  dispatchGroup,
				DispatchResult: dispatchDesc,
				DispatchCall:   callInfo,
			}
			dispatchCacheData.Store(cacheKey, dispatchCacheEntry{result: result, cachedAt: time.Now()})
			return result, nil
		}
		log.Warnf("dispatch LLM fallback: route=%s, err=%v", route.Name, err)
	}

	// 3. 主分组兜底
	group, err := GroupGetEnabled(route.PrimaryGroupID, ctx)
	if err != nil {
		return RouteResult{}, fmt.Errorf("primary group not found or no enabled items")
	}
	result := RouteResult{
		Group:          group,
		DispatchGroup:  "primary",
		DispatchResult: "fallback",
	}
	return result, nil
}

// matchWorkGroup 根据用户消息内容匹配工作分组（匹配 Keywords 和 Description）
func matchWorkGroup(workGroups []model.RouteGroup, requestContent string) *model.RouteGroup {
	requestContent = strings.ToLower(requestContent)
	for i := range workGroups {
		wg := &workGroups[i]
		if matchKeywords(wg.Keywords, requestContent) || matchKeywords(wg.Description, requestContent) {
			return wg
		}
	}
	return nil
}

// matchKeywords 检查逗号分隔的关键词列表是否包含匹配项
func matchKeywords(field string, target string) bool {
	if field == "" {
		return false
	}
	for _, kw := range strings.Split(field, ",") {
		kw = strings.TrimSpace(strings.ToLower(kw))
		if kw != "" && strings.Contains(target, kw) {
			return true
		}
	}
	return false
}


func routeRefreshCacheByID(id int, ctx context.Context) error {
	var route model.Route
	if err := db.GetDB().WithContext(ctx).
		Preload("WorkGroups").
		First(&route, id).Error; err != nil {
		return err
	}
	routeCache.Set(route.ID, route)
	return nil
}

func routeRefreshCache(ctx context.Context) error {
	routes := []model.Route{}
	if err := db.GetDB().WithContext(ctx).
		Preload("WorkGroups").
		Find(&routes).Error; err != nil {
		return err
	}
	for _, route := range routes {
		routeCache.Set(route.ID, route)
	}
	return nil
}

// RouteGroupUpsert 批量 upsert 工作分组（避免重复）
func RouteGroupUpsert(routeID int, groups []model.RouteGroup, ctx context.Context) error {
	if len(groups) == 0 {
		return nil
	}
	for i := range groups {
		groups[i].RouteID = routeID
	}
	return db.GetDB().WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "route_id"}, {Name: "category"}},
			DoUpdates: clause.AssignmentColumns([]string{"group_id", "description", "keywords"}),
		}).
		Create(&groups).Error
}
