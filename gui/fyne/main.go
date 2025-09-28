package main

import (
	"context"
	"strings"
	"github.com/m0090-dev/eec-go/core"
	"github.com/m0090-dev/eec-go/core/ext"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"fyne.io/fyne/v2/dialog"
)

func main() {
	a := app.New()
	w := a.NewWindow("eec GUI")

	// Engine
	e := core.NewEngine(&ext.OS{
		FS:          ext.OSFS{},
		Executor:    ext.DefaultExecutor{},
		Console:     ext.DefaultConsole{},
		Env:         ext.OSEnv{},
		CommandLine: ext.DefaultCommandLine{},
	}, nil)

	var configFile, programPath string
	var tagName string

	// Widgets
	configLabel := widget.NewLabel("設定ファイル: 未選択")
	programEntry := widget.NewEntry()
	programEntry.SetPlaceHolder("実行するプログラムのパスを入力")

	argsEntry := widget.NewEntry()
	argsEntry.SetPlaceHolder("プログラム引数をカンマ区切りで入力")

	tagEntry := widget.NewEntry()
	tagEntry.SetPlaceHolder("タグ名")

	outputLog := widget.NewMultiLineEntry()
	outputLog.SetPlaceHolder("ログ出力")
	outputLog.Wrapping = fyne.TextWrapWord

	// ログ出力関数
	logMsg := func(msg string) {
		outputLog.SetText(outputLog.Text + msg + "\n")
	}

	// GUIレイアウト
	w.SetContent(container.NewVBox(
		configLabel,
		widget.NewButton("設定ファイル選択", func() {
			dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
				if err == nil && reader != nil {
					configFile = reader.URI().Path()
					configLabel.SetText("設定ファイル: " + configFile)
				}
			}, w)
		}),
		widget.NewLabel("実行するプログラム"),
		programEntry,
		widget.NewLabel("プログラム引数"),
		argsEntry,
		widget.NewLabel("タグ名"),
		tagEntry,
		widget.NewButton("Run", func() {
			programPath = programEntry.Text
			if configFile == "" || programPath == "" {
				dialog.ShowInformation("エラー", "設定ファイルとプログラムを入力してください", w)
				return
			}

			opts := core.RunOptions{
				ConfigFile:  configFile,
				Program:     programPath,
				ProgramArgs: strings.Split(argsEntry.Text, ","),
			}

			go func() {
				logMsg("実行開始: " + programPath)
				if err := e.Run(context.Background(), opts); err != nil {
					logMsg("Run error: " + err.Error())
					dialog.ShowError(err, w)
				} else {
					logMsg("Run 完了")
					dialog.ShowInformation("完了", "処理が完了しました", w)
				}
			}()
		}),
		widget.NewButton("Tag Add", func() {
			if tagEntry.Text == "" {
				dialog.ShowInformation("エラー", "タグ名を入力してください", w)
				return
			}
			tagName = tagEntry.Text
			tagData := ext.TagData{
				ConfigFile:       configFile,
				ImportConfigFiles: []string{"dev"},
			}
			if err := e.TagAdd(tagName, tagData); err != nil {
				logMsg("TagAdd error: " + err.Error())
				dialog.ShowError(err, w)
			} else {
				logMsg("TagAdd 完了: " + tagName)
			}
		}),
		widget.NewButton("Tag List", func() {
			if err := e.TagList(); err != nil {
				logMsg("TagList error: " + err.Error())
				dialog.ShowError(err, w)
			} else {
				logMsg("TagList 完了")
			}
		}),
		widget.NewButton("Tag Read", func() {
			if tagName == "" {
				dialog.ShowInformation("エラー", "タグ名を入力してください", w)
				return
			}
			if err := e.TagRead(tagName); err != nil {
				logMsg("TagRead error: " + err.Error())
				dialog.ShowError(err, w)
			} else {
				logMsg("TagRead 完了: " + tagName)
			}
		}),
		widget.NewButton("Tag Remove", func() {
			if tagName == "" {
				dialog.ShowInformation("エラー", "タグ名を入力してください", w)
				return
			}
			if err := e.TagRemove(tagName); err != nil {
				logMsg("TagRemove error: " + err.Error())
				dialog.ShowError(err, w)
			} else {
				logMsg("TagRemove 完了: " + tagName)
			}
		}),
		widget.NewButton("Info", func() {
			if err := e.Info(); err != nil {
				logMsg("Info error: " + err.Error())
				dialog.ShowError(err, w)
			} else {
				logMsg("Info 完了")
			}
		}),
		widget.NewButton("Gen Script", func() {
			if err := e.GenScript(); err != nil {
				logMsg("GenScript error: " + err.Error())
				dialog.ShowError(err, w)
			} else {
				logMsg("GenScript 完了")
			}
		}),
		outputLog,
	))

	w.Resize(fyne.NewSize(600, 600))
	w.ShowAndRun()
}
