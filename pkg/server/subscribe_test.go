package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt"
	"github.com/golang-migrate/migrate/v4"
	//"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pennsieve/pennsieve-agent/pkg/shared/test"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve/models/authentication"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve/models/organization"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve/models/user"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var (
	userID               = "N:user:8888"
	userFirst            = "Harry"
	userLast             = "Proctor"
	org1ID               = "N:organization:1111"
	org2ID               = "N:organization:2222"
	expectedUserProfiles = []UserProfile{
		{Profile: Profile{
			Name:      "profile-1",
			APIToken:  "profile-1-key",
			APISecret: "profile-1-secret",
		},
			User: user.User{
				ID:                    userID,
				FirstName:             userFirst,
				LastName:              userLast,
				PreferredOrganization: org1ID,
			},
			Org: organization.Organization{
				ID:   org1ID,
				Name: "Organization 1"}},
		{Profile: Profile{
			Name:      "profile-2",
			APIToken:  "profile-2-key",
			APISecret: "profile-2-secret",
		},
			User: user.User{
				ID:                    userID,
				FirstName:             userFirst,
				LastName:              userLast,
				PreferredOrganization: org2ID},
			Org: organization.Organization{
				ID:   org2ID,
				Name: "Organization 2"}},
	}
)

type ServerTestSuite struct {
	suite.Suite
	dbPath        string
	db            *sql.DB
	mockPennsieve *MockPennsieve
	testServer    *agentServer
}

func (suite *ServerTestSuite) SetupSuite() {
	dbDir := suite.T().TempDir()
	dbPath := filepath.Join(dbDir, "pennsieve_server_test.db")
	suite.dbPath = dbPath
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on&mode=rwc&_journal_mode=WAL")
	if err != nil {
		suite.FailNow("could not open database", "%s", err)
	}

	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	m, err := migrate.NewWithDatabaseInstance(
		"file://../../db/migrations",
		"sqlite3", driver)

	//m, err := migrate.New(
	//	"file://../../db/migrations",
	//	fmt.Sprintf("sqlite3://%s?_foreign_keys=on&mode=rwc&_journal_mode=WAL", dbPath),
	//)
	if err != nil {
		suite.T().Fatal(err)
	}
	if err := m.Up(); err != nil {
		suite.T().Fatal(err)
	}

	testDataPath := filepath.Join("..", "..", "test", "sql", "server-test-data.sql")
	err = test.LoadTestData(db, testDataPath)
	if err != nil {
		suite.T().Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		suite.T().Fatal(err)
	}

	suite.db = db
}

func (suite *ServerTestSuite) clearDatabase() {
	for _, t := range []string{"manifests", "manifest_files", "user_record", "user_settings"} {
		q := fmt.Sprintf("DELETE FROM %s", t)
		_, err := suite.db.Exec(q)
		if err != nil {
			suite.Fail("could not truncate table", "table: %s, error: %s", t, err)
		}
	}
}

// Programmatically inits viper and for each profile sets api_host the suite's mock URLs.
// Also updates config.AWSEndpoints to point to mock Cognito
func (suite *ServerTestSuite) initConfig() {
	// Initialize Viper
	viper.Set("agent.db_path", suite.dbPath)
	viper.Set("agent.useConfigFile", true)
	viper.Set("global.default_profile", expectedUserProfiles[0].Profile.Name)
	viper.Set("migration.path", "file://../../db/migrations")
	for _, up := range expectedUserProfiles {
		profile := up.Profile
		viper.Set(profile.Name+".api_token", profile.APIToken)
		viper.Set(profile.Name+".api_secret", profile.APISecret)
		viper.Set(profile.Name+".api_host", suite.mockPennsieve.API.Server.URL)
	}
	pennsieve.AWSEndpoints = pennsieve.AWSCognitoEndpoints{IdentityProviderEndpoint: suite.mockPennsieve.IDProvider.Server.URL}
}

