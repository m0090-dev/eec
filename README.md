# eec (env-exec)

`eec` is a **Go-based Environment Execution Controller**.  
It lets you safely manage and execute programs with environment variables defined in configuration files (TOML / YAML / JSON / .env), without polluting your system environment.

---

## Features

- Configuration-file-based environment definitions (TOML / YAML / JSON / .env)
- **Tags** for grouping and reusing multiple environments
- Script generation for shortcut commands
- Safe execution — never modifies the global system environment
- Interactive REPL (`eec repl`) with in-session `reload` of edited configs, plus a `--watch` restart mode for `run`
- Inspection commands: `which`, `validate` and `doctor` for resolved environments
- Duplicate environment variable tracking with configurable warning behavior (`--duplicate-strict` / `--duplicate-no-warn`)
- Environment variable dump support for Unix, Windows and PowerShell shells
- Build automation via `mage`:
  - `mage buildcli debug` / `mage buildcli release`
  - `mage buildlib debug` / `mage buildlib release`
  - **Note:** `GOOS=linux` is **not supported** for `mage buildlib` on Windows

---

## Installation

```bash
# Build from source
mage buildcli release
```

---

## Commands

### 1. `run` — Run a program with a managed environment

Temporarily applies environment variables from a config file or tag, then launches the specified program. The global system environment is never affected.

```bash
eec run -c test.toml -p powershell -a "-NoExit","-Command","Write-Output 'hello world'"
```

| Flag | Description |
|---|---|
| `-c`, `--config-file` | Path to the configuration file |
| `-p`, `--program` | Program to execute |
| `-a`, `--args` | Program arguments (comma-separated) |
| `--tag` | Tag name to use |
| `-i`, `--imports` | Additional config files or tags to import (comma-separated) |
| `--wait-timeout` | Process wait timeout in seconds |
| `--hide-window` | Launch the program without a visible window |
| `--duplicate-strict` | Treat duplicate environment variable definitions as an error |
| `--duplicate-no-warn` | Suppress warnings for duplicate environment variables |
| `--no-expand` | Disable `${}` variable expansion (also available on `dump`) |
| `--no-exec` | Disable `$()` command substitution, leaving it literal (also available on `dump`) |
| `--watch` | Restart the program whenever a config file changes (see below) |
| `-v`, `--verbose` | Enable debug logging |

**Watch mode:** with `--watch`, `run` supervises the program: whenever one of the config files that fed the resolution changes, the child is stopped, the environment is re-resolved from scratch, and the program restarts. If the child exits on its own, watch mode ends with the child's exit code; Ctrl+C stops the child and exits 130. Watch mode requires file-based configs (inline configs cannot be watched).

```bash
eec run -c dev.toml -p myserver --watch
```

**Run using a tag:**

```bash
eec run --tag dev
```

---

### 2. `tag add` — Register a tag

Bundles multiple config files and/or other tags into a single named tag for easy reuse.

```bash
# Bundle multiple config files
eec tag add dev -i "base-dev.toml,go-dev.toml,python-dev.toml"

# Include other tags as well
eec tag add dev -i "base-dev.toml,testTag1,testTag2"

# Specify a config file directly (program path/args are auto-filled)
eec tag add myapp -c myapp.toml
```

- Items passed via `-i` that exist as files are automatically normalized to absolute paths.
- When `--config-file` is specified, `program.path` and `program.args` from that config are automatically populated into the tag.

---

### 3. `tag list` — List all tags

Displays all currently registered tags.

```bash
eec tag list
```

---

### 4. `tag read` — Show tag details

Shows the full configuration of a specific tag (config file, program, args, imports).

```bash
eec tag read dev
```

Example output:
```
=== Tag information ===
Tag:                  dev
Config:               /home/user/.eec/configs/base.toml
Program:              /usr/bin/bash
Args:                 -l, -i
Import config files:  go-dev.toml, python-dev.toml
```

---

### 5. `tag remove` — Delete a tag

Removes the specified tag. The remaining tag list is displayed after deletion.

```bash
eec tag remove dev
```

---

### 6. `tree` — Display the dependency tree of a tag

