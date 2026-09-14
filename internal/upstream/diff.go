package upstream

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
)

// The diff is structural over the digest tree rather than textual over
// the digest file.
//
// A textual diff of sorted JSON is readable enough for a human, but the
// report has two other readers: the exit code, which has to know whether
// anything that matters changed, and the issue body, which has to name
// the constant a maintainer will go and edit. Diffing the tree gives
// both. A hook event that appears upstream is one line naming
// hooks.events and the event, not six lines of context and brackets.
//
// Three sections are partitioned into an informational block that never
// affects the exit code: meta, which tracks upstream's own version
// markers; schemastore, which is the cross-check tier and loses to the
// documentation; and disagreements, which is derived from the other two
// (DESIGN-0006 §5, OQ9).

// Change kinds.
const (
	ChangeAdded   = "added"
	ChangeRemoved = "removed"
	ChangeChanged = "changed"
)

// informationalSections are the top-level digest sections whose changes
// are reported but never counted as drift.
var informationalSections = map[string]struct{}{
	"disagreements": {},
	"meta":          {},
	"schemastore":   {},
}

// Change is one difference between two digests.
//
// Path is the dotted path of the container that changed and Item names
// what changed inside it, so a report groups by Path and a reader goes
// straight from a line to the constant behind it.
type Change struct {
	// Path is the dotted digest path, such as "hooks.events".
	Path string `json:"path"`
	// Kind is added, removed, or changed.
	Kind string `json:"kind"`
	// Item is the set member, map key, or "<row>.<field>" that differs.
	Item string `json:"item"`
	// Old is the previous value, set on changed and on a removed scalar.
	Old any `json:"old,omitempty"`
	// New is the current value, set on changed and on an added scalar.
	New any `json:"new,omitempty"`
}

// Report is the outcome of comparing two digests.
type Report struct {
	// Changes is the drift: everything outside the informational
	// sections. A non-empty Changes is what makes the tool exit 1.
	Changes []Change `json:"changes"`
	// Informational is the changes under meta, schemastore, and
	// disagreements.
	Informational []Change `json:"informational"`
	// SourcesChanged is the source ids whose content hash moved, from
	// the lock. It is set by the caller, which is the only layer that
	// has both locks.
	SourcesChanged []string `json:"sources_changed,omitempty"`
}

// HasDrift reports whether anything outside the informational sections
// changed.
func (r *Report) HasDrift() bool { return len(r.Changes) > 0 }

// Empty reports whether the two digests were identical.
func (r *Report) Empty() bool { return len(r.Changes) == 0 && len(r.Informational) == 0 }

// Diff compares two digests. The result is deterministic: paths are
// visited in sorted order and set members are reported sorted.
func Diff(before, after *Digest) (*Report, error) {
	beforeTree, err := before.Tree()
	if err != nil {
		return nil, err
	}
	afterTree, err := after.Tree()
	if err != nil {
		return nil, err
	}

	var all []Change
	diffMap("", beforeTree, afterTree, &all)

	// Both blocks are non-nil so the JSON report always carries arrays.
	// A consumer piping this through jq should not have to distinguish
	// "no changes" from "null".
	report := &Report{Changes: []Change{}, Informational: []Change{}}
	for i := range all {
		if all[i].informational() {
			report.Informational = append(report.Informational, all[i])

			continue
		}
		report.Changes = append(report.Changes, all[i])
	}

	return report, nil
}

// informational reports whether a change belongs to a section that never
// counts as drift. The section is the first path segment, or the item
// itself for a change at the root.
func (c *Change) informational() bool {
	section := c.Item
	if c.Path != "" {
		section, _, _ = strings.Cut(c.Path, ".")
	}

	_, ok := informationalSections[section]

	return ok
}

