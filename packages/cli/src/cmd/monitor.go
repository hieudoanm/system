/*
Copyright © 2026 system-cli
*/
package cmd

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/spf13/cobra"
)

/* ─── Styles ───────────────────────────────────────────────────────────── */

var (
	styleTitle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#c9a84c")).Bold(true)
	styleMuted   = lipgloss.NewStyle().Foreground(lipgloss.Color("#6b5a3e"))
	styleSuccess = lipgloss.NewStyle().Foreground(lipgloss.Color("#81c784"))
	styleWarning = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffb74d"))
	styleError   = lipgloss.NewStyle().Foreground(lipgloss.Color("#e57373"))
	styleGold    = lipgloss.NewStyle().Foreground(lipgloss.Color("#c9a84c"))
	styleRAM     = lipgloss.NewStyle().Foreground(lipgloss.Color("#34d399"))
	styleCard    = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#2a1f0e")).
			Padding(0, 1)
)

func statusStyle(v float64) lipgloss.Style {
	if v > 85 {
		return styleError
	}
	if v > 65 {
		return styleWarning
	}
	return styleSuccess
}

/* ─── Sparkline ────────────────────────────────────────────────────────── */

const sparkChars = "▁▂▃▄▅▆▇█"

func sparkline(data []float64, width int) string {
	if len(data) == 0 {
		return strings.Repeat(" ", width)
	}
	if len(data) > width {
		data = data[len(data)-width:]
	}

	var sb strings.Builder
	for _, v := range data {
		idx := int(math.Round(v / 100 * float64(len(sparkChars)-1)))
		idx = max(0, min(len(sparkChars)-1, idx))
		sb.WriteRune(rune(sparkChars[idx]))
	}

	return strings.Repeat(" ", width-len(data)) + sb.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

/* ─── Bar ─────────────────────────────────────────────────────────────── */

func bar(pct float64, width int) string {
	filled := int(math.Round(pct / 100 * float64(width)))
	filled = max(0, min(width, filled))
	return statusStyle(pct).Render(
		strings.Repeat("█", filled) + strings.Repeat("░", width-filled),
	)
}

/* ─── Metrics ─────────────────────────────────────────────────────────── */

type ProcessInfo struct {
	PID   int32
	Name  string
	CPU   float64
	MemMB uint64
}

type Metrics struct {
	CPUTotal    float64
	CPUPerCore  []float64
	RAMUsedGB   float64
	RAMTotalGB  float64
	RAMPct      float64
	DiskReadKB  uint64
	DiskWriteKB uint64
	NetRxKB     uint64
	NetTxKB     uint64
	Uptime      uint64
	TopProcs    []ProcessInfo
}

func gatherMetrics() (Metrics, error) {
	var m Metrics

	if p, _ := cpu.Percent(0, false); len(p) > 0 {
		m.CPUTotal = p[0]
	}
	m.CPUPerCore, _ = cpu.Percent(0, true)

	if vm, err := mem.VirtualMemory(); err == nil {
		m.RAMUsedGB = float64(vm.Used) / 1e9
		m.RAMTotalGB = float64(vm.Total) / 1e9
		m.RAMPct = vm.UsedPercent
	}

	if ioMap, err := disk.IOCounters(); err == nil {
		for _, d := range ioMap {
			m.DiskReadKB += d.ReadBytes / 1024
			m.DiskWriteKB += d.WriteBytes / 1024
		}
	}

	if netStats, _ := net.IOCounters(false); len(netStats) > 0 {
		m.NetRxKB = netStats[0].BytesRecv / 1024
		m.NetTxKB = netStats[0].BytesSent / 1024
	}

	m.Uptime, _ = host.Uptime()

	procs, _ := process.Processes()
	for _, p := range procs {
		name, _ := p.Name()
		cpuPct, _ := p.CPUPercent()
		memInfo, _ := p.MemoryInfo()

		var memMB uint64
		if memInfo != nil {
			memMB = memInfo.RSS / 1048576
		}

		m.TopProcs = append(m.TopProcs, ProcessInfo{
			PID: p.Pid, Name: name, CPU: cpuPct, MemMB: memMB,
		})
	}

	sort.Slice(m.TopProcs, func(i, j int) bool {
		return m.TopProcs[i].CPU > m.TopProcs[j].CPU
	})
	if len(m.TopProcs) > 10 {
		m.TopProcs = m.TopProcs[:10]
	}

	return m, nil
}

/* ─── Bubbletea Model ─────────────────────────────────────────────────── */

type tickMsg time.Time
type metricsMsg Metrics

type model struct {
	metrics    Metrics
	cpuHistory []float64
	ramHistory []float64
	width      int
	err        error
}

func (m model) Init() tea.Cmd {
	return tea.Batch(fetchMetrics(), tickEvery())
}

func fetchMetrics() tea.Cmd {
	return func() tea.Msg {
		m, err := gatherMetrics()
		if err != nil {
			return err
		}
		return metricsMsg(m)
	}
}

func tickEvery() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width

	case metricsMsg:
		m.metrics = Metrics(msg)

		m.cpuHistory = append(m.cpuHistory, m.metrics.CPUTotal)
		m.ramHistory = append(m.ramHistory, m.metrics.RAMPct)

	case tickMsg:
		return m, tea.Batch(fetchMetrics(), tickEvery())

	case error:
		m.err = msg
	}

	return m, nil
}

/* ─── View (v2 FIX) ───────────────────────────────────────────────────── */

func (m model) View() tea.View {
	if m.err != nil {
		return tea.NewView(styleError.Render("error: " + m.err.Error()))
	}

	uptime := time.Duration(m.metrics.Uptime) * time.Second

	content := fmt.Sprintf(
		"%s\nCPU: %.1f%%\nRAM: %.1f%%\nUptime: %s\n\nPress q to quit",
		styleTitle.Render("SYSTEM MONITOR"),
		m.metrics.CPUTotal,
		m.metrics.RAMPct,
		uptime.Truncate(time.Second),
	)

	return tea.NewView(content)
}

/* ─── Command ─────────────────────────────────────────────────────────── */

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Run the monitor operation for the system app",
	RunE: func(cmd *cobra.Command, args []string) error {
		p := tea.NewProgram(model{width: 100})
		_, err := p.Run()
		return err
	},
}

func init() {
	rootCmd.AddCommand(monitorCmd)
}
