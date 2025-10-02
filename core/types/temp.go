package types

/*type TempData struct {*/
    /*ParentPID int*/
    /*ChildPID int*/
    /*ConfigFile string*/
    /*Program string*/
    /*ProgramArgs []string*/
/*}*/


type TempData struct {
	ParentPID int
	ChildPID  int
	ConfigFile  string
	Program     string
	ProgramArgs []string
	Tag         string
	Imports     []string
	WaitTimeout int64  // time.Duration を秒またはミリ秒に変換して保存
	HideWindow  bool
	DeleterPath string
	DeleterHideWindow bool
}
