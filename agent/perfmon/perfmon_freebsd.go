package perfmon

import "errors"

func (p *procStat) UpdateSysStat() error {
	return nil
}

func (p *procStat) UpdatePidStatInfo() error {
	return nil
}

func InitCgroup() error {
	return nil
}

func GetAgentCpuLoadWithTop(times int) (error, float64) {
	return errors.New("not supported"), 0.0
}
