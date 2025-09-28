package domain

import (
	"github.com/m0090-dev/eec-go/core/utils/general"
	"github.com/m0090-dev/eec-go/core/ext"
)

// ReadOrFallback is the same helper behavior as original core.
func ReadOrFallback(os ext.OS,logger ext.Logger,name string) (ext.Config, error) {
	var cfg ext.Config
	if os.FS.FileExists(name) {
		return ext.ReadConfig(os,logger,name)
	}
	tagData, err := ext.ReadTagData(os,logger,name)
	if err != nil {
		return cfg, err
	}
	for _, f := range tagData.ImportConfigFiles {
		var fcfg ext.Config
		if general.FileExists(f) {
			fcfg, _ = ext.ReadConfig(os,logger,f)
		} else {
			fcfg, _ = ext.ReadInlineConfig(os,logger,f)
		}
		fcfg.ApplyEnvs(os,logger)
		cfg = fcfg
	}
	return cfg, nil
}

