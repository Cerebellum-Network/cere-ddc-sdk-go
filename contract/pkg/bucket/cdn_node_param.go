package bucket

import "encoding/json"

// CDNNodeParams Structure-helper for json on the CDN Node Params string
// Public key is hex-encoded
// Default expected key type is Sr25519
type CDNNodeParams struct {
	Url       string `json:"url"`
	Size      uint8  `json:"size"`
	Location  string `json:"location"`
	PublicKey string `json:"publicKey"`
}

func ReadCDNNodeParams(s string) (p CDNNodeParams, err error) {
	err = json.Unmarshal([]byte(s), &p)
	return
}
