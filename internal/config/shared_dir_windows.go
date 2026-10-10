// SPDX-License-Identifier: MIT

package config

import (
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// sharedDirRights are the rights that let a principal plant a file in a
// directory, directly or by taking control of it.
const sharedDirRights = windows.FILE_WRITE_DATA | // FILE_ADD_FILE on a directory
	windows.GENERIC_WRITE |
	windows.GENERIC_ALL |
	windows.WRITE_DAC |
	windows.WRITE_OWNER

// broadPrincipals are the well-known groups that stand for "other local users".
var broadPrincipals = sync.OnceValue(func() []*windows.SID {
	var sids []*windows.SID
	for _, kind := range []windows.WELL_KNOWN_SID_TYPE{
		windows.WinWorldSid,             // Everyone
		windows.WinAuthenticatedUserSid, // Authenticated Users
		windows.WinBuiltinUsersSid,      // BUILTIN\Users
	} {
		if sid, err := windows.CreateWellKnownSid(kind); err == nil {
			sids = append(sids, sid)
		}
	}
	return sids
})

// isSharedDir reports whether principals broader than the current user may
// create files in dir, which marks it as a shared location that config
// discovery must not walk into. Windows has no Unix permission bits (Go reports
// 0777 for every directory), so the DACL is inspected instead. A descriptor
// that cannot be read counts as shared, which stops the walk. See ADR-0019.
func isSharedDir(dir string) bool {
	sd, err := windows.GetNamedSecurityInfo(dir, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return true
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		return true // no DACL present
	}
	if dacl == nil {
		return true // NULL DACL grants everyone full access
	}
	return daclGrantsBroadWrite(dacl)
}

// daclGrantsBroadWrite reports whether an effective access-allowed ACE grants a
// broad principal any of sharedDirRights. Deny ACEs are not evaluated, so this
// errs toward reporting a directory as shared.
func daclGrantsBroadWrite(dacl *windows.ACL) bool {
	broad := broadPrincipals()
	if len(broad) == 0 {
		return true // cannot tell who is broad; fail closed
	}
	for i := range uint32(dacl.AceCount) {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, i, &ace); err != nil {
			return true
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE ||
			ace.Header.AceFlags&windows.INHERIT_ONLY_ACE != 0 ||
			ace.Mask&sharedDirRights == 0 {
			continue
		}
		// The trustee SID is stored inline, starting at SidStart.
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		for _, principal := range broad {
			if sid.Equals(principal) {
				return true
			}
		}
	}
	return false
}
