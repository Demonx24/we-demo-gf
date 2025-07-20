package boot

import (
	"context"
	"sync"
)

var (
	mu      sync.Mutex
	closers []func(ctx context.Context) error
)

// RegisterCloser 注册一个在程序退出时调用的回调
func RegisterCloser(f func(ctx context.Context) error) {
	mu.Lock()
	defer mu.Unlock()
	closers = append(closers, f)
}

// CloseAll 顺序调用所有注册的回调
func CloseAll(ctx context.Context) {
	mu.Lock()
	defer mu.Unlock()
	for _, f := range closers {
		_ = f(ctx)
	}
}
