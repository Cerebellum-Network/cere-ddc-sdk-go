package ddcuri

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGoodDdcUri(t *testing.T) {
	goodDdcUri(t,
		"/ddc/buc/123/ipiece/cid123",
		DdcQuery{
			Protocol:    "ipiece",
			BucketId:    123,
			BucketIdSet: true,
			Cid:         "cid123",
		})

	goodDdcUri(t,
		"  /ddc/buc/123/ipiece/cid123   ",
		DdcQuery{
			Protocol:    "ipiece",
			BucketId:    123,
			BucketIdSet: true,
			Cid:         "cid123",
		})

	goodDdcUri(t,
		"ddc/buc/123/ifile/cid123",
		DdcQuery{
			Protocol:    "ifile",
			BucketId:    123,
			BucketIdSet: true,
			Cid:         "cid123",
		})

	goodDdcUri(t,
		"ddc/org/my_org/buc/my_bucket/ifile/cid123",
		DdcQuery{
			Organization: "my_org",
			BucketName:   "my_bucket",
			Protocol:     "ifile",
			Cid:          "cid123",
		})

	goodDdcUri(t,
		"/ddc/org/my_org/buc/my_bucket/ifile/cid123?option=yes",
		DdcQuery{
			Organization: "my_org",
			BucketName:   "my_bucket",
			Protocol:     "ifile",
			Cid:          "cid123",
			Options:      "option=yes",
		})

	goodDdcUri(t,
		"/ddc/org/my_org/buc/my_bucket/file/my_folder/image.png?option=yes",
		DdcQuery{
			Organization: "my_org",
			BucketName:   "my_bucket",
			Protocol:     "file",
			Path:         []string{"my_folder", "image.png"},
			Options:      "option=yes",
		})

	goodDdcUri(t,
		"/ddc/org/my_org/buc/my_bucket/file/",
		DdcQuery{
			Organization: "my_org",
			BucketName:   "my_bucket",
			Protocol:     "file",
			Path:         []string{""},
		})
}

func TestBadDdcUri(t *testing.T) {
	badDdcUri(t, "", "DDC URI must start with /ddc/")
	badDdcUri(t, "http://something", "DDC URI must start with /ddc/")
	badDdcUri(t, "/ddc/org/my_org/buc//ifile/cid123", "invalid bucket name ()")
	badDdcUri(t, "/ddc/org/my_org/buc/?my_bucket/ifile/cid123", "invalid bucket name ()")
	badDdcUri(t, "/ddc/buc/5000000000/ipiece/too_big", "invalid bucket ID (5000000000)")
	badDdcUri(t, "/ddc/org/my_org/buc/my_bucket/file", "unrecognized field [file]")
}

func goodDdcUri(t *testing.T, uri string, expected DdcQuery) {
	parsed, err := Parse(uri)

	assert.NoError(t, err)
	assert.Equal(t, expected, parsed)
}

func badDdcUri(t *testing.T, uri string, errMsg string) {
	_, err := Parse(uri)
	if assert.Error(t, err) {
		assert.Equal(t, err.Error(), errMsg)
	}
}
