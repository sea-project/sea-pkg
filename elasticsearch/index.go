package elasticsearch

// ResponseCreateIndex 创建索引请求返回信息
type ResponseCreateIndex struct {
	Acknowledged       bool   `json:"acknowledged"`
	ShardsAcknowledged bool   `json:"shards_acknowledged"`
	Index              string `json:"index"`
}

// IndexMapping 索引中mapping信息
type IndexMapping struct {
	Properties interface{} `json:"properties" binding:"required"`
}

// ResponseQueryIndex 查询索引返回信息
type ResponseQueryIndex struct {
	Aliases  interface{}  `json:"aliases"`
	Mappings IndexMapping `json:"mappings" binding:"required"`
	Settings interface{}  `json:"settings"`
}

// ResponseDeleteIndex 删除索引返回信息
type ResponseDeleteIndex struct {
	Acknowledged bool `json:"acknowledged"`
}
