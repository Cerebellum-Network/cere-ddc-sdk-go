package sandbox

import (
	"encoding/json"
	"testing"
)

func rulesJSON(rules []Allow) string {
	b, _ := json.Marshal(rules)
	return string(b)
}

// The same cases as the Python SDK's merge_allow and remove_allow, and the
// CLI's `cere sandbox egress allow|remove`.
func TestMergeAllow(t *testing.T) {
	current := []Allow{
		{Host: "api.github.com", Methods: []string{"GET"}, Credential: "gh"},
		{Host: "mcp.slack.com", Methods: []string{"POST"}, Placeholders: []string{"b", "a"}},
	}
	got := mergeAllow(current, []Allow{
		{Host: "API.github.com", Methods: []string{"head", "GET"}, Credential: "gh"}, // same target: methods merge
		{Host: "api.github.com", Methods: []string{"GET"}},                           // no credential: a new rule
		{Host: "mcp.slack.com", Methods: []string{"*"}, Placeholders: []string{"a", "b"}},
		{Host: "pypi.org"},
		{Host: "pypi.org", Methods: []string{"HEAD"}}, // merges into the new pypi rule
	})
	want := `[{"host":"api.github.com","methods":["GET"]},{"host":"pypi.org","methods":["GET","HEAD"]},` +
		`{"host":"api.github.com","methods":["GET","HEAD"],"credential":"gh"},` +
		`{"host":"mcp.slack.com","methods":["*"],"placeholders":["b","a"]}]`
	if rulesJSON(got) != want {
		t.Fatalf("\n got %s\nwant %s", rulesJSON(got), want)
	}
	if rulesJSON(current) != `[{"host":"api.github.com","methods":["GET"],"credential":"gh"},`+
		`{"host":"mcp.slack.com","methods":["POST"],"placeholders":["b","a"]}]` {
		t.Fatal("mergeAllow changed its input")
	}
}

func TestRemoveAllow(t *testing.T) {
	current := []Allow{
		{Host: "a.example", Methods: []string{"GET", "POST"}},
		{Host: "a.example", Methods: []string{"GET"}, PathPrefix: "/x/"},
		{Host: "b.example", Methods: []string{"*"}},
	}
	got, n, err := removeAllow(current, "A.example", []string{"get"})
	if err != nil || n != 2 || rulesJSON(got) != `[{"host":"a.example","methods":["POST"]},{"host":"b.example","methods":["*"]}]` {
		t.Fatalf("%s %d %v", rulesJSON(got), n, err)
	}
	if _, _, err := removeAllow(current, "b.example", []string{"GET"}); err == nil {
		t.Error("removing a method from a * rule: no error")
	}
	got, n, err = removeAllow(current, "b.example", nil)
	if err != nil || n != 1 || len(got) != 2 {
		t.Fatalf("%s %d %v", rulesJSON(got), n, err)
	}
	if _, n, _ := removeAllow(current, "c.example", nil); n != 0 {
		t.Error("a host with no rule changed something")
	}
}
