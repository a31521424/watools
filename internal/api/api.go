package api

import (
	"sync"
)

var (
	waApiInstance *WaApi
	waApiOnce     sync.Once
)

type WaApi struct {
}

func GetWaApi() *WaApi {
	waApiOnce.Do(func() {
		waApiInstance = &WaApi{}
	})
	return waApiInstance
}
