package elasticsearch

type CausedByErr struct {
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

// ErrorInfo 具体业务错误信息
type ErrorInfo struct {
	Type     string      `json:"type"`
	Reason   string      `json:"reason"`
	CausedBy CausedByErr `json:"caused_by"`
}

// ResponseError http请求错误时返回信息
type ResponseError struct {
	Error  ErrorInfo `json:"error" binding:"required"`
	Status int       `json:"status" binding:"required"`
}
