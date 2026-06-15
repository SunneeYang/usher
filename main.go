package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	desktopApp, err := NewDesktopApp()
	if err != nil {
		log.Fatal(err)
	}

	err = wails.Run(&options.App{
		Title:     "usher",
		Width:     680,
		Height:    480,
		MinWidth:  380,
		MinHeight: 420,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 18, G: 18, B: 20, A: 1},
		OnStartup:        desktopApp.startup,
		Bind: []interface{}{
			desktopApp,
		},
		Mac: &mac.Options{
			TitleBar: mac.TitleBarDefault(),
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
