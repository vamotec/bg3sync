package main

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed icon.png
var iconData []byte

var resourceIconPng = &fyne.StaticResource{
	StaticName:    "icon.png",
	StaticContent: iconData,
}