// diffMap walks two objects, recursing into nested objects and arrays
// and reporting scalar leaves against the path of their container.
func diffMap(path string, before, after map[string]any, out *[]Change) {
	for _, key := range sortedUnion(before, after) {
		oldValue, inBefore := before[key]
		newValue, inAfter := after[key]

		switch {
		case !inAfter:
			*out = append(*out, Change{Path: path, Kind: ChangeRemoved, Item: key, Old: scalarOrNil(oldValue)})
		case !inBefore:
			*out = append(*out, Change{Path: path, Kind: ChangeAdded, Item: key, New: scalarOrNil(newValue)})
		default:
			diffValue(joinPath(path, key), path, key, oldValue, newValue, out)
		}
	}
}

// diffValue dispatches on the shape of a pair of values that are present
// on both sides.
func diffValue(childPath, path, key string, before, after any, out *[]Change) {
	beforeMap, beforeIsMap := before.(map[string]any)
	afterMap, afterIsMap := after.(map[string]any)
	if beforeIsMap && afterIsMap {
		diffMap(childPath, beforeMap, afterMap, out)

		return
	}

	beforeArr, beforeIsArr := before.([]any)
	afterArr, afterIsArr := after.([]any)
	if beforeIsArr && afterIsArr {
		diffArray(childPath, beforeArr, afterArr, out)

		return
	}

	if leafText(before) != leafText(after) {
		*out = append(*out, Change{Path: path, Kind: ChangeChanged, Item: key, Old: before, New: after})
	}
}

// diffArray diffs an array as a set of strings, or by row identity when
// it holds objects.
func diffArray(path string, before, after []any, out *[]Change) {
	if holdsObjects(before) || holdsObjects(after) {
		diffRows(path, before, after, out)

		return
	}

	diffSet(path, before, after, out)
}

// diffSet reports the members added to and removed from a string set.
func diffSet(path string, before, after []any, out *[]Change) {
	beforeSet := textSet(before)
	afterSet := textSet(after)

	for _, item := range slices.Sorted(maps.Keys(afterSet)) {
		if _, ok := beforeSet[item]; !ok {
			*out = append(*out, Change{Path: path, Kind: ChangeAdded, Item: item})
		}
	}

	for _, item := range slices.Sorted(maps.Keys(beforeSet)) {
		if _, ok := afterSet[item]; !ok {
			*out = append(*out, Change{Path: path, Kind: ChangeRemoved, Item: item})
		}
	}
}

// diffRows diffs an array of objects by row identity: a row present on
// one side only is added or removed, and a row on both sides reports one
// change per differing key.
func diffRows(path string, before, after []any, out *[]Change) {
	beforeRows := indexRows(before)
	afterRows := indexRows(after)

	for _, id := range sortedUnion(beforeRows, afterRows) {
		oldRow, inBefore := beforeRows[id]
		newRow, inAfter := afterRows[id]

		switch {
		case !inAfter:
			*out = append(*out, Change{Path: path, Kind: ChangeRemoved, Item: id})
		case !inBefore:
			*out = append(*out, Change{Path: path, Kind: ChangeAdded, Item: id})
		default:
			diffRowFields(path, id, oldRow, newRow, out)
		}
	}
}

// diffRowFields reports one change per key that differs between two
// versions of the same row.
func diffRowFields(path, id string, before, after map[string]any, out *[]Change) {
	for _, key := range sortedUnion(before, after) {
		oldValue := before[key]
		newValue := after[key]

		if leafText(oldValue) == leafText(newValue) {
			continue
		}

		*out = append(*out, Change{
			Path: path,
			Kind: ChangeChanged,
			Item: id + "." + key,
			Old:  scalarOrNil(oldValue),
			New:  scalarOrNil(newValue),
		})
	}
}

// rowIdentityKeys and rowQualifierKeys are the object keys that identify
// a row, in priority order. The qualifier is only used when the primary
// key repeats: the plugin manifest tables list a field name under more
// than one group, and the disagreement list carries a topic once per
// cross-check source.
var (
	rowIdentityKeys  = []string{"name", "topic"}
	rowQualifierKeys = []string{"group", "source"}
)

