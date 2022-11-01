package service

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/pennsieve/pennsieve-agent/pkg/store"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"time"
)

// UserService provides methods associated with user information and profiles
type UserService struct {
	uiStore store.UserInfoStore
	usStore store.UserSettingsStore
	client  *pennsieve.Client
}

type UserDTO struct {
	Id               string    `json:"id"`
	Name             string    `json:"name"`
	SessionToken     string    `json:"session_token"`
	RefreshToken     string    `json:"refresh_token"`
	TokenExpire      time.Time `json:"token_expire"`
	IdToken          string    `json:"id_token"`
	Profile          string    `json:"profile"`
	Environment      string    `json:"environment"`
	ApiHost          string    `json:"api_host"`
	Api2Host         string    `json:"api2_host"`
	OrganizationId   string    `json:"organization_id"`
	OrganizationName string    `json:"organization_name"`
}

// NewUserService returns a new instance of a UserService.
func NewUserService(uis store.UserInfoStore, uss store.UserSettingsStore) *UserService {
	return &UserService{
		uiStore: uis,
		usStore: uss,
	}
}

// SetPennsieveClient adds a Pennsieve Client to the Service.
// This is not done in the NewUserService as that generates a cyclical dependency.
func (s *UserService) SetPennsieveClient(client *pennsieve.Client) {
	s.client = client
}

func (s *UserService) UpdateActiveDataset(datasetId string) error {
	err := s.usStore.UpdateActiveDataset(datasetId)
	return err
}

func (s *UserService) GetUserSettings() (*store.UserSettings, error) {
	userSettings, err := s.usStore.Get()
	return userSettings, err
}

func (s *UserService) GetActiveUserId() (string, error) {
	userSettings, err := s.usStore.Get()
	if err != nil {
		return "", fmt.Errorf("No active user found in %s.\n",
			viper.ConfigFileUsed())
	}

	return userSettings.UserId, nil

}

// GetActiveUser returns the user that is currently set in UserSettings table
// This method does not update the active session, or handles situations where
// the active user is not set.
//
// Use the UpdateActiveUser method to handle those situations.
func (s *UserService) GetActiveUser() (*UserDTO, error) {
	// Get current user-settings. This is either 0, or 1 entry.
	userSettings, err := s.usStore.Get()

	var currentUserInfo *store.UserInfo
	if err != nil {

		// If no entry is found in database, check default profile in config and setup DB
		if errors.Is(err, &store.NoClientSessionError{}) {
			fmt.Println("No record found in User Settings --> Checking Default Profile.")

			selectedProfile := viper.GetString("global.default_profile")
			fmt.Println("Selected Profile: ", selectedProfile)

			if selectedProfile == "" {
				return nil, fmt.Errorf("No default profile defined in %s. Please update configuration.\n",
					viper.ConfigFileUsed())
			}

			// Create new user settings
			params := store.UserSettingsParams{
				UserId:  "",
				Profile: selectedProfile,
			}
			_, err = s.usStore.CreateNewUserSettings(params)
			if err != nil {
				fmt.Println("Error Creating new UserSettings")
				return nil, err
			}

			currentUserInfo, err = s.SwitchUser(selectedProfile)
			if err != nil {
				fmt.Println("Error switching user.")
				return nil, err
			}

		} else {
			return nil, err
		}

	} else {
		currentUserInfo, err = s.uiStore.GetUserInfo(userSettings.UserId, userSettings.Profile)
		if err != nil {
			return nil, err
		}
	}

	userDTO := UserDTO{
		Id:               currentUserInfo.Id,
		Name:             currentUserInfo.Name,
		SessionToken:     currentUserInfo.SessionToken,
		RefreshToken:     currentUserInfo.RefreshToken,
		TokenExpire:      currentUserInfo.TokenExpire,
		IdToken:          currentUserInfo.IdToken,
		Profile:          currentUserInfo.Profile,
		Environment:      currentUserInfo.Environment,
		ApiHost:          s.client.GetAPIParams().ApiHost,
		Api2Host:         s.client.GetAPIParams().ApiHost2,
		OrganizationId:   currentUserInfo.OrganizationId,
		OrganizationName: currentUserInfo.OrganizationName,
	}

	return &userDTO, nil
}

