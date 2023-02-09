package server

import (
	"context"
	"encoding/json"
	v1 "github.com/pennsieve/pennsieve-agent/api/v1"
	"github.com/pennsieve/pennsieve-agent/pkg/store"
	"github.com/pennsieve/pennsieve-go/pkg/pennsieve/models/dataset"
	"github.com/stretchr/testify/suite"
	"net/http"
	"testing"
)

const expectedDatasetId = "N:dataset:4567"

type DatasetsTestSuite struct {
	ServerTestSuite
}

func (s *DatasetsTestSuite) TestUseDataset() {
	s.mockPennsieve.API.Mux.HandleFunc("/datasets/"+expectedDatasetId, func(writer http.ResponseWriter, request *http.Request) {
		s.Equal("GET", request.Method)
		getDatasetResp := dataset.GetDatasetResponse{
			Content: dataset.Content{ID: expectedDatasetId},
		}
		respBytes, err := json.Marshal(getDatasetResp)
		if s.NoError(err) {
			_, err := writer.Write(respBytes)
			s.NoError(err)
		}
	})
	req := v1.UseDatasetRequest{DatasetId: expectedDatasetId}
	resp, err := s.testServer.UseDataset(context.Background(), &req)
	if s.NoError(err) {
		s.Equal(expectedDatasetId, resp.DatasetId)
		actual := store.UserSettings{}
		row := s.db.QueryRow("select * from user_settings where user_id = ?", expectedUserProfiles[0].User.ID)
		err = row.Scan(&actual.UserId, &actual.Profile, &actual.UseDatasetId)
		if s.NoError(err) {
			s.Equal(expectedDatasetId, actual.UseDatasetId)
		}
	}
}

func TestDatasetsSuite(t *testing.T) {
	suite.Run(t, new(DatasetsTestSuite))
}
