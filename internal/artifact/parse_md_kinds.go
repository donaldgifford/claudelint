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
	s.Name = doc.asString(keyName)
	s.Description = doc.asString(keyDescription)
	s.Model = doc.asString(keyModel)
	s.WhenToUse = doc.asString(keyWhenToUse)
	s.Context = doc.asString(keyContext)
	s.Agent = doc.asString(keyAgent)
	s.AllowedTools = doc.asToolList(keyAllowedTools)
	s.DisallowedTools = doc.asToolList(keyDisallowedTools)
	s.DisableModelInvocation, _ = doc.asBool(keyDisableModelInvocation)
	if v, ok := doc.asBool(keyUserInvocable); ok {
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
	c.Description = doc.asString(keyDescription)
	c.ArgumentHint = doc.asString(keyArgumentHint)
	c.Model = doc.asString(keyModel)
	c.WhenToUse = doc.asString(keyWhenToUse)
	c.Context = doc.asString(keyContext)
	c.Agent = doc.asString(keyAgent)
	c.AllowedTools = doc.asToolList(keyAllowedTools)
	c.DisallowedTools = doc.asToolList(keyDisallowedTools)
	c.DisableModelInvocation, _ = doc.asBool(keyDisableModelInvocation)
	if v, ok := doc.asBool(keyUserInvocable); ok {
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
	a.Name = doc.asString(keyName)
	a.Description = doc.asString(keyDescription)
	a.Tools = doc.asToolList(keyTools)
	a.DisallowedTools = doc.asToolList(keyDisallowedToolsCamel)
	a.Model = doc.asString(keyModel)
	a.PermissionMode = doc.asString(keyPermissionMode)
	a.MaxTurns = doc.asInt64(keyMaxTurns)
	a.Skills = doc.asStringList(keySkills)
	a.HasMCPServers = doc.has(keyMCPServers)
	a.HasHooks = doc.has(keyHooks)
	a.Memory = doc.asString(keyMemory)
	a.Background, _ = doc.asBool(keyBackground)
	a.Effort = doc.asString(keyEffort)
	a.Isolation = doc.asString(keyIsolation)
	a.Color = doc.asString(keyColor)
	a.InitialPrompt = doc.asString(keyInitialPrompt)
	return a, nil
}
