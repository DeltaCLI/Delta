package validation

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
)

// PermissionRequirement represents required permissions for a command
type PermissionRequirement struct {
	RequiresRoot       bool
	RequiresWrite      bool
	RequiresExecute    bool
	AffectedPaths      []string
	MissingPermissions []string
}

// CheckPermissionRequirements analyzes permission requirements for a command
func CheckPermissionRequirements(command string, ctx EnvironmentContext) PermissionRequirement {
	req := PermissionRequirement{
		AffectedPaths:      []string{},
		MissingPermissions: []string{},
	}
	
	// Check for sudo/root requirement
	if strings.Contains(command, "sudo") || strings.Contains(command, "su ") {
		req.RequiresRoot = true
	}
	
	// Check for operations that typically require root
	rootRequiredOps := []string{
		"systemctl", "service", "apt", "yum", "dnf", "pacman",
		"mount", "umount", "fdisk", "parted", "mkfs",
		"iptables", "firewall-cmd", "ufw",
		"useradd", "userdel", "usermod", "groupadd",
		"chown", "chmod.*[0-7][0-7][0-7]", // chmod with octal
	}
	
	for _, op := range rootRequiredOps {
		if strings.Contains(command, op) {
			req.RequiresRoot = true
			break
		}
	}
	
	// Check for write operations
	writeOps := []string{
		">", ">>", "rm", "mv", "cp", "mkdir", "touch", "sed -i", "tee",
	}
	
	for _, op := range writeOps {
		if strings.Contains(command, op) {
			req.RequiresWrite = true
			break
		}
	}
	
	// Extract potential file paths from command
	paths := extractPaths(command)
	req.AffectedPaths = paths
	
	// Check permissions for each path
	currentUser, _ := user.Current()
	for _, path := range paths {
		// Expand home directory
		if strings.HasPrefix(path, "~") {
			path = filepath.Join(os.Getenv("HOME"), path[1:])
		}
		
		// Check if path exists
		info, err := os.Stat(path)
		if err != nil {
			// Path doesn't exist, check parent directory
			parent := filepath.Dir(path)
			info, err = os.Stat(parent)
			if err != nil {
				continue
			}
			path = parent
		}
		
		// Check write permission
		if req.RequiresWrite && !canWrite(path, info, currentUser) {
			req.MissingPermissions = append(req.MissingPermissions, 
				"Write permission required for: "+path)
		}
		
		// Check if operation requires root for system paths
		if isSystemPath(path) && !isRoot() {
			req.RequiresRoot = true
			if req.RequiresWrite {
				req.MissingPermissions = append(req.MissingPermissions,
					"Root access required to modify system path: "+path)
			}
		}
	}
	
	// Check current user permissions
	if req.RequiresRoot && !isRoot() {
		req.MissingPermissions = append(req.MissingPermissions,
			"Command requires root privileges but running as regular user")
	}
	
	return req
}

// extractPaths attempts to extract file paths from a command
func extractPaths(command string) []string {
	paths := []string{}
	
	// Split command into tokens
	tokens := strings.Fields(command)
	
	for _, token := range tokens {
		// Skip flags and operators
		if strings.HasPrefix(token, "-") || isOperator(token) {
			continue
		}
		
		// Check if token looks like a path
		if strings.Contains(token, "/") || strings.HasPrefix(token, "~") {
			// Remove quotes if present
			token = strings.Trim(token, `"'`)
			paths = append(paths, token)
		}
	}
	
	return paths
}

// isOperator checks if a token is a shell operator
func isOperator(token string) bool {
	operators := []string{"|", "||", "&&", ";", ">", ">>", "<", "<<", "&"}
	for _, op := range operators {
		if token == op {
			return true
		}
	}
	return false
}

// canWrite checks if current user can write to a path
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

// isSystemPath checks if a path is a system directory
func isSystemPath(path string) bool {
	systemPaths := []string{
		"/etc", "/usr", "/bin", "/sbin", "/boot", "/dev", "/proc", "/sys",
		"/lib", "/lib64", "/var/log", "/opt", "/root",
	}
	
	for _, sysPath := range systemPaths {
		if strings.HasPrefix(path, sysPath) {
			return true
		}
	}
	
	return false
}

// isRoot checks if running as root
func isRoot() bool {
	return os.Geteuid() == 0
}