package pg

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// connectStringValue uses proper quoting for connect string values
func connectStringValue(objectName string) (escaped string) {
	return fmt.Sprintf("'%s'", strings.ReplaceAll(objectName, "'", "\\'"))
}

// ConnParamKey represents the key of a connection string parameter
type ConnParamKey string

// In a connect string, the dbname key point to the database name
const (
	ConnParamDBName ConnParamKey = "dbname"
)

// ConnParams can hold all connection parameters as key, value pairs
type ConnParams map[ConnParamKey]string

// String joins all Connection Parameters into a connection string
func (dsn ConnParams) String() string {
	var pairs []string
	dsnSortedKeys := slices.Sorted(maps.Keys(dsn))
	for _, key := range dsnSortedKeys {
		pairs = append(pairs,
			fmt.Sprintf(
				"%s=%s",
				key,
				connectStringValue(dsn[key]),
			),
		)
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
