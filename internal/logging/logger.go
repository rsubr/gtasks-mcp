package logging

import "log"

var level = "info"

func Init(l string) { level = l }

func Debug(v ...any) { if level == "debug" { log.Println(v...) } }
func Info(v ...any)  { if level == "debug" || level == "info" { log.Println(v...) } }
func Warn(v ...any)  { if level != "error" { log.Println(v...) } }
func Error(v ...any) { log.Println(v...) }
