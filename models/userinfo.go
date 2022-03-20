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
	Profile          string    `json:"profile"`
	Environment      string    `json:"environment"`
	OrganizationId   string    `json:"organization_id"`
	OrganizationName string    `json:"organization_name"`
	EncryptionKey    string    `json:"encryption_key"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type UserInfoParams struct {
	Id               string
	Name             string
	SessionToken     string
	Profile          string
	Environment      string
	OrganizationId   string
	OrganizationName string
	EncryptionKey    string
}

func (user *UserInfo) Create(data UserInfoParams) (*UserInfo, error) {
	var updatedAt = time.Now().UTC()
	statement, _ := config.DB.Prepare("INSERT INTO user_record (id, name, session_token, profile, " +
		"environment, organization_id, organization_name, encryption_key, updated_at) VALUES (?, ?, ?, ?)")
	result, err := statement.Exec(data.Id, data.Name, data.SessionToken, data.Profile,
		data.Environment, data.OrganizationId, data.OrganizationName, updatedAt)
	if err == nil {
		innerId, _ := result.LastInsertId()
		user.InnerId = int(innerId)
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

func (user *UserInfo) Fetch(id string) (*UserInfo, error) {
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
			"encryption_key, "+
			"updated_at "+
			"FROM user_record WHERE id=?", id).Scan(
		&user.InnerId, &user.Id, &user.Name, &user.SessionToken, &user.Profile, &user.Environment,
		&user.OrganizationId, &user.OrganizationName, &user.EncryptionKey, &user.UpdatedAt)
	return user, err
}

func (user *UserInfo) GetAll() ([]UserInfo, error) {
	rows, err := config.DB.Query("SELECT * FROM user_record")
	var allUsers []UserInfo
	if err == nil {
		for rows.Next() {
			var currentUser UserInfo
			_ = rows.Scan(
				&user.InnerId, &user.Id, &user.Name, &user.SessionToken, &user.Profile, &user.Environment,
				&user.OrganizationId, &user.OrganizationName, &user.EncryptionKey, &user.UpdatedAt)
			allUsers = append(allUsers, currentUser)
		}
		return allUsers, err
	}
	return allUsers, err
}
