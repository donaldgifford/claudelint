package upstream

// Source ids. Every id appears in the source table, in the extractors
// that read it, and in their tests, so each is a named constant rather
// than a repeated literal.
const (
	SrcDocsSkills       = "docs.skills"
	SrcDocsAgents       = "docs.agents"
	SrcDocsPlugins      = "docs.plugins"
	SrcDocsMarketplaces = "docs.marketplaces"
	SrcDocsHooks        = "docs.hooks"
	SrcDocsMCP          = "docs.mcp"
	SrcDocsTools        = "docs.tools"
	SrcDocsMemory       = "docs.memory"

	SrcSchemaStorePlugin      = "schemastore.plugin"
	SrcSchemaStoreMarketplace = "schemastore.marketplace"
	SrcSchemaStoreSettings    = "schemastore.settings"

	SrcAgentSkillsSpec      = "agentskills.spec"
	SrcAgentSkillsValidator = "agentskills.validator"
	SrcAnthropicQuickValid  = "anthropic.quick_validate"

	SrcClaudeCodeChangelog = "claudecode.changelog"

	SrcProbePluginSchema      = "probe.plugin_schema"
	SrcProbeMarketplaceSchema = "probe.marketplace_schema"
)

// Work-directory file extensions, without a leading dot.
const (
	extMarkdown = "md"
	extMDX      = "mdx"
	extJSON     = "json"
	extPython   = "py"
)

// Tier classifies how much authority a source carries, following
// DESIGN-0006 section 1.
const (
	// TierAuthoritative is the published Claude Code documentation. The
	// guardrail test compares Go constants against these sections only.
	TierAuthoritative byte = 'A'
	// TierCrossCheck is machine-readable but known to lag. Where it
	// disagrees with tier A the difference is recorded as information,
	// never as drift.
	TierCrossCheck byte = 'B'
	// TierPortable is the portable Agent Skills specification, kept
	// separate from the Claude Code superset.
	TierPortable byte = 'C'
	// TierMetadata is used in reports only.
	TierMetadata byte = 'D'
)

// Source is one upstream document.
type Source struct {
	// ID is the stable identifier used as the digest's source key, the
	// lock key, and the work-directory file stem.
	ID string
	// URL is fetched verbatim; redirects are followed.
	URL string
	// Ext is the work-directory file extension, without a dot.
	Ext string
	// Tier is one of the Tier constants.
	Tier byte
	// Optional marks a source whose failure is recorded but never fatal.
	// The code.claude.com/schemas probes are optional because they do
	// not exist yet; they are re-probed on every run so the tool notices
	// when they go live.
	Optional bool
}

// sourceTable is the fifteen documents of DESIGN-0006 section 1 plus
// the two schema probes, in id order.
//
// DESIGN-0006 section 1 lists fifteen rows; IMPL-0005 says sixteen.
// Fifteen is correct, and the design table is the source of truth.
var sourceTable = []Source{
	{
		ID:   SrcAgentSkillsSpec,
		URL:  "https://raw.githubusercontent.com/agentskills/agentskills/main/docs/specification.mdx",
		Ext:  extMDX,
		Tier: TierPortable,
	},
	{
		ID:   SrcAgentSkillsValidator,
		URL:  "https://raw.githubusercontent.com/agentskills/agentskills/main/skills-ref/src/skills_ref/validator.py",
		Ext:  extPython,
		Tier: TierPortable,
	},
	{
		ID:   SrcAnthropicQuickValid,
		URL:  "https://raw.githubusercontent.com/anthropics/skills/main/skills/skill-creator/scripts/quick_validate.py",
		Ext:  extPython,
		Tier: TierPortable,
	},
	{
		ID:   SrcClaudeCodeChangelog,
		URL:  "https://raw.githubusercontent.com/anthropics/claude-code/main/CHANGELOG.md",
		Ext:  extMarkdown,
		Tier: TierMetadata,
	},
	{
		ID:   SrcDocsAgents,
		URL:  "https://code.claude.com/docs/en/sub-agents.md",
		Ext:  extMarkdown,
		Tier: TierAuthoritative,
	},
	{
		ID:   SrcDocsHooks,
		URL:  "https://code.claude.com/docs/en/hooks.md",
		Ext:  extMarkdown,
		Tier: TierAuthoritative,
	},
	{
		ID:   SrcDocsMarketplaces,
		URL:  "https://code.claude.com/docs/en/plugin-marketplaces.md",
		Ext:  extMarkdown,
		Tier: TierAuthoritative,
	},
	{
		ID:   SrcDocsMCP,
		URL:  "https://code.claude.com/docs/en/mcp.md",
		Ext:  extMarkdown,
		Tier: TierAuthoritative,
	},
	{
		ID:   SrcDocsMemory,
		URL:  "https://code.claude.com/docs/en/memory.md",
		Ext:  extMarkdown,
		Tier: TierAuthoritative,
	},
	{
		ID:   SrcDocsPlugins,
		URL:  "https://code.claude.com/docs/en/plugins-reference.md",
		Ext:  extMarkdown,
		Tier: TierAuthoritative,
	},
	{
		ID:   SrcDocsSkills,
		URL:  "https://code.claude.com/docs/en/skills.md",
		Ext:  extMarkdown,
		Tier: TierAuthoritative,
	},
	{
		ID:   SrcDocsTools,
		URL:  "https://code.claude.com/docs/en/tools-reference.md",
		Ext:  extMarkdown,
		Tier: TierAuthoritative,
	},
	{
		ID:       SrcProbeMarketplaceSchema,
		URL:      "https://code.claude.com/schemas/marketplace.json",
		Ext:      extJSON,
		Tier:     TierCrossCheck,
		Optional: true,
	},
	{
		ID:       SrcProbePluginSchema,
		URL:      "https://code.claude.com/schemas/plugin.json",
		Ext:      extJSON,
		Tier:     TierCrossCheck,
		Optional: true,
	},
	{
		ID:   SrcSchemaStoreMarketplace,
		URL:  "https://www.schemastore.org/claude-code-marketplace.json",
		Ext:  extJSON,
		Tier: TierCrossCheck,
	},
	{
		ID:   SrcSchemaStorePlugin,
		URL:  "https://www.schemastore.org/claude-code-plugin-manifest.json",
		Ext:  extJSON,
		Tier: TierCrossCheck,
	},
	{
		ID:   SrcSchemaStoreSettings,
		URL:  "https://www.schemastore.org/claude-code-settings.json",
		Ext:  extJSON,
		Tier: TierCrossCheck,
	},
}

// Sources returns every upstream source in id order. The returned
// slice is a copy, so callers cannot mutate the table.
func Sources() []Source {
	out := make([]Source, len(sourceTable))
	copy(out, sourceTable)

	return out
}

// SourceByID returns the source with the given id. ok is false when no
// such source exists.
func SourceByID(id string) (Source, bool) {
	for _, s := range sourceTable {
		if s.ID == id {
			return s, true
		}
	}

	return Source{}, false
}

// File is the work-directory file name for a source.
func (s Source) File() string {
	return s.ID + "." + s.Ext
}
