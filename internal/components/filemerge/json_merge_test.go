package filemerge

import (
	"encoding/json"
	"testing"
)

func TestMergeJSONObjectsRecursively(t *testing.T) {
	base := []byte(`{"plugins":["a"],"settings":{"theme":"default","flags":{"x":true}}}`)
	overlay := []byte(`{"settings":{"theme":"gentleman","flags":{"y":true}},"extra":1}`)

	merged, err := MergeJSONObjects(base, overlay)
	if err != nil {
		t.Fatalf("MergeJSONObjects() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(merged, &got); err != nil {
		t.Fatalf("Unmarshal merged json error = %v", err)
	}

	settings := got["settings"].(map[string]any)
	flags := settings["flags"].(map[string]any)

	if settings["theme"] != "gentleman" {
		t.Fatalf("theme = %v", settings["theme"])
	}

	if flags["x"] != true || flags["y"] != true {
		t.Fatalf("flags = %#v", flags)
	}

	plugins := got["plugins"].([]any)
	if len(plugins) != 1 || plugins[0] != "a" {
		t.Fatalf("plugins = %#v", plugins)
	}
}

func TestMergeJSONObjectsSupportsJSONCBase(t *testing.T) {
	base := []byte(`{
	  // VS Code-style comments and trailing commas
	  "editor.fontSize": 14,
	  "files.exclude": {
	    "**/.git": true,
	  },
	}`)
	overlay := []byte(`{"chat.tools.autoApprove": true}`)

	merged, err := MergeJSONObjects(base, overlay)
	if err != nil {
		t.Fatalf("MergeJSONObjects() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(merged, &got); err != nil {
		t.Fatalf("Unmarshal merged json error = %v", err)
	}

	autoApprove, ok := got["chat.tools.autoApprove"].(bool)
	if !ok || !autoApprove {
		t.Fatalf("chat.tools.autoApprove = %#v", got["chat.tools.autoApprove"])
	}

	if got["editor.fontSize"] != float64(14) {
		t.Fatalf("editor.fontSize = %v", got["editor.fontSize"])
	}
}

func TestMergeJSONObjectsMalformedBaseReturnsOverlayOnly(t *testing.T) {
	// Real user machines (e.g. Windows) may have a malformed ~/.cursor/mcp.json.
	// The installer should recover by treating the broken base as {} and continuing.
	tests := []struct {
		name    string
		base    []byte
		overlay []byte
		wantKey string
	}{
		{
			name:    "base starting with letter",
			base:    []byte(`allow: all`),
			overlay: []byte(`{"mcpServers": {"context7": {"type": "remote"}}}`),
			wantKey: "mcpServers",
		},
		{
			name:    "unclosed json object",
			base:    []byte(`{"ok": true`),
			overlay: []byte(`{"chat.tools.autoApprove": true}`),
			wantKey: "chat.tools.autoApprove",
		},
		{
			name:    "arbitrary text",
			base:    []byte(`a`),
			overlay: []byte(`{"servers": {"engram": {"command": "engram"}}}`),
			wantKey: "servers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merged, err := MergeJSONObjects(tt.base, tt.overlay)
			if err != nil {
				t.Fatalf("MergeJSONObjects() error = %v; want nil (malformed base should be treated as {})", err)
			}

			var got map[string]any
			if err := json.Unmarshal(merged, &got); err != nil {
				t.Fatalf("merged result is not valid JSON: %v", err)
			}

			if _, ok := got[tt.wantKey]; !ok {
				t.Fatalf("merged result missing key %q from overlay; got keys: %v", tt.wantKey, got)
			}
		})
	}
}

// ─── __replace__ sentinel tests ───────────────────────────────────────────────

func TestMergeJSONObjectsReplaceSentinelErasesBaseKeys(t *testing.T) {
	base := []byte(`{"mcp":{"engram":{"command":"/opt/homebrew/bin/engram","args":["mcp","--tools=agent"],"type":"local"}}}`)
	overlay := []byte(`{"mcp":{"engram":{"__replace__":{"command":["/opt/homebrew/bin/engram","mcp","--tools=agent"],"type":"local"}}}}`)

	merged, err := MergeJSONObjects(base, overlay)
	if err != nil {
		t.Fatalf("MergeJSONObjects() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(merged, &got); err != nil {
		t.Fatalf("Unmarshal merged error = %v", err)
	}

	mcp := got["mcp"].(map[string]any)
	eng := mcp["engram"].(map[string]any)

	// args must be gone
	if _, ok := eng["args"]; ok {
		t.Fatalf("engram still has 'args' after __replace__; got: %v", eng)
	}
	// command must be an array
	cmd, ok := eng["command"].([]any)
	if !ok {
		t.Fatalf("engram command is not an array; got: %T = %v", eng["command"], eng["command"])
	}
	if len(cmd) != 3 {
		t.Fatalf("engram command has %d elements, want 3", len(cmd))
	}
	// __replace__ must not appear in output
	if _, ok := eng["__replace__"]; ok {
		t.Fatal("__replace__ sentinel leaked into output")
	}
}

func TestMergeJSONObjectsReplaceSentinelNoBaseKey(t *testing.T) {
	base := []byte(`{}`)
	overlay := []byte(`{"a":{"__replace__":{"z":3}}}`)

	merged, err := MergeJSONObjects(base, overlay)
	if err != nil {
		t.Fatalf("MergeJSONObjects() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(merged, &got); err != nil {
		t.Fatalf("Unmarshal merged error = %v", err)
	}

	a, ok := got["a"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'a' to be a map; got: %T = %v", got["a"], got["a"])
	}
	if a["z"] != float64(3) {
		t.Fatalf("a.z = %v, want 3", a["z"])
	}
	if _, ok := a["__replace__"]; ok {
		t.Fatal("__replace__ sentinel leaked into output")
	}
}

func TestMergeJSONObjectsReplaceSentinelPreservesOtherKeys(t *testing.T) {
	base := []byte(`{"a":{"old":1},"b":"keep"}`)
	overlay := []byte(`{"a":{"__replace__":{"new":2}}}`)

	merged, err := MergeJSONObjects(base, overlay)
	if err != nil {
		t.Fatalf("MergeJSONObjects() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(merged, &got); err != nil {
		t.Fatalf("Unmarshal merged error = %v", err)
	}

	// "b" must survive untouched
	if got["b"] != "keep" {
		t.Fatalf("sibling key 'b' lost; got: %v", got)
	}

	a := got["a"].(map[string]any)
	// "old" must be gone (replaced atomically)
	if _, ok := a["old"]; ok {
		t.Fatalf("'a.old' survived __replace__; got: %v", a)
	}
	if a["new"] != float64(2) {
		t.Fatalf("a.new = %v, want 2", a["new"])
	}
}

func TestMergeJSONObjectsReplaceSentinelNotInOutput(t *testing.T) {
	base := []byte(`{"x":{"y":1}}`)
	overlay := []byte(`{"x":{"__replace__":{"z":2}}}`)

	merged, err := MergeJSONObjects(base, overlay)
	if err != nil {
		t.Fatalf("MergeJSONObjects() error = %v", err)
	}

	if !json.Valid(merged) {
		t.Fatalf("merged output is not valid JSON: %s", string(merged))
	}

	var got map[string]any
	if err := json.Unmarshal(merged, &got); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	// Walk all nested maps and ensure __replace__ never appears as a key
	var walk func(m map[string]any)
	walk = func(m map[string]any) {
		for k, v := range m {
			if k == "__replace__" {
				t.Fatalf("sentinel '__replace__' leaked into output: %v", m)
			}
			if sub, ok := v.(map[string]any); ok {
				walk(sub)
			}
		}
	}
	walk(got)
}

func TestMergeJSONObjectsAppendUniquePreservesBaseEntries(t *testing.T) {
	base := []byte(`{"deny":["user-rule-1","stock-b","user-rule-2"]}`)
	overlay := []byte(`{"deny":{"__append_unique__":["stock-a","stock-b"]}}`)

	merged, err := MergeJSONObjects(base, overlay)
	if err != nil {
		t.Fatalf("MergeJSONObjects() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(merged, &got); err != nil {
		t.Fatalf("Unmarshal merged error = %v", err)
	}

	deny, ok := got["deny"].([]any)
	if !ok {
		t.Fatalf("deny is not a list: %#v", got["deny"])
	}

	want := []any{"user-rule-1", "stock-b", "user-rule-2", "stock-a"}
	if len(deny) != len(want) {
		t.Fatalf("deny = %#v, want %#v", deny, want)
	}
	for i := range want {
		if deny[i] != want[i] {
			t.Fatalf("deny[%d] = %v, want %v (full: %#v)", i, deny[i], want[i], deny)
		}
	}
}

func TestMergeJSONObjectsAppendUniqueNoBaseKey(t *testing.T) {
	base := []byte(`{}`)
	overlay := []byte(`{"deny":{"__append_unique__":["a","b","a"]}}`)

	merged, err := MergeJSONObjects(base, overlay)
	if err != nil {
		t.Fatalf("MergeJSONObjects() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(merged, &got); err != nil {
		t.Fatalf("Unmarshal merged error = %v", err)
	}

	deny, ok := got["deny"].([]any)
	if !ok {
		t.Fatalf("deny is not a list: %#v", got["deny"])
	}
	if len(deny) != 2 || deny[0] != "a" || deny[1] != "b" {
		t.Fatalf("deny = %#v, want [a b] (deduped)", deny)
	}
}

func TestMergeJSONObjectsAppendUniqueBaseNotList(t *testing.T) {
	base := []byte(`{"deny":"corrupted"}`)
	overlay := []byte(`{"deny":{"__append_unique__":["a"]}}`)

	merged, err := MergeJSONObjects(base, overlay)
	if err != nil {
		t.Fatalf("MergeJSONObjects() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(merged, &got); err != nil {
		t.Fatalf("Unmarshal merged error = %v", err)
	}

	deny, ok := got["deny"].([]any)
	if !ok {
		t.Fatalf("deny is not a list: %#v", got["deny"])
	}
	if len(deny) != 1 || deny[0] != "a" {
		t.Fatalf("deny = %#v, want [a]", deny)
	}
}

func TestMergeJSONObjectsAppendUniqueIsIdempotent(t *testing.T) {
	base := []byte(`{"deny":["user-rule"]}`)
	overlay := []byte(`{"deny":{"__append_unique__":["stock-a","stock-b"]}}`)

	once, err := MergeJSONObjects(base, overlay)
	if err != nil {
		t.Fatalf("MergeJSONObjects() first error = %v", err)
	}

	twice, err := MergeJSONObjects(once, overlay)
	if err != nil {
		t.Fatalf("MergeJSONObjects() second error = %v", err)
	}

	if string(once) != string(twice) {
		t.Fatalf("second merge changed output:\nfirst:  %s\nsecond: %s", once, twice)
	}
}

func TestMergeJSONObjectsAppendUniqueSentinelNotInOutput(t *testing.T) {
	base := []byte(`{}`)
	overlay := []byte(`{"nested":{"deny":{"__append_unique__":["a"]}}}`)

	merged, err := MergeJSONObjects(base, overlay)
	if err != nil {
		t.Fatalf("MergeJSONObjects() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(merged, &got); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	var walk func(m map[string]any)
	walk = func(m map[string]any) {
		for k, v := range m {
			if k == "__append_unique__" {
				t.Fatalf("sentinel '__append_unique__' leaked into output: %v", m)
			}
			if sub, ok := v.(map[string]any); ok {
				walk(sub)
			}
		}
	}
	walk(got)

	nested, ok := got["nested"].(map[string]any)
	if !ok {
		t.Fatalf("nested missing: %#v", got)
	}
	deny, ok := nested["deny"].([]any)
	if !ok || len(deny) != 1 || deny[0] != "a" {
		t.Fatalf("nested.deny = %#v, want [a]", nested["deny"])
	}
}
