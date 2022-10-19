package model

type GetAdmins struct {
	Code       string   `json:"code"`
	Additional []string `json:"additional"`
}

type GetIncomeInfo struct {
	UserID  int64  `json:"user_id"`
	TypeBot string `json:"type_bot"`
}
