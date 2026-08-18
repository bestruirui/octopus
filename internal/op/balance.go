package op

import (
	"context"
	"fmt"

	"github.com/bestruirui/octopus/internal/db"
	"github.com/bestruirui/octopus/internal/model"
)

// BalanceSave 把余额快照写入数据库并刷新渠道缓存。
// 使用 Updates 传结构体,让 gorm 的 serializer:json 生效;其余零值字段会被跳过。
func BalanceSave(id int, balance *model.Balance, ctx context.Context) error {
	if err := db.GetDB().WithContext(ctx).Model(&model.Channel{}).Where("id = ?", id).Updates(&model.Channel{Balance: balance}).Error; err != nil {
		return fmt.Errorf("failed to save channel balance: %w", err)
	}
	if channel, ok := channelCache.Get(id); ok {
		channel.Balance = balance
		channelCache.Set(id, channel)
	}
	return nil
}
