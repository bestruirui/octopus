package relay

import (
	dbmodel "github.com/bestruirui/octopus/internal/model"
	"github.com/bestruirui/octopus/internal/relay/balancer"
	"github.com/gin-gonic/gin"
	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/transformer"
)

// relayRun 保存一次客户端请求在负载均衡循环中共享的状态。
type relayRun struct {
	c               *gin.Context
	inAdapter       transformer.Inbound
	internalRequest *llm.Request
	metrics         *RelayMetrics
	iter            *balancer.Iterator
	group           dbmodel.Group
	route           *dbmodel.Route // 可选，当 API Key 绑定了路由时有值
}

// relayAttempt 保存一次上游通道尝试的状态。
type relayAttempt struct {
	*relayRun

	outAdapter transformer.Outbound
	channel    *dbmodel.Channel
	usedKey    dbmodel.ChannelKey
}
