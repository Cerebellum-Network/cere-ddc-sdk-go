package bucket

import (
	"encoding/json"
)

// Structure-helper for json on the CDN Node Params string
type CDNNodeParams struct {
	Url      string  `json:"url"`
	Size     FlexInt `json:"size"`
	Location string  `json:"location"`
}

func ReadCDNNodeParams(s string) (p CDNNodeParams, err error) {
	err = json.Unmarshal([]byte(s), &p)
	return
}
