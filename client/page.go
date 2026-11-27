package client

import (
	"context"
	"net/url"
	"strconv"
)

// MaxPageSize is the most items the API returns in one page.
const MaxPageSize = 500

// Page is one page of a listing: {"items": [...], "next_cursor": "..." | null}.
type Page[T any] struct {
	Items      []T     `json:"items"`
	NextCursor *string `json:"next_cursor"`
}

// List sends r (a GET of a paged listing) and follows next_cursor until it has
// limit items or the listing ends; limit <= 0 reads the whole listing. A
// next_cursor the listing already gave is an error, not an endless loop.
func List[T any](ctx context.Context, c *Client, r Request, limit int) ([]T, error) {
	q := url.Values{}
	for k, v := range r.Query {
		q[k] = append([]string(nil), v...)
	}
	var out []T
	seen := map[string]bool{}
	for {
		want := MaxPageSize
		if limit > 0 {
			want = min(MaxPageSize, limit-len(out))
		}
		q.Set("limit", strconv.Itoa(want))
		r.Query = q
		var page Page[T]
		if err := c.Do(ctx, &r, &page); err != nil {
			return nil, err
		}
		out = append(out, page.Items...)
		if page.NextCursor == nil || *page.NextCursor == "" || len(page.Items) == 0 || (limit > 0 && len(out) >= limit) {
			if limit > 0 && len(out) > limit {
				out = out[:limit]
			}
			return out, nil
		}
		if seen[*page.NextCursor] {
			return nil, wrapf("%s %s: the listing gave cursor %q a second time", r.Method, r.Path, *page.NextCursor)
		}
		seen[*page.NextCursor] = true
		q.Set("cursor", *page.NextCursor)
	}
}
