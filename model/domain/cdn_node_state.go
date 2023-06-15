package domain

import (
	"encoding/json"
	"errors"
	"fmt"
)

type NodeState int32

var ErrNodeStateInvalid = errors.New("node status is invalid")

const (
	NA NodeState = iota
	Grey
	Green
	Blue
	Red
)

var defNodeStateNameToValue = map[string]NodeState{
	"NA":    NA,
	"Grey":  Grey,
	"Green": Green,
	"Blue":  Blue,
	"Red":   Red,
}

var defNodeStateValueToName = map[NodeState]string{
	NA:    "NA",
	Grey:  "Grey",
	Green: "Green",
	Blue:  "Blue",
	Red:   "Red",
}

func (r *NodeState) String() string {
	s, ok := defNodeStateValueToName[*r]
	if !ok {
		return "Invalid"
	}
	return s
}

func (r *NodeState) Validate() error {
	_, ok := defNodeStateValueToName[*r]
	if !ok {
		return ErrNodeStateInvalid
	}
	return nil
}
func (r *NodeState) MarshalJSON() ([]byte, error) {
	s, ok := defNodeStateValueToName[*r]
	if !ok {
		return nil, fmt.Errorf("node state (%d) is invalid value", r)
	}
	return json.Marshal(s)
}

func (r *NodeState) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("node state should be a string, got %s", string(data))
	}
	v, ok := defNodeStateNameToValue[s]
	if !ok {
		return fmt.Errorf("node state(%q) is invalid value", s)
	}
	*r = v
	return nil
}
