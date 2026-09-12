//go:build windows

package disk

import (
	"golang.org/x/sys/windows"
)

// FreeSizeBytes returns free and total bytes of the volume containing path.
func FreeSizeBytes(path string) (free, total int64, err error) {
	p, perr := windows.UTF16PtrFromString(path)
	if perr != nil {
		return 0, 0, perr
	}
	var freeAvail, totalBytes, totalFree uint64
	if werr := windows.GetDiskFreeSpaceEx(p, &freeAvail, &totalBytes, &totalFree); werr != nil {
		return 0, 0, werr
	}
	return int64(freeAvail), int64(totalBytes), nil
}