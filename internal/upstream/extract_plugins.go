package upstream

import (
	"fmt"
	"regexp"
	"strings"
)

// Anchors on the plugins reference page.
var (
	anchorPluginManifest      = headingAnchor(2, "Plugin manifest schema")
	anchorPluginLocations     = headingAnchor(3, "File locations reference")
	anchorPluginEnvironment   = headingAnchor(3, "Environment variables")
	anchorPluginRequired      = headingAnchor(3, "Required fields")
	anchorPluginMetadata      = headingAnchor(3, "Metadata fields")
	anchorPluginComponentPath = headingAnchor(3, "Component path fields")
)

// manifestGroups maps each subsection of the manifest schema to the
// group its rows belong to and the requiredness that follows from the
// heading, since those tables carry no requiredness column.
var manifestGroups = []struct {
	anchor   *regexp.Regexp
	group    string
	required string
}{
	{anchor: anchorPluginRequired, group: groupRequired, required: RequiredYes},
	{anchor: anchorPluginMetadata, group: groupMetadata, required: RequiredNo},
	{anchor: anchorPluginComponentPath, group: groupComponent, required: RequiredNo},
}

// colSubstitutionComponent and colSubstitutionFields are the columns of
// the placeholder-resolution table.
const (
	colSubstitutionComponent = "plugin component"
	colSubstitutionFields    = "fields where placeholders resolve"
)

// pluginsManifestFields reads the three field tables of the plugin
// manifest schema, tagging each row with the subsection it came from.
type pluginsManifestFields struct{}

var _ Extractor = pluginsManifestFields{}

func (pluginsManifestFields) Section() string        { return "plugins.manifest_fields" }
func (pluginsManifestFields) Source() string         { return SrcDocsPlugins }
func (pluginsManifestFields) Anchor() *regexp.Regexp { return anchorPluginManifest }

func (pluginsManifestFields) Extract(section string, out *Digest) error {
	var fields []Field

	for _, g := range manifestGroups {
		sub, err := subSection(section, g.anchor)
		if err != nil {
			return err
		}

		t, err := firstTableWithHeaders(sub, colField, colType)
		if err != nil {
			return fmt.Errorf("%s fields: %w", g.group, err)
		}

		rows := fieldRows(t, g.group, g.required)
		if err := nonEmpty(rows, g.group+" manifest fields"); err != nil {
			return err
		}
		fields = append(fields, rows...)
	}

	out.Plugins.ManifestFields = fields

	return nil
}

// pluginsLocations reads the default on-disk path of each plugin
// component.
type pluginsLocations struct{}

var _ Extractor = pluginsLocations{}

func (pluginsLocations) Section() string        { return "plugins.locations" }
func (pluginsLocations) Source() string         { return SrcDocsPlugins }
func (pluginsLocations) Anchor() *regexp.Regexp { return anchorPluginLocations }

func (pluginsLocations) Extract(section string, out *Digest) error {
	t, err := firstTableWithHeaders(section, "component", "default location")
	if err != nil {
		return err
	}

	nameCol := t.Column("component")
	pathCol := t.Column("default location")

	locations := make(map[string]string, len(t.Rows))
	for i := range t.Rows {
		name := strings.ToLower(Plain(t.Cell(i, nameCol)))
		path := Plain(t.Cell(i, pathCol))
		if name != "" && path != "" {
			locations[name] = path
		}
	}

	out.Plugins.Locations = locations

	return nonEmptyMap(locations, "plugin component locations")
}

// pluginsSubstitutionFields reads which fields of each plugin component
// resolve path placeholders.
type pluginsSubstitutionFields struct{}

var _ Extractor = pluginsSubstitutionFields{}

func (pluginsSubstitutionFields) Section() string        { return "plugins.substitution_fields" }
func (pluginsSubstitutionFields) Source() string         { return SrcDocsPlugins }
func (pluginsSubstitutionFields) Anchor() *regexp.Regexp { return anchorPluginEnvironment }

func (pluginsSubstitutionFields) Extract(section string, out *Digest) error {
	t, err := firstTableWithHeaders(section, colSubstitutionComponent, colSubstitutionFields)
	if err != nil {
		return err
	}

	nameCol := t.Column(colSubstitutionComponent)
	fieldsCol := t.Column(colSubstitutionFields)

	byComponent := make(map[string][]string, len(t.Rows))
	for i := range t.Rows {
		name := strings.ToLower(Plain(t.Cell(i, nameCol)))
		if name == "" {
			continue
		}
		byComponent[name] = BacktickedTokens(t.Cell(i, fieldsCol))
	}

	out.Plugins.SubstitutionFields = byComponent

	return nonEmptyMap(byComponent, "plugin substitution fields")
}
