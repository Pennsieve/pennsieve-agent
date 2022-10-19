package service

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/pennsieve/pennsieve-agent/pkg/store"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// UserService provides methods associated with user information and profiles
type UserService struct {
	uiStore store.UserInfoStore
	usStore store.UserSettingsStore
	client  *pennsieve.Client
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
func (s *UserService) GetActiveUser() (*store.UserInfo, error) {
	// Get current user-settings. This is either 0, or 1 entry.
	userSettings, err := s.usStore.Get()

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

			fmt.Printf("about to switch")

			currentUser, err := s.SwitchUser(selectedProfile)
			if err != nil {
				fmt.Println("Error switching user.")
				return nil, err
			}

			return currentUser, nil
		} else {
			return nil, err
		}

	}

	// If entries found in database, continue with active profile
	currentUserInfo, err := s.uiStore.GetUserInfo(userSettings.UserId, userSettings.Profile)
	if err != nil {
		return nil, err
	}

	return currentUserInfo, nil
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

	if customUploadBucket != "" {
		s.client.UploadBucket = customUploadBucket
	}

	// Directly update baseURL, so we can authenticate against new profile before setting up new Client
	customAPIHost := viper.GetString(profile + ".api_host")
	if customAPIHost != "" {
		s.client.SetBasePathForServices(customAPIHost, "https://api2.pennsieve.net")
	} else {
		s.client.SetBasePathForServices(pennsieve.BaseURLV1, pennsieve.BaseURLV2)
	}

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
	params := store.UserSettingsParams{
		UserId:  existingUser.ID,
		Profile: profile,
	}
	_, err = s.usStore.CreateNewUserSettings(params)
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

// UpdateActiveUser returns userInfo for active user and updates local SQlite DB
func (s *UserService) UpdateActiveUser() (*store.UserInfo, error) {

	// Get current user-settings. This is either 0, or 1 entry.
	userSettings, err := s.usStore.Get()

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

			fmt.Printf("about to switch")

			currentUser, err := s.SwitchUser(selectedProfile)
			if err != nil {
				fmt.Println("Error switching user.")
				return nil, err
			}

			return currentUser, nil
		} else {
			return nil, err
		}
	}

	// If entries found in database, continue with active profile
	currentUserInfo, err := s.uiStore.GetUserInfo(userSettings.UserId, userSettings.Profile)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("No userInfo found for user settings")
			_, err := s.SwitchUser(userSettings.Profile)
			if err != nil {
				fmt.Println("error switching user:", err)
			}

		} else {
			log.Fatal(err)
		}

		return nil, err
	}

	s.client.APISession = pennsieve.APISession{
		Token:        currentUserInfo.SessionToken,
		IdToken:      currentUserInfo.IdToken,
		Expiration:   currentUserInfo.TokenExpire,
		RefreshToken: currentUserInfo.RefreshToken,
		IsRefreshed:  false,
	}

	apiToken := viper.GetString("pennsieve.api_token")
	apiSecret := viper.GetString("pennsieve.api_secret")
	//apiToken := viper.GetString(userSettings.Profile + ".api_token")
	//apiSecret := viper.GetString(userSettings.Profile + ".api_secret")

	s.client.APICredentials = pennsieve.APICredentials{
		ApiKey:    apiToken,
		ApiSecret: apiSecret,
	}

	return currentUserInfo, nil
}

func (s *UserService) UpdateTokenForUser(user *store.UserInfo, credentials *pennsieve.APISession) (*store.UserInfo, error) {
	userInfo, err := s.uiStore.UpdateTokenForUser(user, credentials)
	return userInfo, err
}
