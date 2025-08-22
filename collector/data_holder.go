package collector

import "sync"

type DataHolder[T any] struct {
	sync.Mutex
	data T
}
