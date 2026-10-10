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

// ACE types from winnt.h that golang.org/x/sys/windows does not define.
const (
	accessDeniedObjectACEType         = 0x6
	accessAllowedCallbackACEType      = 0x9 // conditional allow; same layout as ACCESS_ALLOWED_ACE
	accessDeniedCallbackACEType       = 0xA
	accessDeniedCallbackObjectACEType = 0xC
)

// broadPrincipals returns the well-known groups that stand for "other local
// users". It fails if any of them cannot be constructed, because a partial set
// would let a grant to the missing group go unnoticed. A variable so tests can
// exercise the failure path.
var broadPrincipals = sync.OnceValues(func() ([]*windows.SID, error) {
	kinds := []windows.WELL_KNOWN_SID_TYPE{
		windows.WinWorldSid,             // Everyone
		windows.WinAuthenticatedUserSid, // Authenticated Users
		windows.WinBuiltinUsersSid,      // BUILTIN\Users
	}
	sids := make([]*windows.SID, 0, len(kinds))
	for _, kind := range kinds {
		sid, err := windows.CreateWellKnownSid(kind)
		if err != nil {
			return nil, err
		}
		sids = append(sids, sid)
	}
	return sids, nil
})

// isSharedDir reports whether principals broader than the current user may
// create files in dir, which marks it as a shared location that config
// discovery must not walk into. Windows has no Unix permission bits (Go reports
// 0777 for every directory), so the DACL is inspected instead. Anything that
// cannot be evaluated counts as shared, which stops the walk. See ADR-0019.
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

// daclGrantsBroadWrite reports whether an effective allow ACE grants a broad
// principal any of sharedDirRights. Every judgment it cannot make errs toward
// reporting the directory as shared:
//   - deny ACEs are not evaluated, so a later deny does not clear a grant;
//   - a conditional (callback) allow ACE is treated as if its condition holds;
//   - any other ACE form that carries a relevant right fails closed, since its
//     trustee cannot be read at the ACCESS_ALLOWED_ACE offset.
func daclGrantsBroadWrite(dacl *windows.ACL) bool {
	broad, err := broadPrincipals()
	if err != nil {
		return true
	}
	for i := range uint32(dacl.AceCount) {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, i, &ace); err != nil {
			return true
		}
		// Every ACE form keeps its access mask right after the header.
		if ace.Header.AceFlags&windows.INHERIT_ONLY_ACE != 0 || ace.Mask&sharedDirRights == 0 {
			continue
		}
		switch ace.Header.AceType {
		case windows.ACCESS_DENIED_ACE_TYPE, accessDeniedObjectACEType,
			accessDeniedCallbackACEType, accessDeniedCallbackObjectACEType:
			continue
		case windows.ACCESS_ALLOWED_ACE_TYPE, accessAllowedCallbackACEType:
			// The trustee SID is stored inline, starting at SidStart.
			sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
			for _, principal := range broad {
				if sid.Equals(principal) {
					return true
				}
			}
		default:
			return true
		}
	}
	return false
}
