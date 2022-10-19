package model

type IncomeInfo struct {
	UserID       int64  `json:"user_id"`
	BotLink      string `json:"bot_link"`
	BotName      string `json:"bot_name"`
	IncomeSource string `json:"income_source"`
	TypeBot      string `json:"type_bot"`
}
