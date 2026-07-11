package artifact

// ParseClaudeMD parses a CLAUDE.md file into a typed *ClaudeMD.
// CLAUDE.md files usually have no frontmatter; when they do, keys are
// preserved with their byte-accurate ranges for rules that care.
func ParseClaudeMD(path string, src []byte) (*ClaudeMD, *ParseError) {
	doc, err := parseMarkdown(path, src)
	if err != nil {
		return nil, err
	}
	return &ClaudeMD{
		Base:        doc.Base,
		Frontmatter: doc.Frontmatter,
		Body:        doc.Body,
	}, nil
}

// ParseSkill parses a SKILL.md file into a typed *Skill. Companion
// file indexing is performed by IndexSkillCompanions, which the
// discovery/engine wiring calls after a successful parse.
func ParseSkill(path string, src []byte) (*Skill, *ParseError) {
	doc, err := parseMarkdown(path, src)
	if err != nil {
		return nil, err
	}
	s := &Skill{
		Base:        doc.Base,
		Frontmatter: doc.Frontmatter,
		Body:        doc.Body,
	}
	s.Name = doc.asString("name")
	s.Description = doc.asString("description")
	s.Model = doc.asString("model")
	s.WhenToUse = doc.asString("when_to_use")
	s.Context = doc.asString("context")
	s.Agent = doc.asString("agent")
	s.AllowedTools = doc.asToolList("allowed-tools")
	s.DisallowedTools = doc.asToolList("disallowed-tools")
	s.DisableModelInvocation, _ = doc.asBool("disable-model-invocation")
	if v, ok := doc.asBool("user-invocable"); ok {
		s.UserInvocable = &v
	}
	return s, nil
}

// ParseCommand parses a slash-command .md file into a typed *Command.
func ParseCommand(path string, src []byte) (*Command, *ParseError) {
	doc, err := parseMarkdown(path, src)
	if err != nil {
		return nil, err
	}
	c := &Command{
		Base:        doc.Base,
		Frontmatter: doc.Frontmatter,
		Body:        doc.Body,
	}
	c.Description = doc.asString("description")
	c.ArgumentHint = doc.asString("argument-hint")
	c.Model = doc.asString("model")
	c.WhenToUse = doc.asString("when_to_use")
	c.Context = doc.asString("context")
	c.Agent = doc.asString("agent")
	c.AllowedTools = doc.asToolList("allowed-tools")
	c.DisallowedTools = doc.asToolList("disallowed-tools")
	c.DisableModelInvocation, _ = doc.asBool("disable-model-invocation")
	if v, ok := doc.asBool("user-invocable"); ok {
		c.UserInvocable = &v
	}
	return c, nil
}

// ParseAgent parses a subagent .md file into a typed *Agent.
func ParseAgent(path string, src []byte) (*Agent, *ParseError) {
	doc, err := parseMarkdown(path, src)
	if err != nil {
		return nil, err
	}
	a := &Agent{
		Base:        doc.Base,
		Frontmatter: doc.Frontmatter,
		Body:        doc.Body,
	}
	a.Name = doc.asString("name")
	a.Description = doc.asString("description")
	a.Tools = doc.asToolList("tools")
	a.DisallowedTools = doc.asToolList("disallowedTools")
	a.Model = doc.asString("model")
	a.PermissionMode = doc.asString("permissionMode")
	a.MaxTurns = doc.asInt64("maxTurns")
	a.Skills = doc.asStringList("skills")
	a.HasMCPServers = doc.has("mcpServers")
	a.HasHooks = doc.has("hooks")
	a.Memory = doc.asString("memory")
	a.Background, _ = doc.asBool("background")
	a.Effort = doc.asString("effort")
	a.Isolation = doc.asString("isolation")
	a.Color = doc.asString("color")
	a.InitialPrompt = doc.asString("initialPrompt")
	return a, nil
}
