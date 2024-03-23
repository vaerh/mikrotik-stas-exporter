package mikrotik

import (
	"fmt"
)

var (
	errEmptyPath = fmt.Errorf("the resource path not defined")
)

func Read(resourcePath string, c Client) ([]MikrotikItem, error) {
	if resourcePath == "" {
		return nil, errEmptyPath
	}

	return c.SendRequest(crudRead, &URL{Path: resourcePath})
}

func ReadFiltered(filter []string, resourcePath string, c Client) ([]MikrotikItem, error) {
	if resourcePath == "" {
		return nil, errEmptyPath
	}

	// Filter format: name=value
	// REST query: name=value; name=value
	// API  query: ?=name=value; ?=name=value
	if c.GetTransport() == TransportAPI {
		for i, s := range filter {
			filter[i] = "?=" + s
		}
	}
	return c.SendRequest(crudRead, &URL{Path: resourcePath, Query: filter})
}