// indexRows keys an array of objects by identity, disambiguating a
// repeated identity with its qualifier so no row is silently dropped.
func indexRows(arr []any) map[string]map[string]any {
	rows := make([]map[string]any, 0, len(arr))
	counts := make(map[string]int, len(arr))

	for _, item := range arr {
		obj, ok := item.(map[string]any)
		if !ok {
			continue
		}
		rows = append(rows, obj)
		counts[rowIdentity(obj)]++
	}

	out := make(map[string]map[string]any, len(rows))
	for _, obj := range rows {
		id := rowIdentity(obj)
		if counts[id] > 1 {
			id = fmt.Sprintf("%s (%s)", id, rowQualifier(obj))
		}
		out[id] = obj
	}

	return out
}

// rowIdentity returns the display identity of one row.
func rowIdentity(obj map[string]any) string {
	if id := firstString(obj, rowIdentityKeys); id != "" {
		return id
	}

	return leafText(obj)
}

// rowQualifier returns the value that distinguishes rows sharing an
// identity.
func rowQualifier(obj map[string]any) string {
	if q := firstString(obj, rowQualifierKeys); q != "" {
		return q
	}

	return "?"
}

// firstString returns the first non-empty string value among keys.
func firstString(obj map[string]any, keys []string) string {
	for _, k := range keys {
		if s, ok := obj[k].(string); ok && s != "" {
			return s
		}
	}

	return ""
}

// holdsObjects reports whether the first element of an array is an
// object. The digest never mixes element types within one array.
func holdsObjects(arr []any) bool {
	if len(arr) == 0 {
		return false
	}
	_, ok := arr[0].(map[string]any)

	return ok
}

// textSet renders an array as a membership set.
func textSet(arr []any) map[string]struct{} {
	out := make(map[string]struct{}, len(arr))
	for _, item := range arr {
		out[leafText(item)] = struct{}{}
	}

	return out
}

// sortedUnion returns the sorted union of two maps' keys.
func sortedUnion[V any](a, b map[string]V) []string {
	seen := make(map[string]struct{}, len(a)+len(b))
	for k := range a {
		seen[k] = struct{}{}
	}
	for k := range b {
		seen[k] = struct{}{}
	}

	return slices.Sorted(maps.Keys(seen))
}

// joinPath appends a segment to a dotted path.
func joinPath(path, segment string) string {
	if path == "" {
		return segment
	}

	return path + "." + segment
}

// scalarOrNil returns a value only if it is a scalar. A whole subtree in
// an Old or New field would swamp the report, and its contents are
// already visible in the digest.
func scalarOrNil(v any) any {
	switch v.(type) {
	case map[string]any, []any:
		return nil
	default:
		return v
	}
}

// leafText renders a value for comparison and display. Numbers keep the
// digest's own formatting because the tree decodes them as json.Number.
func leafText(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case json.Number:
		return t.String()
	case bool:
		return strconv.FormatBool(t)
	}

	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}

	return string(raw)
}

// Text renders the report as one line per change, which is what a CI log
// and a terminal both want.
func (r *Report) Text() string {
	if r.Empty() {
		return r.textNoDrift()
	}

	var b strings.Builder

	if len(r.Changes) == 0 {
		b.WriteString("no drift: only informational sections changed\n\n")
	}
	for i := range r.Changes {
		b.WriteString(r.Changes[i].line() + "\n")
	}

	if len(r.Informational) > 0 {
		if len(r.Changes) > 0 {
			b.WriteString("\n")
		}
		b.WriteString("informational:\n")
		for i := range r.Informational {
			b.WriteString("  " + r.Informational[i].line() + "\n")
		}
	}

	return b.String()
}

// textNoDrift is the clean-run output, which still names the sources
// that moved so a reader knows the tool looked.
func (r *Report) textNoDrift() string {
	if len(r.SourcesChanged) == 0 {
		return "no drift\n"
	}

	return fmt.Sprintf("no drift (%s changed without affecting the digest)\n",
		strings.Join(r.SourcesChanged, ", "))
}

