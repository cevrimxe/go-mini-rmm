package collector

import (
	"runtime"

	"github.com/shirou/gopsutil/v3/disk"
)

func DiskPercent() (float64, error) {
	path := "/"
	if runtime.GOOS == "windows" {
		path = "C:\\"
	}
	usage, err := disk.Usage(path)
	if err != nil {
		return 0, err
	}
	return usage.UsedPercent, nil
}
