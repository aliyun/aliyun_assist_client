package perfmon

// import (
// 	"fmt"
// 	"os"
// 	"testing"
// 	"time"

// 	"github.com/stretchr/testify/assert"
// )

// func TestStartPerfmon(t *testing.T) {
// 	var t_perf *procStat
// 	var t_cpu_overlaod_count int = 0
// 	var t_mem_overlaod_count int = 0
// 	t_perf = StartPerfmon(os.Getpid(), 1, func(cpusage float64, memory uint64, threads uint64) {
// 		if cpusage >= CPU_LIMIT {
// 			t_cpu_overlaod_count += 1
// 		}
// 		if memory >= MEM_LIMIT {
// 			t_mem_overlaod_count += 1
// 		}
// 	})
// 	running := true
// 	for i := 0; i < 100; i++ {
// 		go func() {
// 			var k uint64 = 0
// 			var i uint64 = 0
// 			for i = 0; i < 100000000000; i++ {
// 				k = k + i
// 				if !running {
// 					return
// 				}
// 			}
// 		}()
// 	}

// 	time.Sleep(time.Duration(15) * time.Second)
// 	running = false
// 	time.Sleep(time.Duration(1) * time.Second)
// 	t_perf.StopPerfmon()
// 	fmt.Println(t_cpu_overlaod_count)
// 	assert.Equal(t, t_cpu_overlaod_count > 12, true)
// 	t_cpu_overlaod_count = 0
// 	t_mem_overlaod_count = 0
// 	t_perf = StartPerfmon(os.Getpid(), 1, func(cpusage float64, memory uint64) {
// 		if cpusage >= CPU_LIMIT {
// 			t_cpu_overlaod_count += 1
// 		}
// 		if memory >= MEM_LIMIT {
// 			t_mem_overlaod_count += 1
// 		}
// 	})
// 	time.Sleep(time.Duration(10) * time.Second)
// 	t_perf.StopPerfmon()
// 	fmt.Println(t_cpu_overlaod_count)
// 	assert.Equal(t, t_cpu_overlaod_count == 0, true)
// }
