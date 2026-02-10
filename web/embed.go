package web

import "embed"

//go:embed templates/*.html templates/*.sh templates/*.ps1
var TemplateFS embed.FS

//go:embed static/*
var StaticFS embed.FS