// line renders one change for the text report.
func (c *Change) line() string {
	prefix := map[string]string{ChangeAdded: "+", ChangeRemoved: "-", ChangeChanged: "~"}[c.Kind]

	head := fmt.Sprintf("%s %s: %s", prefix, c.pathOrRoot(), c.Item)
	if c.Kind != ChangeChanged {
		return head
	}

	return fmt.Sprintf("%s: %s -> %s", head, leafText(c.Old), leafText(c.New))
}

// pathOrRoot names the container a change belongs to, including the
// digest root.
func (c *Change) pathOrRoot() string {
	if c.Path == "" {
		return "digest"
	}

	return c.Path
}

// JSON renders the report with the canonical encoding, so a golden test
// compares bytes.
func (r *Report) JSON() ([]byte, error) {
	var buf bytes.Buffer
	if err := encodeJSON(&buf, r); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// markdownMaxItems caps how many set members one line lists. A tool
// reference that gains thirty entries should not push the rest of the
// report out of an issue body; the full set is in the digest.
const markdownMaxItems = 12

// Markdown renders the report for an issue body or a step summary,
// grouped by digest path with one line per kind.
func (r *Report) Markdown() string {
	var b strings.Builder

	switch {
	case r.Empty():
		b.WriteString("No drift: the committed digest matches upstream.\n")
	case len(r.Changes) == 0:
		b.WriteString("No drift: only the informational sections changed.\n")
	default:
		writeMarkdownGroups(&b, r.Changes)
	}

	if len(r.Informational) > 0 {
		b.WriteString("\n## Informational\n\nThese never count as drift.\n\n")
		writeMarkdownGroups(&b, r.Informational)
	}

	r.writeMarkdownSources(&b)

	return b.String()
}

// writeMarkdownSources appends the lock diff, which is the only part of
// the report that says anything when the digest is unchanged.
func (r *Report) writeMarkdownSources(b *strings.Builder) {
	if len(r.SourcesChanged) == 0 {
		return
	}

	heading := "Sources changed"
	if r.Empty() {
		heading = "Sources changed without affecting the digest"
	}

	if !strings.HasSuffix(b.String(), "\n\n") {
		b.WriteString("\n")
	}
	fmt.Fprintf(b, "## %s\n\n", heading)
	for _, id := range r.SourcesChanged {
		fmt.Fprintf(b, "- `%s`\n", id)
	}
}

// writeMarkdownGroups writes one section per digest path.
func writeMarkdownGroups(b *strings.Builder, changes []Change) {
	for _, path := range groupPaths(changes) {
		fmt.Fprintf(b, "### %s\n\n", path)

		for _, kind := range []string{ChangeAdded, ChangeRemoved, ChangeChanged} {
			items := itemsOf(changes, path, kind)
			if len(items) == 0 {
				continue
			}
			fmt.Fprintf(b, "- %s: %s\n", kind, joinItems(items))
		}

		b.WriteString("\n")
	}
}

// groupPaths returns the distinct paths in the order they first appear,
// which is the sorted order Diff produced.
func groupPaths(changes []Change) []string {
	var (
		paths []string
		seen  = make(map[string]struct{}, len(changes))
	)

	for i := range changes {
		p := changes[i].pathOrRoot()
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		paths = append(paths, p)
	}

	return paths
}

// itemsOf renders the items of one path and kind.
func itemsOf(changes []Change, path, kind string) []string {
	var out []string

	for i := range changes {
		c := &changes[i]
		if c.pathOrRoot() != path || c.Kind != kind {
			continue
		}

		item := fmt.Sprintf("`%s`", c.Item)
		if kind == ChangeChanged {
			item = fmt.Sprintf("`%s`: %s -> %s", c.Item, leafText(c.Old), leafText(c.New))
		}
		out = append(out, item)
	}

	return out
}

// joinItems renders a list, truncating a long one with its total.
func joinItems(items []string) string {
	if len(items) <= markdownMaxItems {
		return strings.Join(items, ", ")
	}

	return fmt.Sprintf("%s, ... (%d)", strings.Join(items[:markdownMaxItems], ", "), len(items))
}
