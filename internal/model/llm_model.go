package model

type LLMModel struct {
	Model       string  `json:"model" gorm:"primaryKey;not null"`
	Alias       string  `json:"alias"`
	InputPrice  float64 `json:"input_price" gorm:"default:0"`
	OutputPrice float64 `json:"output_price" gorm:"default:0"`
}
