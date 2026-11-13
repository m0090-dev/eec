# eec (env-exec)

`eec` is a **Go-based Environment Execution Controller**.  
It lets you safely manage and execute programs with environment variables defined in configuration files (TOML / YAML / JSON / .env), without polluting your system environment.

---

## Features

- Configuration-file-based environment definitions (TOML / YAML / JSON / .env)
- **Tags** for grouping and reusing multiple environments
- Script generation for shortcut commands
- Safe execution — never modifies the global system environment
- Interactive REPL and restart functionality with state management
- Duplicate environment variable tracking with configurable warning behavior (`--duplicate-strict` / `--duplicate-no-warn`)
- Environment variable dump support for both Unix and Windows shells
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
| `-v`, `--verbose` | Enable debug logging |

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

Visualizes the full dependency structure of a tag — which config files and sub-tags it pulls in — in a hierarchical tree format. Useful for auditing complex environments and spotting redundant or conflicting variable definitions.

```bash
eec tree dev
```

Example output:
```
Dependency tree for tag: dev
└── Imported tag: dev-base
    └── Imported file: base-dev.toml
        ├── Env: PATH
        ├── Env: INCLUDE
        └── Env: LIB
└── Imported tag: dev-lang
    ├── Imported file: go-dev.toml
    ├── Imported file: rust-dev.toml
    └── Imported file: python-dev.toml
└── Imported tag: dev-tools
    ├── Imported file: use-tools-dev.toml
    └── Imported file: gnu-tools-dev.toml
```

Duplicate environment variable warnings are also shown after the tree is printed.

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
# Unix format (export KEY=VALUE)
eec dump --tag dev --shell unix

# Windows format (set KEY=VALUE)
eec dump --tag dev --shell win

# Raw KEY=VALUE format
eec dump --tag dev
```

| `--shell` value | Output format |
|---|---|
| `unix` | `export KEY=VALUE` |
| `win` | `set KEY=VALUE` |
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