func (suite *ServerTestSuite) SetupTest() {
	suite.mockPennsieve = NewMockPennsieve(suite.T(), authentication.CognitoConfig{
		Region: "us-east-1",
		UserPool: authentication.UserPool{
			Region:      "us-east-1",
			ID:          "mock-user-pool-id",
			AppClientID: "mock-user-pool-app-client-id",
		},
		TokenPool: authentication.TokenPool{
			Region:      "us-east-1",
			AppClientID: "mockTokenPoolAppClientId",
		},
		IdentityPool: authentication.IdentityPool{
			Region: "us-east-1",
			ID:     "mock-identity-pool-id",
		}},
		expectedUserProfiles...)

	suite.clearDatabase()
	suite.initConfig()
	suite.Require().NoError(test.LoadTestData(suite.db, "../../test/sql/server-test-data.sql"))

	testServer, err := NewAgentServer(grpc.NewServer())
	suite.db = testServer.SqliteDB()
	suite.NoError(err)
	suite.testServer = testServer

}

func (suite *ServerTestSuite) TearDownTest() {
	suite.mockPennsieve.Close()
	viper.Reset()
	pennsieve.AWSEndpoints.Reset()
}

func (suite *ServerTestSuite) TearDownSuite() {
	if suite.db != nil {
		if err := suite.db.Close(); err != nil {
			suite.Fail("could not close database", "%s", err)
		}
	}
}

type MockServer struct {
	Server *httptest.Server
	Mux    *http.ServeMux
}

func (m *MockServer) Close() {
	m.Server.Close()
}

type Profile struct {
	Name      string
	APIToken  string
	APISecret string
}

type UserProfile struct {
	Profile Profile
	User    user.User
	Org     organization.Organization
}

type MockPennsieve struct {
	API                 *MockServer
	IDProvider          *MockServer
	APIKeyToUserProfile map[string]UserProfile
	JWTToAPIKey         map[string]string
}

func (m *MockPennsieve) Close() {
	m.API.Close()
	m.IDProvider.Close()
}

const (
	OrgNodeIdClaimKey = "custom:organization_node_id"
	OrgIdClaimKey     = "custom:organization_id"
)

func (m *MockPennsieve) NewJwt(t *testing.T, apiKey string, sessionTTL time.Duration) string {
	userConfig, ok := m.APIKeyToUserProfile[apiKey]
	if !ok {
		t.Fatalf("No user found for key %q in %v", apiKey, m.APIKeyToUserProfile)
	}
	orgNodeId := userConfig.User.PreferredOrganization
	orgId := fmt.Sprintf("%d", userConfig.Org.IntID)
	claims := jwt.MapClaims{
		"exp":             time.Now().Add(sessionTTL).UTC().Unix(),
		OrgNodeIdClaimKey: orgNodeId,
		OrgIdClaimKey:     orgId,
	}
	idToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	idTokenString, err := idToken.SignedString([]byte("test-signing-key"))
	if err != nil {
		t.Errorf("error getting signed string from JWT token: %s", err)
	}
	return idTokenString
}

func (m *MockPennsieve) attachNewMockIDProviderServer(t *testing.T) {
	mock := MockServer{}
	mock.Mux = http.NewServeMux()
	mock.Server = httptest.NewServer(mock.Mux)
	mock.Mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/" {
			t.Errorf("unexpected cognito identity provider call: expected: %q, got: %q", "/", request.URL)
		}
		reqMap := map[string]any{}
		err := json.NewDecoder(request.Body).Decode(&reqMap)
		if err != nil {
			t.Fatal(err)
		}
		authParams := reqMap["AuthParameters"].(map[string]any)
		apiKey, ok := authParams["USERNAME"].(string)
		if !ok {
			t.Fatal("USERNAME not set in authentication request")
		}
		idTokenString := m.NewJwt(t, apiKey, time.Hour)
		accessToken := uuid.NewString()
		m.JWTToAPIKey[accessToken] = apiKey

		_, err = fmt.Fprintf(writer, `{"AuthenticationResult": {"AccessToken": %q, "ExpiresIn": 3600, "IdToken": %q, "RefreshToken": %q, "TokenType": "Bearer"}, "ChallengeParameters": {}}`,
			accessToken,
			idTokenString,
			"mock-refresh-token")
		if err != nil {
			t.Error("error writing AuthenticationResult")
		}
	})
	m.IDProvider = &mock

}

