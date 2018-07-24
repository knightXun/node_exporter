package collector

import (
	"syscall"
)

type DiskUsage struct {
	Name 		string
	Size 		uint64
	Used 		uint64
	Free        uint64
	Avail 		uint64
	MountedPath string
	Inodes      uint64
	InodesFree  uint64
}
func UpdateFsStats(path string) (ds DiskUsage, err error) {
	var s syscall.Statfs_t
	if err = syscall.Statfs(path, &s); err != nil {
		return DiskUsage{}, err
	}
	total := uint64(s.Frsize) * s.Blocks
	free := uint64(s.Frsize) * s.Bfree
	avail := uint64(s.Frsize) * s.Bavail
	inodes := uint64(s.Files)
	inodesFree := uint64(s.Ffree)
	res := DiskUsage{
		Name: path,
		Size: total,
		Used: total - avail,
		Free: free,
		Avail: avail,
		Inodes: inodes,
		InodesFree: inodesFree,
	}
	return res, nil
}

