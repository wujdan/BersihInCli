//go:build linux

package scanner

import (
	"os"
	"syscall"
	"time"
)

// accessTime returns the real last-access time of a file.
func accessTime(info os.FileInfo) time.Time {
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		return st.Atim
	}
	return info.ModTime()
}