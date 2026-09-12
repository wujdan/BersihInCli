//go:build !windows && !linux && !darwin && !freebsd && !netbsd && !openbsd && !dragonfly

package scanner

import (
	"os"
	"time"
)

// accessTime falls back to modtime on platforms without an easy atime hook.
func accessTime(info os.FileInfo) time.Time {
	return info.ModTime()
}