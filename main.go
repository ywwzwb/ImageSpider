package main

import "ywwzwb/imagespider/app"

// import (
// 	"log/slog"
// 	"os"
// 	"ywwzwb/imagespider/plugins"
// )

func main() {
	app.GetApplication().Run()
	// h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
	// 	Level: slog.LevelDebug,
	// })
	// l := slog.New(h)
	// slog.SetDefault(l)
	// f := plugins.FileIntegrityChecker{}
	// f.RunTest()
}
