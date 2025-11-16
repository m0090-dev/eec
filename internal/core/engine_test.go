package core_test

import (
	"context"
	"testing"
	"time"
	"github.com/m0090-dev/eec/internal/core"
	"github.com/m0090-dev/eec/internal/ext/types"
	"github.com/m0090-dev/eec/internal/ext/interfaces"
	"github.com/rs/zerolog/log"
)

func TestEngineRun(t *testing.T) {
	os := types.OS{
		FS: interfaces.OSFS{},
		Executor: interfaces.DefaultExecutor{},
		Console: interfaces.DefaultConsole{},
		Env: interfaces.OSEnv{},
		CommandLine: interfaces.DefaultCommandLine{},
	}
	e := core.NewEngine(&os,nil)
	opts := types.RunOptions{
		ConfigFile: "../test.toml",
		Program: "checkitems",
		WaitTimeout: 1000*time.Second,
	}
	if err := e.Run(context.Background(),opts); err != nil {
		log.Fatal().Err(err).Msg("Run failed")
	}
}
func TestEngineTag(t *testing.T) {
	e := core.NewEngine(nil,nil)
	tagName := "うぇーい"
	tagData := types.TagData {
		ConfigFile: "../test.toml",
		ImportConfigFiles: []string{"dev"},
	}
	// add 
	if err := e.TagAdd(tagName,tagData); err != nil {
		log.Fatal().Err(err).Msg("Tag add failed")
	}
	// list
	if err := e.TagList(); err != nil {
		log.Fatal().Err(err).Msg("Tag list failed")
	}
	// read
	if err := e.TagRead(tagName); err != nil {
		log.Fatal().Err(err).Msg("Tag read failed")
	}
	// remove
	if err := e.TagRemove(tagName); err != nil {
		log.Fatal().Err(err).Msg("Tag remove failed")
	}
	
}
func TestEngineInfo(t *testing.T) {
	e := core.NewEngine(nil,nil)
	if err := e.Info();err != nil {
		log.Fatal().Err(err).Msg("Info failed")
	}
}
func TestEngineGenScript(t *testing.T) {
	e := core.NewEngine(nil,nil)
	if err := e.GenScript(); err != nil {
		log.Fatal().Err(err).Msg("Gen script failed")
	}
}

