//go:build windows

package pty

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// GetChildProcesses returns the executable names of all child and grandchild processes of parentPid.
func GetChildProcesses(parentPid int) ([]string, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(snapshot)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))

	err = windows.Process32First(snapshot, &entry)
	if err != nil {
		return nil, err
	}

	var children []string
	for {
		if int(entry.ParentProcessID) == parentPid {
			name := windows.UTF16ToString(entry.ExeFile[:])
			children = append(children, name)
			sub, _ := GetChildProcesses(int(entry.ProcessID))
			children = append(children, sub...)
		}
		err = windows.Process32Next(snapshot, &entry)
		if err != nil {
			break
		}
	}
	return children, nil
}
