//go:build !windows

package disk

import (
	"os"
	"syscall"
)

// FreeSizeBytes returns free and total bytes of the volume containing path.
func FreeSizeBytes(path string) (free, total int64, err error) {
	if _, e := os.Stat(path); e != nil {
		return 0, 0, e
	}
	var st syscall.Statfs_t
	if e := syscall.Statfs(path, &st); e != nil {
		return 0, 0, e
	}
	return int64(st.Bavail) * int64(st.Bsize), int64(st.Blocks) * int64(st.Bsize), nil
}