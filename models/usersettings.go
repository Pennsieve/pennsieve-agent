package models

import (
	"fmt"
	"github.com/pennsieve/pennsieve-agent/config"
	"log"
)

type UserSettings struct {
	UserId       string `json:"user_id"`
	Profile      string `json:"profile"`
	UseDatasetId string `json:"use_dataset_id"`
}

type UserSettingsParams struct {
	UserId  string
	Profile string
}

func GetAllUserSettings() ([]UserSettings, error) {
	rows, err := config.DB.Query("SELECT * FROM user_settings")
	if err != nil {
		fmt.Println("Error getting all rows from User_Settings table")
		return nil, err
	}

	var allConfigs []UserSettings
	for rows.Next() {
		var currentConfig UserSettings
		_ = rows.Scan(
			&currentConfig.UserId, &currentConfig.Profile, &currentConfig.UseDatasetId)
		allConfigs = append(allConfigs, currentConfig)
	}
	return allConfigs, err

}

func CreateNewUserSettings(data UserSettingsParams) (*UserSettings, error) {
	userSettings := &UserSettings{}
	statement, _ := config.DB.Prepare("INSERT INTO user_settings (user_id, profile) VALUES (?, ?)")
	_, err := statement.Exec(data.UserId, data.Profile)
	if err != nil {
		log.Println("Unable to create user_record", err.Error())
		return nil, err
	}

	userSettings.UserId = data.UserId
	userSettings.Profile = data.Profile

	return userSettings, err
}