Visualizes the full dependency structure of a tag — which config files and sub-tags it pulls in — together with the value of every variable, so you can audit complex environments and spot redundant or conflicting definitions.

```bash
eec tree dev
```

Example output:
```
Dependency tree for tag: dev
Program: cmd /k vcvars64.bat
├── D:\dev\configs\base.toml
│   ├── Config: base environment  [sep=';']
│   ├── Env: GREETING = hello
│   ├── Env: PATH = D:\tools\bin [list]
│   ├── Env: PATH = D:\priority [prepend]
│   └── Env: WINKEY = x [os: windows]
└── Imported file: D:\dev\configs\extra.toml
    ├── Env: SHARED = from-extra
    └── Capture: setup-env.bat

Overrides:
SHARED
  D:\dev\configs\base.toml: from-base
  D:\dev\configs\extra.toml: from-extra
  final: from-extra

Summary: 2 config(s), 7 env definition(s), 1 override(s), 42 final variable(s)
```

Reading the output:

- Each config file is shown once by its **full path**; imported files and imported tags are nested under the tag that pulls them in. Files reached twice are marked `(already visited)`, imports that are neither a file nor a tag are marked `Unknown import`.
- Env lines show the defined value. List values are joined with the list separator; values longer than 60 characters are truncated with `…` (use `--full` to print them whole). Badges mark how a definition merges: `[list]` for a plain list, `[append]` / `[prepend]` for merge-section entries, and `[os: windows]` (etc.) for OS-section entries.
- After the tree, the **Overrides** section lists every variable defined more than once: the definition chain in resolution order (file → value) and the winning `final:` value. A **Summary** line counts configs, env definitions, overrides and final variables.
- When writing to a terminal, file names are bold, keys are cyan, badges are yellow. Colors turn off automatically when output is piped, with `--no-color`, or when the `NO_COLOR` environment variable is set.

To trace a single variable instead of printing the whole tree:

```bash
eec tree dev --key PATH
```

```
PATH
  D:\dev\configs\base.toml: D:\tools\bin [list]
  D:\dev\configs\base.toml: D:\priority [prepend]
  final: D:\priority;D:\tools\bin;C:\Windows
```

This prints only that variable's definition chain across the tag's configs, in resolution order, plus its fully resolved final value — handy for answering "why does PATH look like this?". The key lookup follows the OS case rules (case-insensitive on Windows). If the variable is not defined by the tag, the command exits with a non-zero status.

---

### 7. `gen script` — Generate utility scripts

Generates shortcut scripts for each registered tag.

```bash
eec gen script
```

| OS | Generated file |
|---|---|
| Windows | `t<tagname>.bat` |
| Linux / macOS | `t<tagname>` (shell script) |

Example (with a `dev` tag registered):

```bash
tdev cmd
# → runs cmd with the dev environment
```

---

### 8. `dump` — Dump resolved environment variables

Outputs the fully resolved environment variables for a given config or tag in shell-ready format. Useful for debugging or sourcing into other scripts.

```bash
# Unix format (export KEY='VALUE')
eec dump --tag dev --shell unix

# Windows format (set "KEY=VALUE")
eec dump --tag dev --shell win

# PowerShell format ($env:KEY = 'VALUE')
eec dump --tag dev --shell pwsh

# Raw KEY=VALUE format
eec dump --tag dev
```

| `--shell` value | Output format |
|---|---|
| `unix` | `export KEY='VALUE'` (single-quoted, `'` escaped as `'\''`) |
| `win` | `set "KEY=VALUE"` |
| `pwsh` / `powershell` | `$env:KEY = 'VALUE'` (single-quoted, `'` escaped by doubling) |
| (omitted) | `KEY=VALUE` |

---

### 9. `info` — Show version and runtime info

Displays eec's version, PID, build hash, and other runtime details.

```bash
eec info
```

Example output:
```
=== eec Information ===
version:     1.0.0
pid:         12345
goOS:        linux
commitHash:  abc1234
logMode:     release
```

---

### 10. `which` — Find a program in the managed environment

Resolves the environment like `run`/`dump` and prints the absolute path of a program as found in that environment's `PATH`. On Windows the `PATHEXT` extensions are tried for bare names; on Unix a file must be executable. Exits with status 1 when the program is not found.

