package elasticsearch

import (
	"encoding/json"
	"fmt"
	"github.com/sea-project/sea-pkg/httppool"
)

const ConnTimeOut = 3
const OKCode = 200

// HTTPRequest http请求
func HTTPRequest(method, url, body string, OKCode int, result interface{}) error {
	hp := httppool.NewHttpPool(0, 0, ConnTimeOut)
	header := make(map[string]string)
	header["Content-Type"] = "application/json"
	res, err, statusCode := hp.Request(url, method, body, header)
	if err != nil {
		return fmt.Errorf("HTTPRequest http.NewRequest err:%v", err.Error())
	}

	if statusCode != OKCode {
		errResult := new(ResponseError)
		err = json.Unmarshal([]byte(res), errResult)
		if err != nil {
			return fmt.Errorf("HTTPRequest response data error json.Unmarshal:%v", err.Error())
		}
		return fmt.Errorf("HTTPRequest request res.StatusCode != %v, res.StatusCode=%v, errType:%v, errReason:%v", OKCode, statusCode, errResult.Error.Type, errResult.Error.Reason)
	}
	err = json.Unmarshal([]byte(res), result)
	if err != nil {
		return fmt.Errorf("HTTPRequest json.Unmarshal err:%v", err.Error())
	}
	return nil
}
