package main

/*
#include <stdlib.h>
typedef struct tagEngine {} Engine;
typedef Engine* PEngine;
// 文字列配列ラッパー
typedef struct {
    char** items;
    int count;
} CStringArray;
*/
import "C"
import (
	"time"
	"sync"
	"unsafe"
	"context"
	"github.com/m0090-dev/eec-go/core"
	"github.com/m0090-dev/eec-go/core/ext"
)

// ----------------- ハンドル管理 -----------------
var (
	engineMap = map[uintptr]*core.Engine{}
	nextID    uintptr = 1
	engineMu  sync.Mutex
)

func getEngine(id uintptr) (*core.Engine, bool) {
	engineMu.Lock()
	defer engineMu.Unlock()
	e, ok := engineMap[id]
	return e, ok
}

// ----------------- エンジン生成・破棄 -----------------

//export Engine_New
func Engine_New() C.PEngine {
	e := core.NewEngine(nil, nil)
	engineMu.Lock()
	id := nextID
	nextID++
	engineMap[id] = e
	engineMu.Unlock()
	return C.PEngine(unsafe.Pointer(id))
}

//export Engine_Close
func Engine_Close(p C.PEngine) {
	id := uintptr(unsafe.Pointer(p))
	engineMu.Lock()
	delete(engineMap, id)
	engineMu.Unlock()
}

// ----------------- Engine メソッドラッパー -----------------

//export Engine_Run
func Engine_Run(p C.PEngine,
	configFile *C.char,
	program *C.char,
	programArgs C.CStringArray,
	tag *C.char,
	imports C.CStringArray,
	waitTimeoutMs C.int,
	hideWindow C.int,deleterPath *C.char,deleterHideWindow C.int) C.int {

	id := uintptr(unsafe.Pointer(p))
	e, ok := getEngine(id)
	if !ok {
		return -1
	}

	// 文字列変換
	goConfigFile := C.GoString(configFile)
	goProgram := C.GoString(program)
	goTag := C.GoString(tag)
	var goHideWindow bool 
	if hideWindow == 1 {
		goHideWindow = true;
	} else {
		goHideWindow = false;
	}

	goDeleterPath := C.GoString(deleterPath)
	var goDeleterHideWindow bool

	if deleterHideWindow == 1 {
		goDeleterHideWindow = true;
	} else {
		goDeleterHideWindow = false;
	}

	// programArgs 配列変換
	var goProgramArgs []string
	count := int(programArgs.count)
	ptrs := (*[1 << 30]*C.char)(unsafe.Pointer(programArgs.items))[:count:count]
	for _, s := range ptrs {
		goProgramArgs = append(goProgramArgs, C.GoString(s))
	}

	// imports 配列変換
	var goImports []string
	countImp := int(imports.count)
	ptrsImp := (*[1 << 30]*C.char)(unsafe.Pointer(imports.items))[:countImp:countImp]
	for _, s := range ptrsImp {
		goImports = append(goImports, C.GoString(s))
	}

	// RunOptions を構築
	opts := core.RunOptions{
		ConfigFile:  goConfigFile,
		Program:     goProgram,
		ProgramArgs: goProgramArgs,
		Tag:         goTag,
		Imports:     goImports,
		WaitTimeout: time.Duration(waitTimeoutMs) * time.Millisecond,
		HideWindow: goHideWindow,
		DeleterPath:goDeleterPath,
		DeleterHideWindow: goDeleterHideWindow,
	}

	// 実行
	if err := e.Run(context.Background(), opts); err != nil {
		return -1
	}
	return 0
}









//export Engine_TagAdd
func Engine_TagAdd(p C.PEngine, name *C.char, configFile *C.char) C.int {
	id := uintptr(unsafe.Pointer(p))
	e, ok := getEngine(id)
	if !ok {
		return -1
	}

	goName := C.GoString(name)
	goConfig := C.GoString(configFile)

	tag := ext.TagData{
		ConfigFile: goConfig,
	}

	if err := e.TagAdd(goName, tag); err != nil {
		return -1
	}
	return 0
}

//export Engine_TagRead
func Engine_TagRead(p C.PEngine, name *C.char) C.int {
	id := uintptr(unsafe.Pointer(p))
	e, ok := getEngine(id)
	if !ok {
		return -1
	}

	goName := C.GoString(name)
	if err := e.TagRead(goName); err != nil {
		return -1
	}
	return 0
}

//export Engine_TagList
func Engine_TagList(p C.PEngine) C.int {
	id := uintptr(unsafe.Pointer(p))
	e, ok := getEngine(id)
	if !ok {
		return -1
	}
	if err := e.TagList(); err != nil {
		return -1
	}
	return 0
}

//export Engine_TagRemove
func Engine_TagRemove(p C.PEngine, name *C.char) C.int {
	id := uintptr(unsafe.Pointer(p))
	e, ok := getEngine(id)
	if !ok {
		return -1
	}
	goName := C.GoString(name)
	if err := e.TagRemove(goName); err != nil {
		return -1
	}
	return 0
}

//export Engine_Info
func Engine_Info(p C.PEngine) C.int {
	id := uintptr(unsafe.Pointer(p))
	e, ok := getEngine(id)
	if !ok {
		return -1
	}
	if err := e.Info(); err != nil {
		return -1
	}
	return 0
}

//export Engine_GenScript
func Engine_GenScript(p C.PEngine) C.int {
	id := uintptr(unsafe.Pointer(p))
	e, ok := getEngine(id)
	if !ok {
		return -1
	}
	if err := e.GenScript(); err != nil {
		return -1
	}
	return 0
}

// main は空でOK
func main() {}
