package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/dlclark/regexp2"
	"github.com/gin-gonic/gin"
	"github.com/lingyuins/octopus/internal/helper"
	"github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/op"
	"github.com/lingyuins/octopus/internal/server/middleware"
	"github.com/lingyuins/octopus/internal/server/resp"
	"github.com/lingyuins/octopus/internal/server/router"
	"github.com/lingyuins/octopus/internal/utils/log"
)

func init() {
	router.NewGroupRouter("/api/v1/group").
		Use(middleware.Auth()).
		Use(middleware.RequireJSON()).
		AddRoute(
			router.NewRoute("/list", http.MethodGet).
				Handle(getGroupList),
		).
		AddRoute(
			router.NewRoute("/create", http.MethodPost).
				Handle(createGroup),
		).
		AddRoute(
			router.NewRoute("/update", http.MethodPost).
				Handle(updateGroup),
		).
		AddRoute(
			router.NewRoute("/auto-group", http.MethodPost).
				Handle(autoGroupModels),
		).
		AddRoute(
			router.NewRoute("/test", http.MethodPost).
				Handle(startGroupTest),
		).
		AddRoute(
			router.NewRoute("/test/progress/:id", http.MethodGet).
				Handle(getGroupTestProgress),
		).
		AddRoute(
			router.NewRoute("/delete-all", http.MethodDelete).
				Handle(deleteAllGroups),
		).
		AddRoute(
			router.NewRoute("/delete/:id", http.MethodDelete).
				Handle(deleteGroup),
		)
	// AddRoute(
	// 	router.NewRoute("/auto-add-item", http.MethodPost).
	// 		Handle(autoAddGroupItem),
	// )
}

func getGroupList(c *gin.Context) {
	groups, err := op.GroupList(c.Request.Context())
	if err != nil {
		resp.InternalError(c)
		return
	}
	resp.Success(c, groups)
}

func createGroup(c *gin.Context) {
	var group model.Group
	if err := c.ShouldBindJSON(&group); err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	group.EndpointType = model.NormalizeEndpointType(group.EndpointType)
	if group.MatchRegex != "" {
		_, err := regexp2.Compile(group.MatchRegex, regexp2.ECMAScript)
		if err != nil {
			resp.Error(c, http.StatusBadRequest, err.Error())
			return
		}
	}
	if err := op.GroupCreate(&group, c.Request.Context()); err != nil {
		if status, message, ok := classifyGroupMutationError(err); ok {
			resp.Error(c, status, message)
			return
		}
		log.Errorf("create group failed: %v", err)
		resp.InternalError(c)
		return
	}
	resp.Success(c, group)
}

func updateGroup(c *gin.Context) {
	var req model.GroupUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.EndpointType != nil {
		normalized := model.NormalizeEndpointType(*req.EndpointType)
		req.EndpointType = &normalized
	}
	if req.MatchRegex != nil {
		_, err := regexp2.Compile(*req.MatchRegex, regexp2.ECMAScript)
		if err != nil {
			resp.Error(c, http.StatusBadRequest, err.Error())
			return
		}
	}
	group, err := op.GroupUpdate(&req, c.Request.Context())
	if err != nil {
		if status, message, ok := classifyGroupMutationError(err); ok {
			resp.Error(c, status, message)
			return
		}
		log.Errorf("update group failed: %v", err)
		resp.InternalError(c)
		return
	}
	resp.Success(c, group)
}

func classifyGroupMutationError(err error) (int, string, bool) {
	if err == nil {
		return 0, "", false
	}

	msg := strings.ToLower(err.Error())

	switch {
	case strings.Contains(msg, "group name is required"):
		return http.StatusBadRequest, "group name is required", true
	case strings.Contains(msg, "endpoint_type") &&
		(strings.Contains(msg, "no such column") ||
			strings.Contains(msg, "has no column named") ||
			strings.Contains(msg, "unknown column") ||
			strings.Contains(msg, "does not exist")):
		return http.StatusServiceUnavailable, "database schema is outdated; restart the service to apply the latest migrations", true
	case strings.Contains(msg, "unique constraint failed: groups.name") ||
		(strings.Contains(msg, "duplicate entry") && strings.Contains(msg, "groups.name")) ||
		(strings.Contains(msg, "duplicate key value violates unique constraint") &&
			(strings.Contains(msg, "groups_name") || strings.Contains(msg, "groups.name"))):
		return http.StatusConflict, "group name already exists", true
	case strings.Contains(msg, "unique constraint failed: group_items.group_id, group_items.channel_id, group_items.model_name") ||
		(strings.Contains(msg, "duplicate entry") && strings.Contains(msg, "idx_group_channel_model")) ||
		(strings.Contains(msg, "duplicate key value violates unique constraint") && strings.Contains(msg, "idx_group_channel_model")):
		return http.StatusConflict, "group contains duplicate channel/model items", true
	default:
		return 0, "", false
	}
}

func startGroupTest(c *gin.Context) {
	var req helper.GroupModelTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	group, err := op.GroupGet(req.GroupID, c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusNotFound, "group not found")
		return
	}

	channels := make(map[int]model.Channel, len(group.Items))
	for _, item := range group.Items {
		if _, ok := channels[item.ChannelID]; ok {
			continue
		}
		channel, err := op.ChannelGet(item.ChannelID, c.Request.Context())
		if err != nil {
			continue
		}
		channels[item.ChannelID] = *channel
	}

	progress, err := helper.StartGroupModelTest(group, channels)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	resp.Success(c, progress)
}

func getGroupTestProgress(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		resp.Error(c, http.StatusBadRequest, "missing progress id")
		return
	}

	progress, ok := helper.GetGroupModelTestProgress(id)
	if !ok {
		resp.Error(c, http.StatusNotFound, "group test progress not found")
		return
	}

	resp.Success(c, progress)
}

func autoGroupModels(c *gin.Context) {
	result, err := op.AutoGroupModels(c.Request.Context())
	if err != nil {
		resp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	resp.Success(c, result)
}

func deleteGroup(c *gin.Context) {
	id := c.Param("id")
	idNum, err := strconv.Atoi(id)
	if err != nil {
		resp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := op.GroupDel(idNum, c.Request.Context()); err != nil {
		resp.InternalError(c)
		return
	}
	resp.Success(c, "group deleted successfully")
}

func deleteAllGroups(c *gin.Context) {
	deletedCount, err := op.GroupDelAll(c.Request.Context())
	if err != nil {
		resp.InternalError(c)
		return
	}
	resp.Success(c, gin.H{"deleted_count": deletedCount})
}

// func autoAddGroupItem(c *gin.Context) {
// 	var req struct {
// 		ID int `json:"id"`
// 	}
// 	if err := c.ShouldBindJSON(&req); err != nil {
// 		resp.Error(c, http.StatusBadRequest, err.Error())
// 		return
// 	}
// 	if req.ID <= 0 {
// 		resp.Error(c, http.StatusBadRequest, "invalid id")
// 		return
// 	}
// 	err := worker.AutoAddGroupItem(req.ID, c.Request.Context())
// 	if err != nil {
// 		resp.Error(c, http.StatusInternalServerError, err.Error())
// 		return
// 	}
// 	resp.Success(c, nil)
// }
