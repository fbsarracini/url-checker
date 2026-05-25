package handler

import (
	"context"

	"github.com/fbsarracini/url-checker/internal/checker"
)

// Checker is defined here, at the consumer, following "accept interfaces, return structs".
type Checker interface {
	Check(ctx context.Context, url string) checker.Result
}
