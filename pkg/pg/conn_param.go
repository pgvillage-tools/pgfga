package pg

import (
	"fmt"
	"strings"
)

// connectStringValue uses proper quoting for connect string values
func connectStringValue(objectName string) (escaped string) {
	return fmt.Sprintf("'%s'", strings.ReplaceAll(objectName, "'", "\\'"))
}

type ConnParamKey string

const (
	ConnParamDBName ConnParamKey = "dbname"
)

// ConnParams can hold all connection parameters as key, value pairs
type ConnParams map[ConnParamKey]string

// String joins all Connection Parameters into a connection string
func (dsn ConnParams) String() string {
	var pairs []string
	for key, value := range dsn {
		pairs = append(pairs, fmt.Sprintf("%s=%s", key, connectStringValue(value)))
	}
	return strings.Join(pairs[:], " ")
}

// Clone returns a copy of this ConnParams
func (dsn ConnParams) Clone() ConnParams {
	clone := ConnParams{}
	for key, value := range dsn {
		clone[key] = value
	}
	return clone
}
