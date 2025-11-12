// Engine is the core library entrypoint. It contains pluggable implementations
// for executing commands and file operations so CLI can inject mocks for tests.
type Engine struct {
	OS types.OS
	//PtyData types.PtyData
	Logger interfaces.Logger
}


