// tree.go — the `eec tree` renderer.
//
// The tree shows, for one tag: every config file it pulls in (its own
// config, imported files and imported tags, recursively), the env
// definitions in each file WITH their values, capture commands and
// programs; then an overrides section listing every variable that is
// defined more than once (the definition chain in resolution order
// plus the winning final value); then a one-line summary.
//
// Output goes to Engine.Out (default os.Stdout) so tests can capture
// it. ANSI colors are emitted only when that writer is a terminal,
// --no-color is not set and NO_COLOR is unset; piped output is always
// plain.
package core

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/m0090-dev/eec/internal/ext/types"
	"github.com/m0090-dev/eec/internal/ext/utils/domain"
)

// TreeOptions controls Engine.TreeWithOptions.
type TreeOptions struct {
	// Full disables the ~60 character truncation of rendered values.
	Full bool
	// NoColor disables ANSI colors even on a terminal.
	NoColor bool
	// Key selects trace mode: only this variable's definition chain
	// and final resolved value are printed.
	Key string
}

// Tree renders the dependency tree of a tag with default options.
func (e *Engine) Tree(tagName string) error {
	return e.TreeWithOptions(tagName, TreeOptions{})
}

// TreeWithOptions renders the dependency tree of a tag (or, with
// opts.Key set, traces a single variable through it).
func (e *Engine) TreeWithOptions(tagName string, opts TreeOptions) error {
	data, err := types.ReadTagData(e.Runtime, tagName)
	if err != nil {
		return fmt.Errorf("failed to read tag data for %s: %w", tagName, err)
	}

	out := e.outWriter()
	style := treeStyle{enabled: treeColorEnabled(out, opts.NoColor)}
	goos := e.Runtime.Env().GOOS()
	sep := e.Runtime.Env().PathListSeparator()

	// Resolve once: the result feeds the overrides section, the
	// summary's final-variable count and trace mode. (Imported tags
	// are expanded recursively by ResolveRunOptions, so one call
	// covers the whole tree.)
	_, _, _, finalEnv, allConfigs, resErr := domain.ResolveRunOptions(types.RunOptions{Tag: tagName}, data, e.Runtime)

	if opts.Key != "" {
		if resErr != nil {
			return fmt.Errorf("failed to resolve tag %s: %w", tagName, resErr)
		}
		tracker := types.TrackEnvOverrides(allConfigs, goos)
		return renderTrace(out, style, tracker, finalEnv, goos, tagName, opts.Key, sep, opts.Full)
	}

	r := &treeRenderer{
		e:            e,
		w:            out,
		style:        style,
		full:         opts.Full,
		visitedFiles: make(map[string]bool),
		visitedTags:  make(map[string]bool),
	}
	r.renderRoot(tagName, data)
	fmt.Fprintln(out)

	if resErr != nil {
		e.Runtime.Logger().Warn().Err(resErr).Msg("tree: failed to resolve run options; overrides unavailable")
		fmt.Fprintf(out, "Summary: %d config(s), %d env definition(s)\n", r.configCount, r.envCount)
		return nil
	}

	tracker := types.TrackEnvOverrides(allConfigs, goos)
	renderOverrides(out, style, tracker, finalEnv, sep, opts.Full)
	overridden := len(tracker.OverriddenKeys())
	fmt.Fprintf(out, "Summary: %d config(s), %d env definition(s), %d override(s), %d final variable(s)\n",
		r.configCount, r.envCount, overridden, len(finalEnv))
	return nil
}

// ---------------- renderer ----------------

type treeRenderer struct {
	e     *Engine
	w     io.Writer
	style treeStyle
	full  bool

	visitedFiles map[string]bool
	visitedTags  map[string]bool
	configCount  int
	envCount     int
}

// branchLine prints one tree node line: prefix + branch + text.
func (r *treeRenderer) branchLine(prefix string, last bool, text string) {
	branch := "├── "
	if last {
		branch = "└── "
	}
	fmt.Fprintln(r.w, prefix+branch+text)
}

// childPrefix returns the prefix under a node printed with
// branchLine(prefix, last, ...).
func childPrefix(prefix string, last bool) string {
	if last {
		return prefix + "    "
	}
	return prefix + "│   "
}

func (r *treeRenderer) renderRoot(tagName string, data types.TagData) {
	fmt.Fprintf(r.w, "Dependency tree for tag: %s\n", tagName)
	if data.Program != "" {
		fmt.Fprintf(r.w, "Program: %s\n", programText(data.Program, data.ProgramArgs))
	}
	r.visitedTags[tagName] = true
	r.renderTagBody(data, "", false)
}

