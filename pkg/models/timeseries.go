package models

import "time"

type ChannelWithRanges struct {
	Channel *TimeSeriesChannel
	Ranges  []TimeSeriesContinuousRange
}

type TimeSeriesChannel struct {
	ID             int // auto-incrementing PK
	NodeId         string
	PackageId      int64 // package FK
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
	Channel   string
	Rate      float64
	Location  string
	Url       string
	StartTime uint64
	EndTime   uint64
}
