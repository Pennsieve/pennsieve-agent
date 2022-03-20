package models

type UserSettings struct {
	UserId       int    `json:"user_id"`
	Profile      string `json:"profile"`
	UseDatasetId string `json:"use_dataset_id"`
}
