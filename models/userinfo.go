package models

import (
	"github.com/pennsieve/pennsieve-agent/config"
	"log"
	"time"
)

type UserInfo struct {
	InnerId          int       `json:"inner_id"`
	Id               string    `json:"id"`
	Name             string    `json:"name"`
	SessionToken     string    `json:"session_token"`
	RefreshToken     string    `json:"refresh_token"`
	TokenExpire      time.Time `json:"token_expire"`
	Profile          string    `json:"profile"'`
	Environment      string    `json:"environment"`
	OrganizationId   string    `json:"organization_id"`
	OrganizationName string    `json:"organization_name"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type UserInfoParams struct {
	Id               string
	Name             string
	SessionToken     string
	RefreshToken     string
	Profile          string
	tokenExpire      time.Time
	Environment      string
	OrganizationId   string
	OrganizationName string
}

func CreateNewUserInfo(data UserInfoParams) (*UserInfo, error) {

	user := &UserInfo{}

	var updatedAt = time.Now().UTC()
	statement, _ := config.DB.Prepare("INSERT INTO user_record (id, name, session_token, refresh_token, profile, " +
		"token_expire, environment, organization_id, organization_name, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	result, err := statement.Exec(data.Id, data.Name, data.SessionToken, data.RefreshToken, data.Profile,
		data.tokenExpire, data.Environment, data.OrganizationId, data.OrganizationName, updatedAt)
	if err == nil {
		innerId, _ := result.LastInsertId()
		user.InnerId = int(innerId)
		user.Id = data.Id
		user.Name = data.Name
		user.SessionToken = data.SessionToken
		user.Profile = data.Profile
		user.Environment = data.Environment
		user.OrganizationId = data.OrganizationId
		user.OrganizationName = data.OrganizationName
		user.UpdatedAt = updatedAt
		return user, err
	}
	log.Println("Unable to create user_record", err.Error())
	return user, err
}

func GetUserInfo(id int) (*UserInfo, error) {
	user := &UserInfo{}
	err := config.DB.QueryRow(
		"SELECT "+
			"inner_id, "+
			"id, "+
			"name, "+
			"session_token, "+
			"profile, "+
			"environment, "+
			"organization_id, "+
			"organization_name, "+
			"updated_at "+
			"FROM user_record WHERE inner_id=?", id).Scan(
		&user.InnerId, &user.Id, &user.Name, &user.SessionToken, &user.Profile, &user.Environment,
		&user.OrganizationId, &user.OrganizationName, &user.UpdatedAt)
	return user, err
}

func (user *UserInfo) GetAll() ([]UserInfo, error) {
	rows, err := config.DB.Query("SELECT * FROM user_record")
	var allUsers []UserInfo
	if err == nil {
		for rows.Next() {
			var currentUser UserInfo
			_ = rows.Scan(
				&currentUser.InnerId, &currentUser.Id, &currentUser.Name, &currentUser.SessionToken, &currentUser.RefreshToken, &currentUser.TokenExpire,
				&currentUser.Profile, &currentUser.Environment, &currentUser.OrganizationId, &currentUser.OrganizationName,
				&currentUser.UpdatedAt)
			allUsers = append(allUsers, currentUser)
		}
		return allUsers, err
	}
	return allUsers, err
}
