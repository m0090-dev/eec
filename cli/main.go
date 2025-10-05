/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import "github.com/m0090-dev/eec-go/cli/cmd"
import "github.com/m0090-dev/eec-go/core/types"
import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	//"fmt"
)

func init() {
	//fmt.Printf("Build mode: %s\n",types.BuildMode)
	if types.BuildMode == "debug" {
	  zerolog.SetGlobalLevel(zerolog.DebugLevel)
        } else {
	  zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
}
func main() {
	cmd.Execute()
}
