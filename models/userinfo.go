package models

import (
	"fmt"
	"github.com/pennsieve/pennsieve-agent/config"
	"github.com/pennsieve/pennsieve-go"
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
	IdToken          string    `json:"id_token"`
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
	IdToken          string
	tokenExpire      time.Time
	Environment      string
	OrganizationId   string
	OrganizationName string
}

func CreateNewUserInfo(data UserInfoParams) (*UserInfo, error) {

	user := &UserInfo{}

	var updatedAt = time.Now().UTC()
	statement, _ := config.DB.Prepare("INSERT INTO user_record (id, name, session_token, refresh_token, profile, " +
		"token_expire, id_token, environment, organization_id, organization_name, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	result, err := statement.Exec(data.Id, data.Name, data.SessionToken, data.RefreshToken, data.Profile,
		data.tokenExpire, data.IdToken, data.Environment, data.OrganizationId, data.OrganizationName, updatedAt)
	if err == nil {
		innerId, _ := result.LastInsertId()
		user.InnerId = int(innerId)
		user.Id = data.Id
		user.Name = data.Name
		user.SessionToken = data.SessionToken
		user.IdToken = data.IdToken
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

func GetUserInfo(id string, profile string) (*UserInfo, error) {
	user := &UserInfo{}
	err := config.DB.QueryRow(
		"SELECT "+
			"inner_id, "+
			"id, "+
			"name, "+
			"session_token, "+
			"id_token, "+
			"profile, "+
			"environment, "+
			"organization_id, "+
			"organization_name, "+
			"updated_at "+
			"FROM user_record WHERE id=? AND profile=?", id, profile).Scan(
		&user.InnerId, &user.Id, &user.Name, &user.SessionToken, &user.IdToken, &user.Profile, &user.Environment,
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
				&currentUser.InnerId, &currentUser.Id, &currentUser.Name, &currentUser.SessionToken, &currentUser.RefreshToken,
				&currentUser.TokenExpire, &currentUser.IdToken,
				&currentUser.Profile, &currentUser.Environment, &currentUser.OrganizationId, &currentUser.OrganizationName,
				&currentUser.UpdatedAt)
			allUsers = append(allUsers, currentUser)
		}
		return allUsers, err
	}
	return allUsers, err
}

func UpdateTokenForUser(user UserInfo, credentials pennsieve.Credentials) (*UserInfo, error) {

	if user.SessionToken != credentials.Token {
		fmt.Println("Session token updated")

		statement, err := config.DB.Prepare(
			"UPDATE user_record SET session_token = ?, refresh_token = ?, token_expire = ?, id_token = ?")
		if err != nil {
			fmt.Println(err)
			return nil, err
		}

		_, err = statement.Exec(credentials.Token, credentials.RefreshToken, credentials.Expiration, credentials.IdToken)
		if err != nil {
			fmt.Sprintln("Unable to update Sessiontoken in database")
			return nil, err
		}

		user.SessionToken = credentials.Token
		user.RefreshToken = credentials.RefreshToken
		user.TokenExpire = credentials.Expiration
		user.IdToken = credentials.IdToken
	}

	return &user, nil
}
