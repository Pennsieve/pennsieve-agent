package db

type TimeSeriesChannel struct {
    ID            int // auto-incrementing PK
    NodeId        string
    PackageNodeId string
    Name          string
    Start         int
    End           int
    Unit          string
    Rate          float64
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
