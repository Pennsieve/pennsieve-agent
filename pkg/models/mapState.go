package models

import "time"

// MapState is persisted as a JSON file in the .pennsieve folder
// in a mapped dataset. It contains information about when the
// last time the dataset was fetched from Pennsieve and a list
// of files that have been pulled.
//
// Using the pull-time, we can check if files were changed after
// pulled from Pennsieve.
type MapState struct {
	LastFetch time.Time        `json:"lastFetch"`
	LastPull  time.Time        `json:"lastPull"`
	Files     []MapStateRecord `json:"files"`
}

type MapStateRecord struct {
	FileId   string    `json:"fileId"`
	Path     string    `json:"path"`
	PullTime time.Time `json:"pullTime"`
	IsLocal  bool      `json:"isLocal"`
	Crc32    uint32    `json:"crc32"`
}

type StatusFileInfo struct {
	Name      string
	Path      string
	Size      int64
	PackageId string
	crc32     uint32
}
