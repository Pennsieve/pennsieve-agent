package store

import (
	"database/sql"
	"fmt"
	"log"
)

type UserSettings struct {
	UserId          string `json:"user_id"`
	Profile         string `json:"profile"`
	UseDatasetId    string `json:"use_dataset_id"`
	UploadSessionId string `json:"upload_session_id"`
}

type UserSettingsParams struct {
	UserId  string
	Profile string
}

type UserSettingsStore interface {
	Get() (*UserSettings, error)
	CreateNewUserSettings(data UserSettingsParams) (*UserSettings, error)
	UpdateActiveDataset(datasetId string) error
	Delete() error
}

func NewUserSettingsStore(db *sql.DB) *userSettingsStore {
	return &userSettingsStore{
		db: db,
	}
}

type userSettingsStore struct {
	db *sql.DB
}

// Get returns the UserSettings object or nil if no user-settings are defined.
func (s *userSettingsStore) Get() (*UserSettings, error) {
	rows, err := s.db.Query("SELECT * FROM user_settings")
	if err != nil {
		return nil, err
	}

	var allConfigs []UserSettings
	for rows.Next() {
		var currentConfig UserSettings
		_ = rows.Scan(
			&currentConfig.UserId, &currentConfig.Profile, &currentConfig.UseDatasetId)
		allConfigs = append(allConfigs, currentConfig)
	}

	// Return first element as UserSettings should always have 0 or 1 rows
	if len(allConfigs) > 0 {
		return &allConfigs[0], err
	} else {
		return nil, &NoClientSessionError{}
	}

}

// CreateNewUserSettings creates or replaces existing user-settings row in config.
func (s *userSettingsStore) CreateNewUserSettings(data UserSettingsParams) (*UserSettings, error) {
	userSettings := &UserSettings{}
	statement, _ := s.db.Prepare("INSERT INTO user_settings (user_id, profile) VALUES (?, ?)")
	_, err := statement.Exec(data.UserId, data.Profile)
	if err != nil {
		log.Println("Unable to create user_record", err.Error())
		return nil, err
	}

	userSettings.UserId = data.UserId
	userSettings.Profile = data.Profile

	return userSettings, err
}

func (s *userSettingsStore) UpdateActiveDataset(datasetId string) error {
	statement, err := s.db.Prepare(
		"UPDATE user_settings SET use_dataset_id = ?")
	if err != nil {
		return err
	}

	_, err = statement.Exec(datasetId)
	if err != nil {
		fmt.Sprintln("Unable to update ActiveDataset in database")
		return err
	}

	return nil

}

func (s *userSettingsStore) Delete() error {
	_, err := s.db.Exec("DELETE FROM user_settings;")
	return err
}

type NoClientSessionError struct{}

func (m *NoClientSessionError) Error() string {
	return "No client session found in the database."
}
