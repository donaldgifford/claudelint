package rules

// DefaultHelpURI returns a stable URL for rule documentation based on
// the rule's ID. Rules that want to keep their help pointer trivially
// attached to the ID can do:
//
//	func (*myRule) HelpURI() string { return rules.DefaultHelpURI("skills/body-size") }
//
// The URL currently points at the project README anchor
// `#rule-<id-slug>` where <id-slug> is the rule ID with "/" replaced
// by "-" so it survives Markdown anchor generators. A future docs
// site can swap this implementation without touching rule files.
func DefaultHelpURI(ruleID string) string {
	return helpURIBase + "#rule-" + slugify(ruleID)
}

// helpURIBase is the single place that changes when the project
// switches from README anchors to a dedicated docs site.
const helpURIBase = "https://github.com/donaldgifford/claudelint/blob/main/README.md"

// slugify lower-cases the rule ID and replaces non-alphanumeric bytes
// with "-". Markdown renderers vary in their anchor rules; sticking
// to [a-z0-9-] sidesteps all of them.
func slugify(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9':
			out = append(out, c)
		case c >= 'A' && c <= 'Z':
			out = append(out, c+('a'-'A'))
		default:
			out = append(out, '-')
		}
	}
	return string(out)
}
