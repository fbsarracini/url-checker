package pool_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fbsarracini/url-checker/internal/checker"
	"github.com/fbsarracini/url-checker/internal/pool"
)

type noopChecker struct{}

func (n *noopChecker) Check(ctx context.Context, url string) checker.Result {
	time.Sleep(1 * time.Millisecond)
	return checker.Result{URL: url, Status: checker.StatusOK}
}

func BenchmarkPool(b *testing.B) {
	for _, size := range []int{10, 100, 1000} {
		size := size
		b.Run(fmt.Sprintf("urls=%d", size), func(b *testing.B) {
			urls := make([]string, size)
			for i := range urls {
				urls[i] = fmt.Sprintf("http://example%d.com", i)
			}
			c := &noopChecker{}
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				pool.Run(context.Background(), urls, c, 50)
			}
		})
	}
}
