package ddcuri

import (
	"strconv"
	"strings"
)

func (q *DdcQuery) toUri() string {
	parts := []string{"/" + DDC}

	if q.Organization != "" {
		parts = append(parts, ORG, q.Organization)
	}
	// Example output: /ddc/org/my_org

	if q.BucketIdSet {
		bucketId := strconv.Itoa(int(q.BucketId))
		parts = append(parts, BUC, bucketId)
		// Example output: /ddc/org/my_org/buc/123
	} else if q.BucketName != "" {
		parts = append(parts, BUC, q.BucketName)
		// Example output: /ddc/org/my_org/buc/my_bucket
	}

	if q.Protocol == IPIECE || q.Protocol == IFILE {
		parts = append(parts, q.Protocol, q.Cid+q.Extension)
		// Example output: /ddc/org/my_org/buc/my_bucket/ifile/cid123.js
	} else if q.Protocol == PIECE || q.Protocol == FILE {
		parts = append(parts, q.Protocol)
		parts = append(parts, q.Path...)
		// Example output: /ddc/org/my_org/buc/my_bucket/file/folder/image.png
	}

	uri := strings.Join(parts, "/")

	if q.Options != "" {
		uri = uri + "?" + q.Options
	}
	// Example output: /ddc/org/my_org/buc/my_bucket/file/folder/image.png?options
	return uri
}
