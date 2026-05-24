package fileinfo

import "syscall"

func newSys() any { return &syscall.Win32FileAttributeData{} }