```bash
eec which -t dev ninja
eec which -c test.toml ./tool
```

---

### 11. `validate` — Check configs without running anything

Loads every config source selected by `-c`/`-t`/`-i` exactly the way `run` would — import recursion and circular-import detection included — and builds its environment. Prints `OK <source>` or `NG <source>: <error>` per source and exits non-zero when any source is NG.

```bash
eec validate -c test.toml -i extra.toml
eec validate -t dev
```

---

### 12. `doctor` — Sanity-check a resolved environment

Prints a checklist for the environment resolved from `-c`/`-t`/`-i`:

- ✓/✗ the configs resolve without error
- ✓/⚠ every absolute `PATH` entry exists (missing entries are warnings, not failures)
- ✓/✗ the program given with `-p` is found in the resolved `PATH`
- ✓/✗ every import of the tag given with `-t` resolves to a file or another tag

The exit status is non-zero only when a ✗ check fails.

```bash
eec doctor -t dev -p ninja
```

---

### 13. `shell` — Interactive shell in the managed environment

Starts an interactive shell as a child process with the resolved environment (`%ComSpec%` on Windows, `$SHELL` or `sh` on Unix) and propagates its exit code. Note that this is a **new** shell process: it cannot modify the shell that launched eec — no program can change its parent process's environment.

```bash
eec shell -t dev
```

---

### 14. `repl` — Interactive session with reload

Starts a line-oriented session in the resolved environment. Each line runs through the OS shell (`cmd /c` on Windows, `sh -c` on Unix) with the session environment and working directory; a failing command prints `[exit N]` and the session continues.

| Builtin | Description |
|---|---|
| `env [KEY]` | Print the session environment (or one variable) |
| `set KEY=VALUE` | Set a session variable (survives `reload`) |
| `unset KEY` | Remove a session variable (survives `reload`) |
| `cd [DIR]` | Change (or print) the session working directory |
| `reload` | Re-read the configs and rebuild the environment |
| `help` | List the builtins |
| `exit` / `quit` | Leave the REPL |

**`reload` is the supported way to pick up config edits in a live session**: edit the config file, type `reload`, and keep working in the same session with the new environment — no process restart needed. Variables changed with `set`/`unset` are re-applied on top of the reloaded environment.

```bash
eec repl -t dev
eec> set DEBUG=1
eec> ninja        # runs with the session environment
eec> reload       # after editing the config file
```

---

### 15. `init` — Write a starter config

Writes a commented starter `eec.toml` (or the given path) with an `[envs]` section, OS-specific section examples, and a commented-out `[capture]` example. Refuses to overwrite an existing file unless `--force` is given.

```bash
eec init
eec init dev.toml --force
```

---

### 16. `completion` — Shell completion scripts

