package models

type ChannelWithRanges struct {
	Channel *TsChannel
	Ranges  []TsBlock
}

type TsChannel struct {
	ChannelNodeId string
	PackageNodeId string
	Name          string
	Start         int64
	End           int64
	Unit          string
	Rate          float64
}

type TsBlock struct {
	BlockNodeId   string
	ChannelNodeId string
	Rate          float64
	Location      string
	StartTime     int64
	EndTime       int64
}
