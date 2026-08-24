// SPDX-License-Identifier: MIT
//go:build integration && windows

package e2e

import (
	"os"
	"os/exec"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

var processJobs sync.Map

func configureProcessTree(cmd *exec.Cmd) {
	cmd.SysProcAttr = &windows.SysProcAttr{CreationFlags: windows.CREATE_NEW_PROCESS_GROUP}
	cmd.Cancel = func() error { return closeProcessJob(cmd) }
}

func attachProcessTree(cmd *exec.Cmd) error {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return err
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		_ = windows.CloseHandle(job)
		return err
	}
	process, err := windows.OpenProcess(windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE, false, uint32(cmd.Process.Pid))
	if err != nil {
		_ = windows.CloseHandle(job)
		return err
	}
	defer windows.CloseHandle(process)
	if err := windows.AssignProcessToJobObject(job, process); err != nil {
		_ = windows.CloseHandle(job)
		return err
	}
	processJobs.Store(cmd, job)
	return nil
}

func closeProcessJob(cmd *exec.Cmd) error {
	value, ok := processJobs.LoadAndDelete(cmd)
	if !ok {
		if cmd == nil || cmd.Process == nil {
			return os.ErrProcessDone
		}
		return cmd.Process.Kill()
	}
	return windows.CloseHandle(value.(windows.Handle))
}

func forceTerminateProcessTree(cmd *exec.Cmd) error { return closeProcessJob(cmd) }

func releaseProcessTree(cmd *exec.Cmd) { _ = closeProcessJob(cmd) }
