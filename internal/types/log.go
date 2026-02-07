package types

// SimpleLog 简洁日志条目
type SimpleLog struct {
	Date    string `json:"date"`    // 日期，格式：2006/1/2
	Time    string `json:"time"`    // 时间，格式：15:04:05
	Message string `json:"message"` // 日志内容
}

// LogQuery 日志查询参数
type LogQuery struct {
	Keyword string `json:"keyword"` // 关键词搜索
	Limit   int    `json:"limit"`   // 返回条数，默认100
}
