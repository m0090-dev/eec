# eec (env-exec) 

`eec` is a **Go-based Environment Execution Controller**.  
It allows you to safely manage and execute environments based on configuration files (TOML/YAML/JSON/.env) without polluting your system environment.  

With `eec`, you can:
- Run programs with temporary environments defined in configuration files  
- Group multiple environments under "tags" for easy access  
- Generate utility scripts for quick launching  
- Use interactive or restart modes for flexible workflow  
- Build the CLI, GUI, and libraries using `mage` (`mage buildcli`, `mage buildgui`, `mage buildlib`)

---

## Features
- Configuration-file-based environment definitions (TOML/YAML/JSON/.env)
- Tags for grouping and easy execution
- Script generation for shortcut commands
- Safe execution without modifying the global system
- Interactive REPL and restart functionality with state management
- Build automation using `mage`:
  - `mage buildcli debug` / `mage buildcli release`
  - `mage buildgui debug` / `mage buildgui release`
  - `mage buildlib debug` / `mage buildlib release`
  - **Note:** On Windows, `GOOS=linux` is **not supported** for `mage buildlib`

---

## Core Commands

### 1. run [command] [flags]
Run a program with a given environment.

Example:
eec run -c test.toml -p powershell -a "-NoExit","-Command","Write-Output 'hello world'"


Effect:
- Loads environment from `test.toml`
- Launches `powershell` and runs `echo hello world`
- The environment is temporary and does not affect the system globally

---

### 2. tag add [tag name] [flags]
Register a configuration or program as a reusable tag.

Example:
eec tag add dev -i "base-dev.toml,go-dev.toml,python-dev.toml"
eec tag add dev -i "base-dev.toml,testTag1,testTag2"

Effect:
- Creates a `dev` tag that combines multiple TOML configurations
- Allows easy launching with `--tag dev`

---

### 3. tag list
List all registered tags.

Example:
eec tag list

Effect:
- Shows all tags currently available in the system

---

### 4. tag read [tag name]
Read the details of a specific tag.

Example:
eec tag read dev

Effect:
- Displays the configuration and imports associated with the `dev` tag

---

---

### 5. tree [tag name]

Example:
eec tree dev

Effect:
- Displays the dependency tree for the specified tag.
- Shows which configuration files and sub-tags are imported.
- Useful for understanding and debugging complex environment setups.

Description:
The `tree` command reads the metadata of a tag and prints its configuration dependency structure in a hierarchical (tree-like) format.
This helps visualize how multiple TOML/YAML/JSON files are combined to form a complete environment.

Example Output:
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

Use Case:
Ideal for reviewing how a tag aggregates its environment definitions, confirming imports, and avoiding redundant or conflicting variable settings.

---

### 6. run with --tag
Run a program using an existing tag.

Example:
eec run --tag dev

Effect:
- Loads the environment linked to `dev` and runs the program defined there
- No need to specify `--config-file` manually

---

### 7. gen script
Generate utility scripts for quick access to tags.

Example:
eec gen script

Effect:
- On Windows: creates `t<tag>.bat`
- On Linux/Mac: creates `t<tag>` shell scripts
- For example, if `dev` exists:
  tdev cmd
  → runs `cmd` with the `dev` environment

---



## Purpose

- Run environments without polluting the system
- Use temporary configurations for testing and isolated development
- Manage complex multi-language setups through configuration files and tags
- Improve usability through generated scripts
- Support safe and flexible workflows with REPL and restart features
- Automate building of CLI, GUI, and libraries via `mage`

---

## Summary

`eec (env-exec)` is not just a tag manager,  
but a **Go-based tool for cleanly managing, isolating, and executing environments**.  

It is especially useful for testing, multi-environment development, and scenarios where you need clean separation from the system configuration.  
