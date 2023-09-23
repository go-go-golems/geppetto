//go:build windows

package ui

// TODO(manuel, 2023-09-22) This code is actually untested, because I don't have a windows machine

import (
	"errors"
	"golang.org/x/sys/windows"
	"os"
)

func openTTY() (*os.File, error) {
	handle, err := windows.GetStdHandle(windows.STD_INPUT_HANDLE)
	if err != nil {
		return nil, err
	}

	fd := os.NewFile(uintptr(handle), "conin$")
	if fd == nil {
		return nil, errors.New("failed to create file from console handle")
	}

	return fd, nil
}
