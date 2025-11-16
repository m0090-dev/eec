package main

import (
	"github.com/m0090-dev/eec/internal/core-deleter"
)
func main(){
	e := core_deleter.NewEngine(nil,nil)
	if err := e.Run(); err != nil {
		e.Logger.Fatal().Err(err).Msg("Failed to run")
	}
}
