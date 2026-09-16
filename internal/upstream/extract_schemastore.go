package upstream

import "regexp"

// Sentinels and path segments used to locate values in the SchemaStore
// documents. Each is a value the schema must still contain for the
// section to mean anything, which is a steadier handle than the deep
// anyOf paths those values actually sit at.
const (
	sentinelHookEvent   = "PreToolUse"
	sentinelDefaultMode = "acceptEdits"

	segProperties  = "properties"
	segHooks       = "hooks"
	segMCPServers  = "mcpServers"
	segType        = "type"
	segSource      = "source"
	segPlugins     = "plugins"
	segItems       = "items"
	segPermissions = "permissions"
	segDefaultMode = "defaultMode"
)

// schemaStorePluginManifest reads the claude-code-plugin-manifest
// schema. Everything it produces is informational: where the schema and
// the documentation disagree, the documentation wins.
type schemaStorePluginManifest struct{}

var _ Extractor = schemaStorePluginManifest{}

func (schemaStorePluginManifest) Section() string        { return "schemastore.plugin_manifest" }
func (schemaStorePluginManifest) Source() string         { return SrcSchemaStorePlugin }
func (schemaStorePluginManifest) Anchor() *regexp.Regexp { return nil }

func (schemaStorePluginManifest) Extract(section string, out *Digest) error {
	root, err := decodeJSONDocument([]byte(section), "plugin manifest schema")
	if err != nil {
		return err
	}

	out.SchemaStore.PluginManifest = SchemaStorePlugin{
		Properties: jsonKeys(root, segProperties),
		HookEvents: jsonEnumsContaining(root, sentinelHookEvent),
		HookTypes:  jsonConstsAt(root, segHooks, segType, jsonKeywordConst),
		MCPTypes:   jsonConstsAt(root, segMCPServers, segType, jsonKeywordConst),
	}

	return nonEmpty(out.SchemaStore.PluginManifest.Properties, "plugin manifest schema properties")
}

// schemaStoreMarketplaceSchema reads the claude-code-marketplace
// schema.
type schemaStoreMarketplaceSchema struct{}

var _ Extractor = schemaStoreMarketplaceSchema{}

func (schemaStoreMarketplaceSchema) Section() string        { return "schemastore.marketplace" }
func (schemaStoreMarketplaceSchema) Source() string         { return SrcSchemaStoreMarketplace }
func (schemaStoreMarketplaceSchema) Anchor() *regexp.Regexp { return nil }

func (schemaStoreMarketplaceSchema) Extract(section string, out *Digest) error {
	root, err := decodeJSONDocument([]byte(section), "marketplace schema")
	if err != nil {
		return err
	}

	out.SchemaStore.Marketplace = SchemaStoreMarketplace{
		Properties:            jsonKeys(root, segProperties),
		PluginEntryProperties: jsonKeys(root, segProperties, segPlugins, segItems, segProperties),
		SourceKinds:           jsonConstsAt(root, segSource, segProperties, segSource, jsonKeywordConst),
	}

	return nonEmpty(out.SchemaStore.Marketplace.Properties, "marketplace schema properties")
}

// schemaStoreSettingsSchema reads the claude-code-settings schema. Its
// hook events are object keys rather than an enum, which is why they
// are read by path instead of by sentinel.
type schemaStoreSettingsSchema struct{}

var _ Extractor = schemaStoreSettingsSchema{}

func (schemaStoreSettingsSchema) Section() string        { return "schemastore.settings" }
func (schemaStoreSettingsSchema) Source() string         { return SrcSchemaStoreSettings }
func (schemaStoreSettingsSchema) Anchor() *regexp.Regexp { return nil }

func (schemaStoreSettingsSchema) Extract(section string, out *Digest) error {
	root, err := decodeJSONDocument([]byte(section), "settings schema")
	if err != nil {
		return err
	}

	events := jsonKeys(root, segProperties, segHooks, segProperties)
	if len(events) == 0 {
		events = jsonEnumsContaining(root, sentinelHookEvent)
	}

	modes := jsonStrings(root, segProperties, segPermissions, segProperties, segDefaultMode, jsonKeywordEnum)
	if len(modes) == 0 {
		modes = jsonEnumsContaining(root, sentinelDefaultMode)
	}

	out.SchemaStore.Settings = SchemaStoreSettings{
		DefaultModes: modes,
		HookEvents:   events,
	}

	return nonEmpty(events, "settings schema hook events")
}
