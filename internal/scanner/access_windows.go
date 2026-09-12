//go:build windows

package scanner

import (
	"os"
	"syscall"
	"time"
)

// accessTime returns the real last-access time of a file. On Windows the
// atime is stored in Win32FileAttributeData supplied by os.Lstat/Stat.
func accessTime(info os.FileInfo) time.Time {
	if st, ok := info.Sys().(*syscall.Win32FileAttributeData); ok {
		return time.Unix(0, st.LastAccessTime.Nanoseconds())
	}
	return info.ModTime()
}