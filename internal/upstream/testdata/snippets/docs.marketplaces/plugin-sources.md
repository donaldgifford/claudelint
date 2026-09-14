## Plugin sources

| Source | Type | Fields | Notes |
| ------------- | ------------------------------- | ---------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Relative path | `string` (e.g. `"./my-plugin"`) | none | Local directory within the marketplace repo. Must start with `./`, unles |
| `github` | object | `repo`, `ref?`, `sha?` |  |
| `url` | object | `url`, `ref?`, `sha?` | Git URL source |
| `git-subdir` | object | `url`, `path`, `ref?`, `sha?` | Subdirectory within a git repo. Clones sparsely to minimize bandwidth fo |
| `npm` | object | `package`, `version?`, `registry?` | Installed via `npm install` |
| `archive` | object | `url`, `sha256?` | Zip archive downloaded over HTTPS. Works without git or npm on the user' |
| `command` | object | `command`, `timeout?`, `mode?` | Plugin directory produced by running a local command, re-run once per se |

### Relative paths

### GitHub repositories

| Field | Type | Description |
| :----- | :----- | :-------------------------------------------------------------------- |
| `repo` | string | Required. GitHub repository in `owner/repo` format |
| `ref` | string | Optional. Git branch or tag (defaults to repository default branch) |
| `sha` | string | Optional. Full 40-character git commit SHA to pin to an exact version |

### Git repositories

| Field | Type | Description |
| :---- | :----- | :------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `url` | string | Required. Full git repository URL (`https://` or `git@`). The `.git` suf |
| `ref` | string | Optional. Git branch or tag (defaults to repository default branch) |
| `sha` | string | Optional. Full 40-character git commit SHA to pin to an exact version |

### Git subdirectories

| Field | Type | Description |
| :----- | :----- | :------------------------------------------------------------------------------------------------------- |
| `url` | string | Required. Git repository URL, GitHub `owner/repo` shorthand, or SSH URL |
| `path` | string | Required. Subdirectory path within the repo containing the plugin (for e |
| `ref` | string | Optional. Git branch or tag (defaults to repository default branch) |
| `sha` | string | Optional. Full 40-character git commit SHA to pin to an exact version |

### npm packages

| Field | Type | Description |
| :--------- | :----- | :------------------------------------------------------------------------------------------- |
| `package` | string | Required. Package name or scoped package (for example, `@org/plugin`) |
| `version` | string | Optional. Version or version range (for example, `2.1.0`, `^2.0.0`, |
| `registry` | string | Optional. Custom npm registry URL. Defaults to the system npm registry ( |

### Zip archives

| Field | Type | Description |
| :------- | :----- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `url` | string | Required. HTTPS URL of the zip archive. Claude Code rejects `http://` UR |
| `sha256` | string | Optional. SHA-256 digest of the archive as 64 hex characters, uppercase |

#### Authenticate archive downloads

| Place | Downloads that get the headers | When Claude Code runs a `headersHelper` set there |
| :----------------------- | :----------------------------------------------------------------------------------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Marketplace `url` source | Archive downloads on the marketplace URL's origin, meaning the same sche | Before each fetch of the marketplace's `marketplace.json` and before eac |
| Plugin entry | That entry's download only | Only when a user installs or updates that one plugin by itself and [acce |

##### Add a headersHelper to a plugin entry

#### Write the headersHelper command

#### When Claude Code skips a headersHelper command or drops its output

#### How users accept a headersHelper command

##### Installs and updates that refuse the command instead of asking

##### When a marketplace `url` source's command runs

| Settings file | When Claude Code runs the command |
| :---------------------------------------------------------------------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| User settings, a `--settings` file, or a managed settings file on the machine | Without asking, including during a background marketplace refresh |
| A project's `.claude/settings.json` or `.claude/settings.local.json` | Only after the user accepts the [workspace trust dialog](/docs/en/permis |
| Server-managed settings | Only after the user approves the delivered settings in the [security app |

### Command sources

| Field | Type | Description |
| :-------- | :----- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `command` | string | Required. Shell command that prints the plugin directory's absolute path |
| `timeout` | number | Optional. Whole number of seconds to wait for the command before giving |
| `mode` | string | Optional. `"copy"` (default) copies the printed directory into the plugi |

#### Copy mode and link mode

#### How users accept the command

#### When Claude Code re-runs the command

### Advanced plugin entries

### Strict mode

| Value | Behavior |
| :--------------- | :--------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `true` (default) | `plugin.json` is the authority. The marketplace entry can supplement it |
| `false` | The marketplace entry is the entire definition. If the plugin also has a |
