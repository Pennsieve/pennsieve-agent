package api

import (
	"github.com/pennsieve/pennsieve-agent/models"
	"github.com/pennsieve/pennsieve-go"
)

var PennsieveClient *pennsieve.Client
var ActiveUser *models.UserInfo

func InitializeAPI() (*models.UserInfo, error) {
	// Initialize Pennsieve Client
	PennsieveClient = pennsieve.NewClient()

	var err error
	ActiveUser, err = GetActiveUser()

	return ActiveUser, err
}
