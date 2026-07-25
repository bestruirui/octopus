package model

// Route 路由 - 分组之上的独立层，根据请求内容自动选择分组
type Route struct {
	ID               int          `json:"id" gorm:"primaryKey"`
	Name             string       `json:"name" gorm:"unique;not null"`
	PrimaryGroupID   int          `json:"primary_group_id" gorm:"not null"`      // 主分组 ID（必选，兜底）
	DispatchGroupID  *int         `json:"dispatch_group_id,omitempty"`            // 分派分组 ID（可选，LLM 分析指派）
	Description      string       `json:"description,omitempty"`                 // 路由描述
	WorkGroups       []RouteGroup `json:"work_groups,omitempty" gorm:"foreignKey:RouteID"`
}

// RouteGroup 路由工作分组 - 每个路由下的分类规则
type RouteGroup struct {
	ID          int    `json:"id" gorm:"primaryKey"`
	RouteID     int    `json:"route_id" gorm:"not null;index:idx_route_category,unique"`
	GroupID     int    `json:"group_id" gorm:"not null"`                       // 关联的分组 ID
	Category    string `json:"category" gorm:"not null;index:idx_route_category,unique"` // 分类名称（如"工具调用"、"编码"）
	Description string `json:"description,omitempty"`                          // 分类描述
	Keywords    string `json:"keywords,omitempty"`                             // 关键词（逗号分隔，用于匹配）
}

// RouteCreateRequest 创建路由请求
type RouteCreateRequest struct {
	Name            string                `json:"name" binding:"required"`
	PrimaryGroupID  int                   `json:"primary_group_id" binding:"required"`
	DispatchGroupID *int                  `json:"dispatch_group_id,omitempty"`
	Description     string                `json:"description,omitempty"`
	WorkGroups      []RouteGroupAddRequest `json:"work_groups,omitempty"`
}

// RouteUpdateRequest 更新路由请求 - 仅包含变更的数据
type RouteUpdateRequest struct {
	ID              int                      `json:"id" binding:"required"`
	Name            *string                  `json:"name,omitempty"`
	PrimaryGroupID  *int                     `json:"primary_group_id,omitempty"`
	DispatchGroupID *int                     `json:"dispatch_group_id,omitempty"`
	Description     *string                  `json:"description,omitempty"`
	WorkGroupsToAdd    []RouteGroupAddRequest    `json:"work_groups_to_add,omitempty"`
	WorkGroupsToUpdate []RouteGroupUpdateRequest `json:"work_groups_to_update,omitempty"`
	WorkGroupsToDelete []int                    `json:"work_groups_to_delete,omitempty"`
}

// RouteGroupAddRequest 新增工作分组请求
type RouteGroupAddRequest struct {
	GroupID     int    `json:"group_id" binding:"required"`
	Category    string `json:"category" binding:"required"`
	Description string `json:"description,omitempty"`
	Keywords    string `json:"keywords,omitempty"`
}

// RouteGroupUpdateRequest 更新工作分组请求
type RouteGroupUpdateRequest struct {
	ID          int    `json:"id" binding:"required"`
	GroupID     int    `json:"group_id,omitempty"`
	Category    string `json:"category,omitempty"`
	Description string `json:"description,omitempty"`
	Keywords    string `json:"keywords,omitempty"`
}
