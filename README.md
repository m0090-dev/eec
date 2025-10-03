# eec (env-exec)

`eec` is a **Go-based Environment Execution Controller**.  
It allows you to safely manage and execute environments based on configuration files (TOML/YAML/JSON) without polluting your system environment.  

With `eec`, you can:
- Run programs with temporary environments defined in TOML files  
- Group multiple environments under "tags" for easy access  
- Generate utility scripts for quick launching  
- Use interactive or restart modes for flexible workflow  

---

## Features
- Configuration-file-based environment definitions (TOML/YAML/JSON, future support for .env)
- Tags for grouping and easy execution
- Script generation for shortcut commands
- Safe execution without modifying the global system
- Interactive REPL and restart functionality with state management

---

## Core Commands

### 1. run
Run a program with a given environment.

Example:
eec run --config-file test.toml --program powershell --program-args="-NoExit","-Command","Write-Output 'hello world'"

Effect:
- Loads environment from `test.toml`
- Launches `powershell` and runs `echo hello world`
- The environment is temporary and does not affect the system globally

---

### 2. tag add
Register a configuration or program as a reusable tag.

Example:
eec tag add dev --import base-dev.toml --import go-dev.toml --import python-dev.toml

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

### 4. tag read
Read the details of a specific tag.

Example:
eec tag read dev

Effect:
- Displays the configuration and imports associated with the `dev` tag

---

### 5. run with --tag
Run a program using an existing tag.

Example:
eec run --tag dev

Effect:
- Loads the environment linked to `dev` and runs the program defined there
- No need to specify `--config-file` manually

---

### 6. gen script
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

### 7. repl
Start an interactive mode for `eec`.

Example:
eec repl

Effect:
- Provides an interactive shell to repeatedly use `eec` features
- Useful for exploration and frequent environment switching

---

### 8. restart
Restart a running environment.

Example:
eec restart

Effect:
- Stops and restarts
- Relies on the **background utility `eec-deleter`** for proper state management and cleanup

---

## Purpose

- Run environments without polluting the system
- Use temporary configurations for testing and isolated development
- Manage complex multi-language setups through configuration files and tags
- Improve usability through generated scripts
- Support safe and flexible workflows with REPL and restart features

---

## Summary

`eec (env-exec)` is not just a tag manager,  
but a **Go-based tool for cleanly managing, isolating, and executing environments**.  

It is especially useful for testing, multi-environment development, and scenarios where you need clean separation from the system configuration.
