package pool

import (
	"context"
	"errors"

	"golang.org/x/sync/errgroup"

	"github.com/fbsarracini/url-checker/internal/checker"
)

type Checker interface {
	Check(ctx context.Context, url string) checker.Result
}

func Run(ctx context.Context, urls []string, c Checker, limit int) []checker.Result {
	results := make([]checker.Result, len(urls))

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(limit)

	for i, url := range urls {
		g.Go(func() error {
			select {
			case <-gctx.Done():
				status := checker.StatusCancelled
				if errors.Is(gctx.Err(), context.DeadlineExceeded) {
					status = checker.StatusTimeout
				}
				results[i] = checker.Result{URL: url, Status: status, Error: gctx.Err().Error()}
				return nil
			default:
			}
			results[i] = c.Check(gctx, url)
			return nil
		})
	}

	g.Wait() //nolint:errcheck
	return results
}
