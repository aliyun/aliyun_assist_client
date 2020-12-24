package perfmon

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/aliyun/aliyun_assist_client/agent/cgroup"
	"github.com/aliyun/aliyun_assist_client/agent/log"
	"github.com/aliyun/aliyun_assist_client/agent/util"
)

const (
	cgroup_cfg        = "/usr/local/share/aliyun-assist/config/cgroup"
	cgroup_name       = "aliyun_assist_cpu"
	default_cpu_limit = 15
)

func readUInt(str string) uint64 {
	val, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return 0
	}
	return val
}

func (p *procStat) UpdateSysStat() error {
	content, err := ioutil.ReadFile("/proc/stat")
	if err != nil {
		return err
	}
	lines := strings.Split(string(content), "\n")
	if len(lines) <= 1 {
		return errors.New("invalid stat file")
	}
	for _, line := range lines {
		parts := strings.Split(strings.TrimSpace(line), " ")
		if parts[0] == "cpu" {
			parts = parts[1:] // global cpu line has an extra space for some human somewhere
			Usr := readUInt(parts[1])
			Nice := readUInt(parts[2])
			Sys := readUInt(parts[3])
			Idle := readUInt(parts[4])
			Iowait := readUInt(parts[5])
			Irq := readUInt(parts[6])
			Softirq := readUInt(parts[7])
			Steal := readUInt(parts[8])
			p.systotal = Usr + Nice + Sys + Idle + Iowait + Irq + Softirq + Steal
			return nil
		}
	}
	return errors.New("invalid stat file")
}

func (p *procStat) UpdatePidStatInfo() error {
	lines, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/stat", p.pid))
	if err != nil {
		return err
	}
	fileStr := strings.TrimSpace(string(lines))
	err = p.procPidStatSplit(fileStr)
	if err != nil {
		return err
	}

	p.utime = readUInt(p.splitParts[13])
	p.stime = readUInt(p.splitParts[14])
	p.threads = readUInt(p.splitParts[19])
	p.rss = readUInt(p.splitParts[23])
	p.rss = p.rss * 4 //p.rss以页面及4K为单位
	return nil
}
func (p *procStat) procPidStatSplit(line string) error {
	line = strings.TrimSpace(line)
	partnum := 0
	strpos := 0
	start := 0
	inword := false
	space := " "[0]
	open := "("[0]
	close := ")"[0]
	groupchar := space

	for ; strpos < len(line); strpos++ {
		if inword {
			if line[strpos] == space && (groupchar == space || line[strpos-1] == groupchar) {
				p.splitParts[partnum] = line[start:strpos]
				partnum++
				start = strpos
				inword = false
			}
		} else {
			if line[strpos] == open {
				groupchar = close
				inword = true
				start = strpos
				strpos = strings.LastIndex(line, ")") - 1
				if strpos <= start { // if we can't parse this insane field, skip to the end
					strpos = len(line)
					inword = false
					return errors.New("Invalid proc stat string")
				}
			} else if line[strpos] != space {
				groupchar = space
				inword = true
				start = strpos
			}
		}
	}

	if inword {
		p.splitParts[partnum] = line[start:strpos]
		partnum++
	}

	for ; partnum < 52; partnum++ {
		p.splitParts[partnum] = ""
	}
	return nil
}

func getCpuLimit() int64 {
	// /usr/local/share/aliyun-assist/config/cgroup
	var cpu_limit int64 = default_cpu_limit
	c, err := ioutil.ReadFile(cgroup_cfg)
	if err == nil {
		i, err := strconv.ParseInt(strings.TrimSpace(string(c)), 10, 64)
		if err == nil {
			cpu_limit = i
			return cpu_limit
		}
	}
	os.MkdirAll(path.Dir(cgroup_cfg), os.ModePerm)
	ioutil.WriteFile(cgroup_cfg, []byte(fmt.Sprintf("%d", cpu_limit)), 0644)
	return cpu_limit
}

func InitCgroup() error {
	c, e := cgroup.NewManager(os.Getpid(), cgroup_name, "cpu")
	if e != nil {
		return e
	}
	cpuLimit := getCpuLimit()
	log.GetLogger().Infoln("cpuLimit=", cpuLimit)
	cfg := &cgroup.Config{
		CpuQuota: int64(1000 * cpuLimit),
	}
	return c.Set(cfg)
}

func GetAgentCpuLoadWithTop(times int) (error, float64) {
	topScript := "top -b -d 1 -p " + fmt.Sprintf("%d", os.Getpid()) + " -n " + fmt.Sprintf("%d", times)
	err, stdout, _ := util.ExeCmd(topScript)
	if err != nil {
		return err, 0.0
	}
	var cpulist []string
	lines := strings.Split(stdout, "\n")
	for _, value := range lines {
		line := strings.TrimSpace(value)
		if len(line) > 0 && (line[0] > '0' && line[0] <= '9') {
			parts := strings.Fields(strings.TrimSpace(line))
			if len(parts) == 12 {
				cpulist = append(cpulist, strings.TrimSpace(parts[8]))
			}
		}
	}
	log.GetLogger().Infoln("GetAgentCpuLoadWithTop:", cpulist)
	if len(cpulist) != times {
		log.GetLogger().Infoln("top result", stdout)
		return errors.New("top execute error"), 0.0
	}
	cpuloadTotal := 0.0
	for i := 0; i < len(cpulist); i++ {
		cpuload, _ := strconv.ParseFloat(cpulist[i], 32)
		cpuloadTotal += cpuload
	}
	return nil, cpuloadTotal / float64(times)
}

// func formatMem(num uint64) string {
// 	letter := string("K")

// 	num = num * 4
// 	if num >= 1000 {
// 		num = (num + 512) / 1024
// 		letter = "M"
// 		if num >= 10000 {
// 			num = (num + 512) / 1024
// 			letter = "G"
// 		}
// 	}
// 	return fmt.Sprintf("%d%s", num, letter)
// }

// func main() {
// 	fmt.Println(runtime.NumCPU())
// 	c := make(chan os.Signal)
// 	signal.Notify(c, os.Interrupt, os.Kill)

// 	pid := os.Getpid()
// 	if len(os.Args) > 1 {
// 		pid = int(readUInt(os.Args[1]))
// 	}
// 	fmt.Println("start monitor pid:", pid)
// 	fmt.Println("cpu(%)\t\t mem\t\t thread")
// 	mon := StartPerfmon(pid, 1, func(cpusage float64, memory uint64, threads uint64) {
// 		fmt.Println(cpusage, "\t\t", formatMem(memory), "\t\t", threads)
// 	})
// 	<-c
// 	fmt.Println("err:", mon.err)
// }