func (m *MockPennsieve) attachNewMockPennsieveServer(t *testing.T, expectedCognitoConfig authentication.CognitoConfig) {
	mock := MockServer{}
	mock.Mux = http.NewServeMux()
	mock.Server = httptest.NewServer(mock.Mux)
	mock.Mux.HandleFunc("/authentication/cognito-config", func(writer http.ResponseWriter, request *http.Request) {
		body, err := json.Marshal(expectedCognitoConfig)
		if err != nil {
			t.Error("could not marshal mock CognitoConfig")
		}
		_, err = writer.Write(body)
		if err != nil {
			t.Error("error writing CognitoConfig response")
		}
	})
	mock.Mux.HandleFunc("/user", func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "GET", request.Method)
		up := m.lookupUserProfile(t, request)
		respBytes, err := json.Marshal(up.User)
		if assert.NoError(t, err) {
			_, err = writer.Write(respBytes)
			assert.NoError(t, err)
		}
	})
	orgsPathPrefix := "/organizations/"
	mock.Mux.HandleFunc(orgsPathPrefix, func(writer http.ResponseWriter, request *http.Request) {
		assert.Equal(t, "GET", request.Method)
		orgId := strings.TrimPrefix(request.URL.Path, orgsPathPrefix)
		assert.NotEmpty(t, orgId)
		up := m.lookupUserProfile(t, request)
		respBytes, err := json.Marshal(organization.GetOrganizationResponse{Organization: up.Org})
		if assert.NoError(t, err) {
			_, err = writer.Write(respBytes)
			assert.NoError(t, err)
		}

	})
	mock.Mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		t.Errorf("Unhandled request: method: %q, path: %q. If this call is expected add a HandleFunc to MockPennsieveServer.Mux", request.Method, request.URL)
	})
	m.API = &mock
}

func (m *MockPennsieve) lookupUserProfile(t *testing.T, r *http.Request) UserProfile {
	authHeader := r.Header.Get("Authorization")
	assert.NotEmpty(t, authHeader)
	prefix := "Bearer "
	assert.True(t, strings.HasPrefix(authHeader, prefix))
	jwtToken := strings.TrimPrefix(authHeader, prefix)
	assert.NotEmpty(t, jwtToken)
	apiKey, ok := m.JWTToAPIKey[jwtToken]
	if !ok {
		t.Fatalf("No api key found for jwt %q in %v", jwtToken, m.JWTToAPIKey)
	}
	up, ok := m.APIKeyToUserProfile[apiKey]
	if !ok {
		t.Fatalf("No UserProfile found for api key %q in %v", apiKey, m.APIKeyToUserProfile)
	}
	return up
}

func NewMockPennsieve(t *testing.T, cognitoConfig authentication.CognitoConfig, userProfiles ...UserProfile) *MockPennsieve {
	apiKeyToUserProfile := make(map[string]UserProfile, len(userProfiles))
	for _, u := range userProfiles {
		apiKeyToUserProfile[u.Profile.APIToken] = u
	}

	mockPennsieve := MockPennsieve{APIKeyToUserProfile: apiKeyToUserProfile, JWTToAPIKey: map[string]string{}}
	mockPennsieve.attachNewMockIDProviderServer(t)
	mockPennsieve.attachNewMockPennsieveServer(t, cognitoConfig)

	return &mockPennsieve
}