// renderTagBody renders the children of a tag (its config file and
// its imports) under prefix. When includeProgram is true the tag's
// program is shown as a child line (used for imported tags; the root
// tag's program is part of the header instead).
func (r *treeRenderer) renderTagBody(data types.TagData, prefix string, includeProgram bool) {
	var children []func(prefix string, last bool)

	if includeProgram && data.Program != "" {
		text := "Program: " + programText(data.Program, data.ProgramArgs)
		children = append(children, func(p string, last bool) {
			r.branchLine(p, last, text)
		})
	}
	if data.ConfigFile != "" {
		cf := data.ConfigFile
		children = append(children, func(p string, last bool) {
			r.renderConfigNode(cf, "", p, last)
		})
	}
	for _, imp := range data.ImportConfigFiles {
		imp := imp
		children = append(children, func(p string, last bool) {
			r.renderImport(imp, p, last)
		})
	}
	for i, ch := range children {
		ch(prefix, i == len(children)-1)
	}
}

func (r *treeRenderer) renderImport(imp string, prefix string, last bool) {
	if isConfigFile(imp) {
		r.renderConfigNode(imp, "Imported file: ", prefix, last)
		return
	}
	sub, err := types.ReadTagData(r.e.Runtime, imp)
	if err != nil {
		r.branchLine(prefix, last, "Unknown import: "+imp)
		return
	}
	if r.visitedTags[imp] {
		r.branchLine(prefix, last, "Imported tag: "+imp+" (already visited)")
		return
	}
	r.visitedTags[imp] = true
	r.branchLine(prefix, last, "Imported tag: "+imp)
	r.renderTagBody(sub, childPrefix(prefix, last), true)
}

// renderConfigNode prints one config file (full path) and its
// contents as children. label prefixes the path ("Imported file: "
// for imports, "" for a tag's own config).
func (r *treeRenderer) renderConfigNode(path string, label string, prefix string, last bool) {
	resolved := domain.ResolveDataDirRelative(r.e.Runtime, path)
	absPath, err := filepath.Abs(resolved)
	if err != nil {
		r.branchLine(prefix, last, label+path+" (failed to resolve path)")
		return
	}
	nodeText := label + r.style.bold(absPath)
	if r.visitedFiles[absPath] {
		r.branchLine(prefix, last, nodeText+" (already visited)")
		return
	}
	r.visitedFiles[absPath] = true
	r.configCount++
	r.branchLine(prefix, last, nodeText)

	config, err := types.ReadConfig(r.e.Runtime, absPath)
	if err != nil {
		r.e.Runtime.Logger().Warn().Err(err).Str("config", absPath).Msg("config読み込み失敗（スキップ）")
		r.branchLine(childPrefix(prefix, last), true, "(could not read config)")
		return
	}
	r.renderConfigBody(config, childPrefix(prefix, last))
}

func (r *treeRenderer) renderConfigBody(config types.Config, prefix string) {
	sep := configSeparator(config, r.e.Runtime.Env().PathListSeparator())

	var children []func(prefix string, last bool)

	for _, meta := range config.Configs {
		meta := meta
		children = append(children, func(p string, last bool) {
			desc := meta.Description
			if desc == "" {
				desc = "(no description)"
			}
			r.branchLine(p, last, fmt.Sprintf("Config: %s  [sep='%s']", desc, meta.Separator))
		})
	}

	for _, env := range config.Envs {
		env := env
		children = append(children, func(p string, last bool) {
			r.envCount++
			r.branchLine(p, last, "Env: "+r.style.cyan(env.Key)+" = "+r.envValueText(env, sep)+envBadgeText(r.style, env, isListValue(env.Value)))
			// A string value pointing at a .env file pulls that file
			// in as a nested config node (historical behavior).
			if val, ok := env.Value.(string); ok && strings.HasSuffix(val, ".env") {
				r.renderConfigNode(val, "", childPrefix(p, last), true)
			}
		})
	}

	for _, cmd := range config.CaptureCommands {
		cmd := cmd
		children = append(children, func(p string, last bool) {
			r.branchLine(p, last, "Capture: "+cmd)
		})
	}

	if config.Program.Path != "" {
		text := "Program: " + programText(config.Program.Path, config.Program.Args)
		children = append(children, func(p string, last bool) {
			r.branchLine(p, last, text)
		})
	}

	for i, ch := range children {
		ch(prefix, i == len(children)-1)
	}
}

// envValueText renders an env entry's value (lists joined with the
// separator), truncated unless full output was requested.
func (r *treeRenderer) envValueText(env types.Environ, sep string) string {
	val, _ := renderEnvValue(env.Value, sep)
	return truncateValue(val, r.full)
}

// ---------------- overrides & trace ----------------

// renderOverrides prints the override chains for every variable that
// was defined more than once, followed by the winning final value.
func renderOverrides(w io.Writer, style treeStyle, tracker *types.EnvTracker, finalEnv []string, sep string, full bool) {
	keys := tracker.OverriddenKeys()
	if len(keys) == 0 {
		return
	}
	finalMap := envListToMap(finalEnv)
	fmt.Fprintln(w, "Overrides:")
	for _, key := range keys {
		src := tracker.Vars[key]
		fmt.Fprintln(w, style.cyan(key))
		for _, def := range src.History {
			fmt.Fprintln(w, "  "+def.Source+": "+definitionText(style, def, sep, full))
		}
		if fv, ok := finalMap[key]; ok {
			fmt.Fprintln(w, "  final: "+truncateValue(fv, full))
		} else {
			fmt.Fprintln(w, "  final: (not set)")
		}
	}
	fmt.Fprintln(w)
}