// SwitchUser switches between profiles and returns active userInfo.
func (s *UserService) SwitchUser(profile string) (*store.UserInfo, error) {

	// Check if profile exist
	isSet := viper.IsSet(profile + ".api_token")
	if !isSet {
		return nil, errors.New(fmt.Sprintf("Profile not found: %s", profile))
	}

	// Profile exists, verify login and refresh token if necessary
	apiToken := viper.GetString(profile + ".api_token")
	apiSecret := viper.GetString(profile + ".api_secret")
	environment := viper.GetString(profile + ".env")
	customUploadBucket := viper.GetString(profile + ".upload_bucket")

	newParams := pennsieve.APIParams{
		ApiKey:        apiToken,
		ApiSecret:     apiSecret,
		Port:          viper.GetString("agent.port"),
		UseConfigFile: true,
		Profile:       profile,
	}

	if customUploadBucket != "" {
		newParams.UploadBucket = customUploadBucket
	} else {
		newParams.UploadBucket = pennsieve.DefaultUploadBucket
	}

	// Directly update baseURL, so we can authenticate against new profile before setting up new Client
	customAPIHost := viper.GetString(profile + ".api_host")
	if customAPIHost != "" {
		newParams.ApiHost = customAPIHost
		newParams.ApiHost2 = "https://api2.pennsieve.net"
	} else {
		newParams.ApiHost = pennsieve.BaseURLV1
		newParams.ApiHost2 = pennsieve.BaseURLV2
	}

	s.client.Updateparams(newParams)

	// Check credentials of new profile
	credentials, err := s.client.Authentication.Authenticate(apiToken, apiSecret)
	if err != nil {
		fmt.Println("Problem with authentication")
		return nil, err
	}

	// Get the User for the new profile
	existingUser, err := s.client.User.GetUser(nil)
	if err != nil {
		fmt.Println("Problem with getting user", err)
		return nil, err
	}

	// Drop existing user settings
	err = s.usStore.Delete()
	if err != nil {
		return nil, err
	}

	// Create new user settings
	usParams := store.UserSettingsParams{
		UserId:  existingUser.ID,
		Profile: profile,
	}
	_, err = s.usStore.CreateNewUserSettings(usParams)
	if err != nil {
		fmt.Println("Error Creating new UserSettings")
		return nil, err
	}

	// Get UserInfo associated with settings or create if not exist.
	newUserInfo, err := s.uiStore.GetUserInfo(existingUser.ID, profile)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("No userInfo found --> Creating new userinfo")

			org, err := s.client.Organization.Get(nil, s.client.OrganizationNodeId)
			if err != nil {
				fmt.Println("Error getting organization")
				return nil, err
			}

			params := store.UserInfoParams{
				Id:               existingUser.ID,
				Name:             existingUser.FirstName + " " + existingUser.LastName,
				SessionToken:     credentials.Token,
				RefreshToken:     credentials.RefreshToken,
				Profile:          profile,
				Environment:      environment,
				OrganizationId:   s.client.OrganizationNodeId,
				OrganizationName: org.Organization.Name,
			}
			newUserInfo, err = s.uiStore.CreateNewUserInfo(params)

			if err != nil {
				fmt.Println("Error creating new userinfo ")
				return nil, err
			}

		} else {
			log.Fatal(err)
		}
	}

	return newUserInfo, nil
}

// ReAuthenticate authenticates user, update server client and return new session.
func (s *UserService) ReAuthenticate() (pennsieve.APISession, error) {
	apiSession, err := s.client.Authentication.ReAuthenticate()

	newSession := pennsieve.APISession{
		Token:        apiSession.Token,
		IdToken:      apiSession.IdToken,
		Expiration:   apiSession.Expiration,
		RefreshToken: apiSession.RefreshToken,
		IsRefreshed:  apiSession.IsRefreshed,
	}

	return newSession, err
}

// UpdateTokenForUser updates the local database with new credentials and returns UserDTO
func (s *UserService) UpdateTokenForUser(user *UserDTO, credentials *pennsieve.APISession) (*UserDTO, error) {

	err := s.uiStore.UpdateTokenForUser(user.Id, credentials)
	if err != nil {
		return nil, err
	}

	user.SessionToken = credentials.Token
	user.RefreshToken = credentials.RefreshToken
	user.TokenExpire = credentials.Expiration
	user.IdToken = credentials.IdToken

	return user, err
}
