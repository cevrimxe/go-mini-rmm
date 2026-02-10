package collector

import (
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
)

func CPUPercent() (float64, error) {
	percents, err := cpu.Percent(time.Second, false)
	if err != nil {
		return 0, err
	}
	if len(percents) == 0 {
		return 0, nil
	}
	return percents[0], nil
}
