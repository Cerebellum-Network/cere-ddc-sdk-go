package ddcuri

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

func consumeOptions(q *DdcQuery, uri string) string {
	// Example input: /ddc/org/my_org/buc/my_bucket/ifile/my_cid?option=yes
	uri = strings.TrimSpace(uri)
	mainUri, options, _ := strings.Cut(uri, "?")
	q.Options = options
	return mainUri
}

func consumeMain(q *DdcQuery, uri string) error {
	// Example input: /ddc/org/my_org/buc/my_bucket/ifile/my_cid
	if !strings.HasPrefix(uri, DDC_PREFIX) {
		return fmt.Errorf("DDC URI must start with " + DDC_PREFIX)
	}
	uri = uri[len(DDC_PREFIX):]

	// Example input: org/my_org/buc/my_bucket/ifile/my_cid
	parts := strings.Split(uri, "/")
	return consumeOrg(q, parts)
}

func consumeOrg(q *DdcQuery, parts []string) error {
	// Example input: [org my_org buc my_bucket ifile my_cid]
	if len(parts) >= 2 && parts[0] == ORG {
		q.Organization = parts[1]
		parts = parts[2:]
	}

	return consumeBuc(q, parts)
}

func consumeBuc(q *DdcQuery, parts []string) error {
	// Example input: [buc my_bucket ifile my_cid]
	if len(parts) >= 2 && parts[0] == BUC {
		value := parts[1]
		parts = parts[2:]

		if startsWithDigit(value) {
			bucketId, err := strconv.ParseUint(value, 10, 32)
			if err != nil {
				return fmt.Errorf("invalid bucket ID (%s)", value)
			}
			q.BucketId = uint32(bucketId)
			q.BucketIdSet = true
		} else {
			if len(value) == 0 {
				return fmt.Errorf("invalid bucket name (%s)", value)
			}
			q.BucketName = value
		}
	}

	return consumeProtocol(q, parts)
}

func consumeProtocol(q *DdcQuery, parts []string) error {
	// Example input: [ifile my_cid]
	if len(parts) >= 2 {
		field := parts[0]
		if field == IPIECE || field == IFILE {
			q.Protocol = field
			q.Cid, q.Extension = splitExtension(parts[1])
			parts = parts[2:]
		} else if field == PIECE || field == FILE {
			q.Protocol = field
			q.Path = parts[1:]
			_, q.Extension = splitExtension(parts[len(parts)-1])
			parts = nil
		}
	}

	return consumeEnd(q, parts)
}

func consumeEnd(q *DdcQuery, parts []string) error {
	// Example input: []
	if len(parts) != 0 {
		return fmt.Errorf("unrecognized field %s", parts)
	}
	return nil
}

func startsWithDigit(s string) bool {
	first, _ := utf8.DecodeRuneInString(s)
	return unicode.IsDigit(first)
}

func splitExtension(name string) (string, string) {
	ext := filepath.Ext(name)
	if ext != "" {
		// Remove the extension.
		name = name[:len(name)-len(ext)]
	}
	return name, ext
}
