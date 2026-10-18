// Package gitops turns approved changes into Git commits and Pull Requests (spec §3.1, §41, §44).
package gitops

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"go.yaml.in/yaml/v3"
)

// ErrClaimNotFound means no YAML document in the file matches the Claim.
var ErrClaimNotFound = errors.New("gitops: claim document not found in manifest")

// Edit is a value to set at a dot-separated path (e.g. spec.parameters.vmSize).
type Edit struct {
	Path  string
	Value any
}

// UpdateManifest applies edits and annotations to the document whose metadata.name is claimName,
// preserving comments, key order and the other documents of the file.
func UpdateManifest(src []byte, claimName string, edits []Edit, annotations map[string]string) ([]byte, error) {
	dec := yaml.NewDecoder(bytes.NewReader(src))
	var docs []*yaml.Node
	for {
		var n yaml.Node
		err := dec.Decode(&n)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("gitops: parse manifest: %w", err)
		}
		docs = append(docs, &n)
	}
	target := -1
	for i, d := range docs {
		if len(docs) == 1 || scalarAt(root(d), "metadata", "name") == claimName {
			target = i
			break
		}
	}
	if target < 0 {
		return nil, ErrClaimNotFound
	}
	doc := root(docs[target])
	if doc == nil || doc.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("gitops: manifest document is not a mapping")
	}
	for _, e := range edits {
		if err := set(doc, strings.Split(e.Path, "."), e.Value); err != nil {
			return nil, fmt.Errorf("gitops: set %s: %w", e.Path, err)
		}
	}
	for k, v := range annotations {
		if err := set(doc, []string{"metadata", "annotations", k}, v); err != nil {
			return nil, fmt.Errorf("gitops: annotation %s: %w", k, err)
		}
	}

	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(2)
	for _, d := range docs {
		if err := enc.Encode(d); err != nil {
			return nil, err
		}
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func root(n *yaml.Node) *yaml.Node {
	if n.Kind == yaml.DocumentNode && len(n.Content) > 0 {
		return n.Content[0]
	}
	return n
}

func scalarAt(n *yaml.Node, path ...string) string {
	cur := n
	for _, p := range path {
		cur = child(cur, p)
		if cur == nil {
			return ""
		}
	}
	return cur.Value
}

func child(m *yaml.Node, key string) *yaml.Node {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

// set writes value at path, creating intermediate mappings. Existing scalar style/comments are kept.
func set(m *yaml.Node, path []string, value any) error {
	for i, key := range path {
		if m.Kind != yaml.MappingNode {
			return fmt.Errorf("%s is not a mapping", strings.Join(path[:i], "."))
		}
		next := child(m, key)
		last := i == len(path)-1
		if next == nil {
			next = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			if last {
				next = &yaml.Node{}
			}
			m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, next)
		}
		if last {
			var enc yaml.Node
			if err := enc.Encode(value); err != nil {
				return err
			}
			head, line := next.HeadComment, next.LineComment
			*next = enc
			next.HeadComment, next.LineComment = head, line
			if s, ok := value.(string); ok && next.Kind == yaml.ScalarNode {
				next.Value, next.Tag, next.Style = s, "!!str", 0
				if s == "" || looksNonString(s) {
					next.Style = yaml.DoubleQuotedStyle
				}
			}
			return nil
		}
		m = next
	}
	return nil
}

// looksNonString reports strings YAML would otherwise read as numbers/booleans/null.
func looksNonString(s string) bool {
	var v any
	if err := yaml.Unmarshal([]byte(s), &v); err != nil {
		return true
	}
	_, isString := v.(string)
	return !isString
}

// ClaimParameters returns spec.parameters of the document named claimName (or of the only document).
func ClaimParameters(manifest []byte, claimName string) (map[string]any, error) {
	dec := yaml.NewDecoder(bytes.NewReader(manifest))
	var docs []map[string]any
	for {
		var d map[string]any
		err := dec.Decode(&d)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("gitops: parse manifest: %w", err)
		}
		if d != nil {
			docs = append(docs, d)
		}
	}
	for _, d := range docs {
		md, _ := d["metadata"].(map[string]any)
		if len(docs) == 1 || md["name"] == claimName {
			spec, _ := d["spec"].(map[string]any)
			params, _ := spec["parameters"].(map[string]any)
			if params == nil {
				params = map[string]any{}
			}
			return params, nil
		}
	}
	return nil, ErrClaimNotFound
}
