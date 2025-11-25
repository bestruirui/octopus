package model

type SettingKey string

const (
	SettingKeyProxyURL          SettingKey = "proxy_url"
	SettingKeyStatsSaveInterval SettingKey = "stats_save_interval" //将统计信息写入数据库的周期;单位分钟
)

type Setting struct {
	Key   SettingKey `json:"key" gorm:"primaryKey"`
	Value string     `json:"value" gorm:"not null"`
}

func DefaultSettings() []Setting {
	return []Setting{
		{Key: SettingKeyProxyURL, Value: ""},
		{Key: SettingKeyStatsSaveInterval, Value: "60"}, //默认1小时保存一次数据库
	}
}
