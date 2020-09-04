package elasticsearch

import "encoding/json"

// ResponseAddRecord 添加单挑记录返回信息
type ResponseAddRecord struct {
	ID string `json:"_id" binding:"required"`
}

// Index 每条记录添加返回信息
type Index struct {
	Index  string    `json:"_index"`
	Type   string    `json:"_type"`
	ID     string    `json:"_id"`
	Result string    `json:"result"`
	Status int       `json:"status"`
	Error  ErrorInfo `json:"error"`
}

// ItemInfo 批量添加记录请求返回信息中每条记录添加返回信息
type ItemInfo struct {
	Index Index `json:"index"`
}

// ResponseBatchAddRecord 批量添加记录请求返回信息
type ResponseBatchAddRecord struct {
	Errors bool       `json:"errors"`
	Items  []ItemInfo `json:"items"`
}

// ResponseUpdateRecord 修改记录请求返回信息
type ResponseUpdateRecord struct {
	Total   int `json:"total"`
	Updated int `json:"updated"`
	Deleted int `json:"deleted"`
	Batches int `json:"batches"`
}

// HitsInfo Hits信息
type HitsInfo struct {
	Index  string      `json:"_index"`
	Type   string      `json:"_type"`
	ID     string      `json:"_id"`
	Source interface{} `json:"_source"`
}

// QueryTotalInfo 查询记录请求返回信息中total字段信息
type QueryTotalInfo struct {
	Value    int    `json:"value"`
	Relation string `json:"relation"`
}

// QueryHits 查询记录请求返回信息对象具体信息
type QueryHits struct {
	Total QueryTotalInfo `json:"total"`
	Hits  []HitsInfo     `json:"hits"`
}

// ResponseQueryRecord 查询记录请求返回信息
type ResponseQueryRecord struct {
	Hits QueryHits `json:"hits"`
}

// PerRecordInfo 查询记录请求返回给上层的每个记录信息
type PerRecordInfo struct {
	ID     string      `json:"id"`
	Source interface{} `json:"_source"`
}

// RetQueryRecord 查询记录请求返回给上层的信息
type RetQueryRecord struct {
	TotalValue int              `json:"totalValue"`
	RecordInfo []*PerRecordInfo `json:"recordInfo"`
}

func (c *RetQueryRecord) String() string {
	str, _ := json.Marshal(c)
	return string(str)
}

// ResponseDeleteRecord 删除记录请求响应信息
type ResponseDeleteRecord struct {
	Result string `json:"result"`
}

// IndexInfo 索引名称信息
type IndexInfo struct {
	Index string `json:"_index"`
}

// RequestFromBatchAddRecord 批量添加记录请求Index信息
type RequestFromBatchAddRecord struct {
	Index IndexInfo `json:"index"`
}
