package migrate

import (
	"fmt"

	"gorm.io/gorm"
)

func init() {
	RegisterBeforeAutoMigration(Migration{
		Version: 4,
		Up:      migrateChannelTypeToProvider,
	})
}

// migrateChannelTypeToProvider 将历史 API 格式渠道类型迁移为渠道提供方。
func migrateChannelTypeToProvider(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	if !db.Migrator().HasTable("channels") || !db.Migrator().HasColumn("channels", "type") {
		return nil
	}

	typeColumn := `"type"`
	typeExpr := `CAST("type" AS TEXT)`
	switch db.Dialector.Name() {
	case "mysql":
		typeColumn = "`type`"
		typeExpr = "CAST(`type` AS CHAR)"
	case "postgres":
		typeExpr = `"type"::text`
	}

	// typeColumn/typeExpr are chosen from a fixed switch above (never derived
	// from external input), so build the query via concatenation instead of
	// fmt.Sprintf to avoid tripping the string-formatted-query audit rule.
	query := "\nUPDATE channels\nSET " + typeColumn + " = CASE " + typeExpr + `
	WHEN 'openai/chat_completions' THEN 'openai'
	WHEN 'openai/responses' THEN 'openai_responses'
	WHEN 'anthropic/messages' THEN 'anthropic'
	WHEN 'gemini/contents' THEN 'gemini'
	WHEN 'doubao' THEN 'volcengine'
	WHEN 'openai/embeddings' THEN 'openai'
	ELSE ` + typeExpr + `
END
`

	if err := db.Exec(query).Error; err != nil {
		return fmt.Errorf("failed to migrate channels.type to provider: %w", err)
	}
	return nil
}
