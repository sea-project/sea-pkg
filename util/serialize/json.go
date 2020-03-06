package serialize

import (
	"github.com/json-iterator/go"
	"strings"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func JsonMarshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func JsonUnMarshal(data []byte, v interface{}) error {
	d := json.NewDecoder(strings.NewReader(string(data)))
	d.UseNumber()
	return d.Decode(&v)
}
