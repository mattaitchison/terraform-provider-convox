package convox

import "github.com/convox/rack/pkg/structs"

// ValueGetter gets values (from terraform schema)
type ValueGetter interface {
	Get(key string) interface{}
	GetOk(key string) (interface{}, bool)
}

// ClientUnpacker unpacks the client provided in the meta-data object
type ClientUnpacker func(valueGetter ValueGetter, meta interface{}) (structs.Provider, error)
