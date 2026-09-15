#### Using skill frontmatter outside Claude Code

| Distribution path                                                                                                                             | Frontmatter fields you can use                                                 |
| :-------------------------------------------------------------------------------------------------------------------------------------------- | :----------------------------------------------------------------------------- |
| Claude Code skills at [any level](#where-skills-live), including [plugin](/docs/en/plugins) skills                                                 | Every field in the table above                                                 |
| claude.ai skill uploads, the Skills API, and packaging with `package_skill.py` from [anthropics/skills](https://github.com/anthropics/skills) | `name`, `description`, `license`, `compatibility`, `metadata`, `allowed-tools` |
