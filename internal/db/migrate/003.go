package migrate

import (
	"gorm.io/gorm"
)

func init() {
	RegisterAfterAutoMigration(Migration{
		Version: 3,
		Up:      createRelayLogsTimeIndex,
	})
}

// 003: create index on relay_logs.time for faster time-range queries
func createRelayLogsTimeIndex(db *gorm.DB) error {
	return db.Exec("CREATE INDEX IF NOT EXISTS idx_relay_logs_time ON relay_logs(time)").Error
}
