/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import "github.com/m0090-dev/eec-go/internal/subcmd"
import "github.com/m0090-dev/eec-go/internal/core/types"
import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	//"fmt"
)

func init() {
	//fmt.Printf("Build mode: %s\n",types.BuildMode)
	
	// debug
	if types.LogMode == "debug" {
	  zerolog.SetGlobalLevel(zerolog.DebugLevel)
        // release
  	} else {
	  zerolog.SetGlobalLevel(zerolog.InfoLevel)
	  subcmd.HideWindowRunFlag =  true
	  subcmd.DeleterHideWindowRunFlag = true
	}
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
}
func main() {
	subcmd.Execute()
}
