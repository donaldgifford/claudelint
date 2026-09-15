package upstream

import (
	"fmt"
	"regexp"
	"strings"
)

// Anchors on the plugin marketplaces page. "Required fields" appears
// under both the marketplace schema and the plugin entry sections, so
// every child anchor is resolved inside its parent section rather than
// against the whole page.
var (
	anchorMarketplaceSchema  = headingAnchor(2, "Marketplace schema")
	anchorPluginEntries      = headingAnchor(2, "Plugin entries")
	anchorPluginSources      = headingAnchor(2, "Plugin sources")
	anchorRequiredFields     = headingAnchor(3, "Required fields")
	anchorOwnerFields        = headingAnchor(3, "Owner fields")
	anchorOptionalFields     = headingAnchor(3, "Optional fields")
	anchorOptionalPluginFlds = headingAnchor(3, "Optional plugin fields")
)

// reservedNamesParagraph matches the sentence that lists the reserved
// marketplace names. The list ends at the first sentence boundary after
// a code span, which keeps the following sentence's examples of blocked
// impersonations out of the set.
var reservedNamesParagraph = regexp.MustCompile(`(?m)^\s*\*\*Reserved names\*\*.*$`)

// colSourceKind and colSourceFields are the columns of the plugin
// source summary table.
const (
	colSourceKind   = "source"
	colSourceFields = "fields"
)

// marketplaceFields reads the top-level marketplace.json fields from
// the required and optional subsections.
type marketplaceFields struct{}

var _ Extractor = marketplaceFields{}

func (marketplaceFields) Section() string        { return "marketplace.fields" }
func (marketplaceFields) Source() string         { return SrcDocsMarketplaces }
func (marketplaceFields) Anchor() *regexp.Regexp { return anchorMarketplaceSchema }

func (marketplaceFields) Extract(section string, out *Digest) error {
	fields, err := requiredAndOptional(section, anchorRequiredFields, anchorOptionalFields)
	if err != nil {
		return err
	}

	out.Marketplace.Fields = fields

	return nonEmpty(fields, "marketplace fields")
}

// marketplaceOwnerFields reads the owner object's fields, which are the
// one marketplace table with its own requiredness column.
type marketplaceOwnerFields struct{}

var _ Extractor = marketplaceOwnerFields{}

func (marketplaceOwnerFields) Section() string        { return "marketplace.owner_fields" }
func (marketplaceOwnerFields) Source() string         { return SrcDocsMarketplaces }
func (marketplaceOwnerFields) Anchor() *regexp.Regexp { return anchorMarketplaceSchema }

func (marketplaceOwnerFields) Extract(section string, out *Digest) error {
	sub, err := subSection(section, anchorOwnerFields)
	if err != nil {
		return err
	}

	t, err := firstTableWithHeaders(sub, colField, colType)
	if err != nil {
		return err
	}

	out.Marketplace.OwnerFields = fieldRows(t, "", RequiredNo)

	return nonEmpty(out.Marketplace.OwnerFields, "marketplace owner fields")
}

// marketplacePluginEntryFields reads the fields of one plugin entry.
type marketplacePluginEntryFields struct{}

var _ Extractor = marketplacePluginEntryFields{}

func (marketplacePluginEntryFields) Section() string        { return "marketplace.plugin_entry_fields" }
func (marketplacePluginEntryFields) Source() string         { return SrcDocsMarketplaces }
func (marketplacePluginEntryFields) Anchor() *regexp.Regexp { return anchorPluginEntries }

func (marketplacePluginEntryFields) Extract(section string, out *Digest) error {
	fields, err := requiredAndOptional(section, anchorRequiredFields, anchorOptionalPluginFlds)
	if err != nil {
		return err
	}

	out.Marketplace.PluginEntryFields = fields

	return nonEmpty(fields, "marketplace plugin entry fields")
}

// marketplaceSources reads the plugin source kinds from the summary
// table, whose fields column marks optional entries with a trailing
// question mark.
//
// Only rows whose first cell is a code span are kinds. The table also
// documents the bare-string relative path form, which is a shape rather
// than a "source" value and has no kind name to record.
type marketplaceSources struct{}

var _ Extractor = marketplaceSources{}

func (marketplaceSources) Section() string        { return "marketplace.sources" }
func (marketplaceSources) Source() string         { return SrcDocsMarketplaces }
func (marketplaceSources) Anchor() *regexp.Regexp { return anchorPluginSources }

func (marketplaceSources) Extract(section string, out *Digest) error {
	t, err := firstTableWithHeaders(section, colSourceKind, colType, colSourceFields)
	if err != nil {
		return err
	}

	kindCol := t.Column(colSourceKind)
	fieldsCol := t.Column(colSourceFields)

	sources := make(map[string]SourceShape, len(t.Rows))

	for i := range t.Rows {
		raw := strings.TrimSpace(t.Cell(i, kindCol))
		if !strings.HasPrefix(raw, "`") {
			continue
		}

		kind := Name(raw)
		if kind == "" {
			continue
		}

		var shape SourceShape
		for _, tok := range BacktickedTokens(t.Cell(i, fieldsCol)) {
			if name, optional := strings.CutSuffix(tok, "?"); optional {
				shape.Optional = append(shape.Optional, name)
			} else {
				shape.Required = append(shape.Required, name)
			}
		}

		sources[kind] = shape
	}

	out.Marketplace.Sources = sources

	return nonEmptyMap(sources, "marketplace plugin source kinds")
}

// marketplaceReservedNames reads the names reserved for Anthropic.
type marketplaceReservedNames struct{}

var _ Extractor = marketplaceReservedNames{}

func (marketplaceReservedNames) Section() string        { return "marketplace.reserved_names" }
func (marketplaceReservedNames) Source() string         { return SrcDocsMarketplaces }
func (marketplaceReservedNames) Anchor() *regexp.Regexp { return anchorMarketplaceSchema }

func (marketplaceReservedNames) Extract(section string, out *Digest) error {
	para := reservedNamesParagraph.FindString(section)
	if para == "" {
		return fmt.Errorf("%w: the **Reserved names** paragraph", ErrAnchorNotFound)
	}

	out.Marketplace.ReservedNames = BacktickedTokens(firstSentence(para))

	return nonEmpty(out.Marketplace.ReservedNames, "reserved marketplace names")
}

// firstSentence truncates at the first sentence boundary that follows a
// code span, which is where the reserved-name list ends and the prose
// about impersonating names begins.
func firstSentence(para string) string {
	if i := strings.Index(para, "`. "); i >= 0 {
		return para[:i+1]
	}

	return para
}

// requiredAndOptional reads a required subsection and an optional one,
// which is how both the marketplace schema and the plugin entry
// sections split their fields.
func requiredAndOptional(section string, required, optional *regexp.Regexp) ([]Field, error) {
	var out []Field

	for _, part := range []struct {
		anchor   *regexp.Regexp
		required string
	}{
		{anchor: required, required: RequiredYes},
		{anchor: optional, required: RequiredNo},
	} {
		sub, err := subSection(section, part.anchor)
		if err != nil {
			return nil, err
		}

		t, err := firstTableWithHeaders(sub, colField, colType)
		if err != nil {
			return nil, err
		}

		rows := fieldRows(t, "", part.required)
		if err := nonEmpty(rows, part.anchor.String()); err != nil {
			return nil, err
		}
		out = append(out, rows...)
	}

	return out, nil
}