Generates completion scripts for bash, zsh, fish and PowerShell (cobra's built-in command). The `-t`/`--tag` flags complete tag names from the tag store, and `-c`/`--config-file` uses regular file completion.

```bash
# bash example
source <(eec completion bash)
```

---

## Data Directory: `EEC_HOME` and Portable Mode

Tags are stored in eec's data directory, which is resolved in this order:

1. **`EEC_HOME`** environment variable — its value *is* the data directory itself (the equivalent of `~/.eec`).
2. **Portable mode** — if a `.eec` directory exists next to the `eec` executable, it is used. A portable folder or removable drive can therefore carry both the binary and its tags, leaving no state on the host machine.
3. **`~/.eec`** — the default.

When a tag references config files that live *inside* the data directory, the paths are stored **relative to the data directory** instead of absolute. Such tags keep working when the data directory moves — for example when a portable drive shows up under a different drive letter. Absolute paths outside the data directory are stored (and resolved) exactly as before.

---

## Configuration File Format

Config files can be written in TOML, YAML, JSON, or .env format.

**TOML:**

```toml
[configs]
description = "My environment"

[envs]
GOROOT = "/usr/local/go"
GOPATH = "/home/user/go"
PATH   = ["/usr/local/go/bin", "/usr/bin"]   # array = entries are joined with the OS path separator

[program]
path = "pwsh"
args = ["-Command", "gci env:GO*"]
```

**YAML:**

```yaml
configs:
  - description: "My environment"

envs:
  GOROOT: "/usr/local/go"
  GOPATH: "/home/user/go"
  PATH:
    - "/usr/local/go/bin"
    - "/usr/bin"

program:
  path: "pwsh"
  args: ["-Command", "gci env:GO*"]
```

**JSON:**

```json
{
  "configs": [{ "description": "My environment" }],
  "envs": [
    { "GOROOT": "/usr/local/go" },
    { "GOPATH": "/home/user/go" },
    { "PATH": ["C:\\go\\bin", "C:\\tools\\bin"] }
  ],
  "program": {
    "path": "pwsh",
    "args": ["-Command", "gci env:GO*"]
  }
}
```

**.env:**

```dotenv
GOROOT=/usr/local/go
GOPATH=/home/user/go
PATH=/usr/local/go/bin:/usr/bin
```

${} variable expansion and $() command substitution are supported in TOML, YAML, and JSON, but not in .env files.


When a variable is defined as an array, its values are merged into (appended to) the existing variable rather than replacing it. For example, PATH = ["C:\\my\\bin"] appends to the current PATH, whereas PATH = "C:\\my\\bin" overwrites it entirely.


### Prepend and Append Sections

By default a list value is **appended** after the existing value. The `[envs.prepend]` sub-table reverses that: its list values are merged **before** the existing value, so they win priority (for PATH-like variables, earlier entries are found first). `[envs.append]` is an explicit synonym for the plain list behavior.

```toml
[envs]
PATH = ["tools"]               # appended

[envs.prepend]
PATH = ["priority-tools"]      # placed before everything else

[envs.append]
PATH = ["more-tools"]          # appended, like the plain list
```

For one variable, the merge order is:

```
[envs.prepend] parts  ++  existing value  ++  plain [envs] list parts  ++  [envs.append] parts
```

(with the config's scalar definition, if any, taking the place of the existing value at its step). Across multiple configs the same rule applies in config order: an earlier config's appends land before a later config's prepends are placed in front, so for configs processed in order the final value is `later-prepend ++ base ++ earlier-append`.

Details:

- Merging splits both sides on the list separator, trims whitespace, and removes duplicates — the **first** occurrence wins. Prepending a part that is already present therefore *moves it to the front* instead of duplicating it.
- A scalar value inside `[envs.prepend]` is treated as a one-element prepend list.
- `${}` / `$()` expansion in prepend values works exactly like in plain values.
- The sections also nest inside OS sections: `[envs.windows.prepend]`, `[envs.linux.append]`, and so on.
- `prepend` and `append` are reserved names directly under `envs` (and under OS sections) — when such a key holds a *table* it is treated as a section. A variable literally named `prepend` holding a scalar or a list (e.g. `prepend = ["x"]`) is still treated as a regular variable.


### OS-Specific Sections

Variables that only apply on one OS can be grouped under `[envs.windows]`, `[envs.linux]`, or `[envs.darwin]` (aliases: `win` for `windows`, `mac` / `macos` for `darwin`). The same nesting works in YAML and JSON. Only the section matching the current OS is applied, after the common `[envs]` entries, so OS-specific values override common ones and `${}` references work in both directions between common and OS-specific variables.

```toml
[envs]
ROOT = "/opt/tools"

[envs.windows]
PATH = ["${ROOT}\\bin"]
GREETING = "windows"

[envs.linux]
PATH = ["${ROOT}/bin"]
GREETING = "linux"
```

Note: `windows`, `linux`, `darwin` (and their aliases) are reserved names directly under `envs` — when such a key holds a table it is treated as an OS section, not a variable. A reserved name holding a scalar (e.g. `windows = "x"`) is still treated as a regular variable.


### Variable Expansion

Use `${}` to reference other environment variables within values. References are resolved via **topological sort**, so declaration order does not matter — forward references work fine.

```toml
[envs]
STEP_01 = "Alpha"
STEP_02 = "${STEP_01}-Beta"
STEP_03 = "${STEP_02}-Gamma"

# Fine even in reverse order — topological sort handles it
FINAL   = "Result: ${STEP_03}"
PATH    = ["D:\\work\\${STEP_03}", "C:\\base\\${STEP_01}"]
```

### Command Substitution

Use `$()` to embed the output of a shell command into a value.

```toml
[envs]
BUILD_DATE = "$(date +%Y%m%d)"
GIT_HASH   = "$(git rev-parse --short HEAD)"
```

Both `${}` and `$()` can be combined freely in a single value.

### Disabling Expansion and Command Substitution

Expansion can be turned off per config in the `[configs]` section, or per invocation with CLI flags (the flags win). When several configs are merged, the last config that specifies a toggle wins.

```toml
[configs]
expand_vars = false    # ${} (and $()) are left completely unexpanded
exec_commands = false  # ${} still expands, but $() is left literal instead of being executed
```

| Where | Setting | Effect |
|---|---|---|
| `[configs]` | `expand_vars = false` | No substitution at all; values are used verbatim |
| `[configs]` | `exec_commands = false` | `${}` expands; `$()` stays literal and is never executed |
| CLI (`run` / `dump`) | `--no-expand` | Force-disables expansion regardless of the config |
| CLI (`run` / `dump`) | `--no-exec` | Force-disables `$()` execution regardless of the config |

This is useful when consuming configs you do not fully trust (a `$()` runs a shell command) or when values must contain literal `${}` / `$()` text.

### Built-in Variables

Two built-in variables are available for `${}` expansion in config values:

| Variable | Value |
|---|---|
| `${CONFIG_DIR}` | The directory containing the config file being processed. Not defined for inline configs. |
| `${EEC_HOME}` | The resolved data directory (see [Data Directory](#data-directory-eec_home-and-portable-mode)). |

```toml
[envs]
TOOLS = "${CONFIG_DIR}/tools"
PATH  = ["${TOOLS}/bin"]
```

Built-ins have the **lowest priority**: a value from the real environment or a variable defined in the config itself wins over the built-in of the same name. They are also **expansion-only** — they never appear in the final environment unless the real environment or the config actually defines them.

### Capturing an Environment from a Script (`[capture]`)

Some tools ship a script whose whole job is to set up environment variables (a compiler toolchain's setup batch file, an SDK's environment script, and so on). The generic `[capture]` section runs such commands and imports the environment they produce — no tool-specific support is involved:

```toml
[capture]
commands = ["C:\\path\\to\\setup-env.bat"]
```

A plain list form (`capture = ["..."]`) works too, in TOML, YAML, and JSON. After the config's own `[envs]` are resolved, each command runs in order through the OS shell on top of the current environment, and the resulting environment is diffed back in:

- variables the command **added or changed** replace the old values wholesale (a captured `PATH` replaces `PATH` entirely),
- variables the command **removed** disappear,
- everything else is left untouched.

`${}` in a command string is expanded against the current environment first. Captures run arbitrary shell commands, so they obey the same gate as `$()`: when `exec_commands` is disabled (config or `--no-exec`), captures are skipped with a warning. Each capture has a 60-second timeout.

> **Caveats:** On Windows, a `.bat` file runs inside the `cmd` process, so a plain path in `commands` is enough (for example, a toolchain setup script such as `vcvars64.bat` works this way — it is just an ordinary batch file as far as eec is concerned). On Unix, the command must run *in the capture shell itself*, so source your script: `commands = [". \"/path/to/setup-env.sh\""]`.

---

## Duplicate Environment Variable Tracking

When multiple config files or tags define the same variable, eec tracks and reports the overrides.

| Option | Behavior |
|---|---|
| Default | Print a warning for each overridden variable |
| `--duplicate-strict` | Treat any override as an error and abort |
| `--duplicate-no-warn` | Suppress all override warnings |

---

## Use Cases

- **Isolated test environments:** Inject test-specific variables temporarily without touching the system
- **Multi-language development:** Switch between Go / Rust / Python environments using tags
- **CI/CD pipelines:** Reproducible environments driven entirely by config files
- **Dependency auditing:** Use `tree` to visualize and verify complex environment compositions

---

## Summary

`eec (env-exec)` is more than a tag manager —  
it's a **Go-based tool for cleanly isolating, managing, and executing environments**.

It shines in testing, multi-environment development, and any scenario where clean separation from system configuration is a hard requirement.
