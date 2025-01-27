package models

import "time"

type ChannelWithRanges struct {
	Channel *TimeSeriesChannel
	Ranges  []TimeSeriesContinuousRange
}

type TimeSeriesChannel struct {
	ID             int // auto-incrementing PK
	NodeId         string
	PackageNodeId  string
	Name           string
	Start          int
	End            int
	Unit           string
	Rate           float64
	Type           string
	Group          string
	LastAnnotation int
	Properties     []map[string]interface{}
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type TimeSeriesContinuousRange struct {
	ID        int // auto-incrementing PK
	NodeId    string
	Channel   string
	Rate      float64
	Location  string
	StartTime uint64
	EndTime   uint64
}