// renderTrace prints only one variable's definition chain and its
// final resolved value — `eec tree <tag> --key KEY`.
func renderTrace(w io.Writer, style treeStyle, tracker *types.EnvTracker, finalEnv []string, goos, tagName, key, sep string, full bool) error {
	canon := types.CanonicalEnvKey(goos, key)
	finalMap := envListToMap(finalEnv)
	src, defined := tracker.Vars[canon]
	finalVal, inFinal := finalMap[canon]
	if !defined && !inFinal {
		return fmt.Errorf("variable %s is not defined in tag %s (nor present in the resolved environment)", key, tagName)
	}
	fmt.Fprintln(w, style.cyan(canon))
	if defined {
		for _, def := range src.History {
			fmt.Fprintln(w, "  "+def.Source+": "+definitionText(style, def, sep, full))
		}
	}
	if inFinal {
		fmt.Fprintln(w, "  final: "+style.green(truncateValue(finalVal, full)))
	} else {
		fmt.Fprintln(w, "  final: (not set)")
	}
	return nil
}

// definitionText renders one tracked definition: its raw value plus
// any badges (same rules as env lines in the tree).
func definitionText(style treeStyle, def types.EnvDefinition, sep string, full bool) string {
	val, isList := renderEnvValue(def.Value, sep)
	env := types.Environ{Prepend: def.Prepend, Section: def.Section, Origin: def.Origin}
	return truncateValue(val, full) + envBadgeText(style, env, isList)
}

// ---------------- value helpers ----------------

// renderEnvValue flattens a raw env value for display: list values
// are joined with the separator; everything else is printed as-is.
func renderEnvValue(v interface{}, sep string) (string, bool) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		parts := make([]string, 0, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			parts = append(parts, fmt.Sprint(rv.Index(i).Interface()))
		}
		return strings.Join(parts, sep), true
	}
	if s, ok := v.(string); ok {
		return s, false
	}
	return fmt.Sprint(v), false
}

func isListValue(v interface{}) bool {
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array
}

// envBadgeText builds the badge suffix for an env definition:
// [prepend] for prepend-section entries, [append] for append-section
// entries, [list] for plain list entries, plus [os: <name>] for
// OS-section entries.
func envBadgeText(style treeStyle, env types.Environ, isList bool) string {
	var badges []string
	switch {
	case env.Prepend:
		badges = append(badges, "[prepend]")
	case env.Section == "append":
		badges = append(badges, "[append]")
	case isList:
		badges = append(badges, "[list]")
	}
	if env.Origin != "" {
		badges = append(badges, "[os: "+env.Origin+"]")
	}
	if len(badges) == 0 {
		return ""
	}
	return " " + style.yellow(strings.Join(badges, " "))
}

// truncateValue caps a rendered value at ~60 characters unless full
// output was requested.
func truncateValue(s string, full bool) string {
	if full {
		return s
	}
	const maxRunes = 60
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes-1]) + "…"
}

func programText(path string, args []string) string {
	if len(args) == 0 {
		return path
	}
	return path + " " + strings.Join(args, " ")
}

// configSeparator picks the separator a config's list values are
// joined with for display: the config's own [configs] separator when
// set, else the OS path-list separator.
func configSeparator(config types.Config, osSep string) string {
	for _, meta := range config.Configs {
		if meta.Separator != "" {
			return meta.Separator
		}
	}
	return osSep
}

func envListToMap(env []string) map[string]string {
	m := make(map[string]string, len(env))
	for _, e := range env {
		if strings.HasPrefix(e, "=") {
			continue
		}
		if parts := strings.SplitN(e, "=", 2); len(parts) == 2 {
			m[parts[0]] = parts[1]
		}
	}
	return m
}

// isConfigFile reports whether name looks like a config file (vs a
// tag name) by its extension.
func isConfigFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".toml", ".yaml", ".yml", ".json", ".env":
		return true
	default:
		return false
	}
}

// ---------------- colors ----------------

type treeStyle struct{ enabled bool }

func (s treeStyle) wrap(code, text string) string {
	if !s.enabled {
		return text
	}
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}

func (s treeStyle) bold(text string) string   { return s.wrap("1", text) }
func (s treeStyle) cyan(text string) string   { return s.wrap("36", text) }
func (s treeStyle) yellow(text string) string { return s.wrap("33", text) }
func (s treeStyle) green(text string) string  { return s.wrap("32", text) }

// treeColorEnabled decides whether ANSI colors may be emitted:
// colors need a terminal on the other end of the writer, no explicit
// --no-color, and no NO_COLOR environment variable.
func treeColorEnabled(w io.Writer, noColor bool) bool {
	if noColor {
		return false
	}
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	st, err := f.Stat()
	return err == nil && (st.Mode()&os.ModeCharDevice) != 0
}
