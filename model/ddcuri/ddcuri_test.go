package ddcuri

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGoodDdcUri(t *testing.T) {
	goodDdcUri(t,
		"/ddc/buc/123/ipiece/cid123",
		"",
		DdcQuery{
			Protocol:    "ipiece",
			BucketId:    123,
			BucketIdSet: true,
			Cid:         "cid123",
		})

	goodDdcUri(t,
		"  /ddc/buc/123/ipiece/cid123   ",
		"/ddc/buc/123/ipiece/cid123", // canonical
		DdcQuery{
			Protocol:    "ipiece",
			BucketId:    123,
			BucketIdSet: true,
			Cid:         "cid123",
		})

	goodDdcUri(t,
		"ddc/buc/123/ifile/cid123",
		"/ddc/buc/123/ifile/cid123", // canonical
		DdcQuery{
			Protocol:    "ifile",
			BucketId:    123,
			BucketIdSet: true,
			Cid:         "cid123",
		})

	goodDdcUri(t,
		"ddc/org/my_org/buc/my_bucket/ifile/cid123",
		"/ddc/org/my_org/buc/my_bucket/ifile/cid123", // canonical
		DdcQuery{
			Organization: "my_org",
			BucketName:   "my_bucket",
			Protocol:     "ifile",
			Cid:          "cid123",
		})

	goodDdcUri(t,
		"/ddc/org/my_org/buc/my_bucket/ifile/cid123?option=yes",
		"",
		DdcQuery{
			Organization: "my_org",
			BucketName:   "my_bucket",
			Protocol:     "ifile",
			Cid:          "cid123",
			Options:      "option=yes",
		})

	goodDdcUri(t,
		"/ddc/org/my_org/buc/my_bucket/file/my_folder/image.png?option=yes",
		"",
		DdcQuery{
			Organization: "my_org",
			BucketName:   "my_bucket",
			Protocol:     "file",
			Path:         []string{"my_folder", "image.png"},
			Options:      "option=yes",
		})

	goodDdcUri(t,
		"/ddc/org/my_org/buc/my_bucket/file/",
		"",
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

func goodDdcUri(t *testing.T, inputUri string, canonicalUri string, expected DdcQuery) {
	parsed, err := Parse(inputUri)
	assert.NoError(t, err)
	assert.Equal(t, expected, parsed)

	rebuilt := expected.String()
	if canonicalUri == "" {
		canonicalUri = inputUri
	}
	assert.Equal(t, rebuilt, canonicalUri)
}

func badDdcUri(t *testing.T, uri string, errMsg string) {
	_, err := Parse(uri)
	assert.EqualError(t, err, errMsg)
}

func TestGoodWebUrl(t *testing.T) {
	uri := "/ddc/buc/123/ipiece/cid123?options"
	parsed1, err1 := Parse(uri)
	parsed2, err2 := ParseWebUrl(uri)
	parsed3, err3 := ParseWebUrl("htts://cdn" + uri)
	assert.Equal(t, parsed1, parsed2)
	assert.Equal(t, parsed1, parsed3)
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)
}

func TestBadWebUrl(t *testing.T) {
	uri := "/ddc/xyz"
	parsed1, err1 := Parse(uri)
	parsed2, err2 := ParseWebUrl("htts://cdn" + uri)
	assert.Equal(t, parsed1, parsed2)
	assert.Equal(t, err1, err2)
	assert.EqualError(t, err2, "unrecognized field [xyz]")

	_, err := ParseWebUrl("")
	if assert.Error(t, err) {
		assert.Equal(t, err.Error(), "not a DDC URL ()")
	}

	_, err = ParseWebUrl("htts://cdn-ddc/buc/123")
	if assert.Error(t, err) {
		assert.Equal(t, err.Error(), "not a DDC URL (htts://cdn-ddc/buc/123)")
	}
}
