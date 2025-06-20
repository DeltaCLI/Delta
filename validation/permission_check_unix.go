//go:build !windows
// +build !windows

package validation

import (
	"fmt"
	"os"
	"os/user"
	"syscall"
)

// canWrite checks if current user can write to a path (Unix implementation)
func canWrite(path string, info os.FileInfo, currentUser *user.User) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return false
	}
	
	// Check owner
	if fmt.Sprint(stat.Uid) == currentUser.Uid {
		// Owner write permission
		return info.Mode().Perm()&0200 != 0
	}
	
	// Check group
	groups, err := currentUser.GroupIds()
	if err == nil {
		for _, gid := range groups {
			if fmt.Sprint(stat.Gid) == gid {
				// Group write permission
				return info.Mode().Perm()&0020 != 0
			}
		}
	}
	
	// Others write permission
	return info.Mode().Perm()&0002 != 0
}

// isRoot checks if running as root (Unix implementation)
func isRoot() bool {
	return os.Geteuid() == 0
}