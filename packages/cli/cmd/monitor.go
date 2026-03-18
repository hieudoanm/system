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

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// ─── Styles ──────────────────────────────────────────────────────────────────

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

// ─── Sparkline ───────────────────────────────────────────────────────────────

const sparkChars = "▁▂▃▄▅▆▇█"

func sparkline(data []float64, width int) string {
	if len(data) == 0 {
		return strings.Repeat(" ", width)
	}
	// trim or pad to width
	trimmed := data
	if len(trimmed) > width {
		trimmed = trimmed[len(trimmed)-width:]
	}
	var sb strings.Builder
	for _, v := range trimmed {
		idx := int(math.Round(v / 100 * float64(len(sparkChars)-1)))
		idx = max(0, min(len(sparkChars)-1, idx))
		sb.WriteRune(rune(sparkChars[idx]))
	}
	// left-pad with spaces if shorter than width
	pad := width - len(trimmed)
	return strings.Repeat(" ", pad) + sb.String()
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

// ─── Bar ─────────────────────────────────────────────────────────────────────

func bar(pct float64, width int) string {
	filled := int(math.Round(pct / 100 * float64(width)))
	filled = max(0, min(width, filled))
	empty := width - filled
	b := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	return statusStyle(pct).Render(b)
}

// ─── Metrics ─────────────────────────────────────────────────────────────────

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
	SwapUsedGB  float64
	SwapTotalGB float64
	DiskReadKB  uint64
	DiskWriteKB uint64
	NetRxKB     uint64
	NetTxKB     uint64
	TempCPU     float64
	HasTemp     bool
	Uptime      uint64
	TopProcs    []ProcessInfo
}

func gatherMetrics() (Metrics, error) {
	var m Metrics

	// CPU
	pcts, err := cpu.Percent(0, false)
	if err == nil && len(pcts) > 0 {
		m.CPUTotal = pcts[0]
	}
	cores, err := cpu.Percent(0, true)
	if err == nil {
		m.CPUPerCore = cores
	}

	// RAM
	vm, err := mem.VirtualMemory()
	if err == nil {
		m.RAMUsedGB = float64(vm.Used) / 1073741824
		m.RAMTotalGB = float64(vm.Total) / 1073741824
		m.RAMPct = vm.UsedPercent
	}

	// Swap
	sw, err := mem.SwapMemory()
	if err == nil {
		m.SwapUsedGB = float64(sw.Used) / 1073741824
		m.SwapTotalGB = float64(sw.Total) / 1073741824
	}

	// Disk I/O
	ioMap, err := disk.IOCounters()
	if err == nil {
		for _, d := range ioMap {
			m.DiskReadKB += d.ReadBytes / 1024
			m.DiskWriteKB += d.WriteBytes / 1024
		}
	}

	// Network
	netStats, err := net.IOCounters(false)
	if err == nil && len(netStats) > 0 {
		m.NetRxKB = netStats[0].BytesRecv / 1024
		m.NetTxKB = netStats[0].BytesSent / 1024
	}

	// Uptime
	uptime, err := host.Uptime()
	if err == nil {
		m.Uptime = uptime
	}

	// Top processes
	procs, err := process.Processes()
	if err == nil {
		var list []ProcessInfo
		for _, p := range procs {
			name, _ := p.Name()
			cpuPct, _ := p.CPUPercent()
			memInfo, _ := p.MemoryInfo()
			var memMB uint64
			if memInfo != nil {
				memMB = memInfo.RSS / 1048576
			}
			list = append(list, ProcessInfo{
				PID:   p.Pid,
				Name:  name,
				CPU:   cpuPct,
				MemMB: memMB,
			})
		}
		sort.Slice(list, func(i, j int) bool {
			return list[i].CPU > list[j].CPU
		})
		if len(list) > 10 {
			list = list[:10]
		}
		m.TopProcs = list
	}

	return m, nil
}

// ─── Bubbletea model ─────────────────────────────────────────────────────────

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
		metrics, err := gatherMetrics()
		if err != nil {
			return err
		}
		return metricsMsg(metrics)
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
		if len(m.cpuHistory) > 60 {
			m.cpuHistory = m.cpuHistory[len(m.cpuHistory)-60:]
		}
		if len(m.ramHistory) > 60 {
			m.ramHistory = m.ramHistory[len(m.ramHistory)-60:]
		}

	case tickMsg:
		return m, tea.Batch(fetchMetrics(), tickEvery())

	case error:
		m.err = msg
	}

	return m, nil
}

func (m model) View() string {
	if m.err != nil {
		return styleError.Render("error: "+m.err.Error()) + "\n"
	}

	met := m.metrics
	w := m.width
	if w == 0 {
		w = 100
	}
	innerW := w - 4 // account for card padding + border

	var sb strings.Builder

	// ── Header ──
	uptime := time.Duration(met.Uptime) * time.Second
	h := int(uptime.Hours())
	min := int(uptime.Minutes()) % 60
	s := int(uptime.Seconds()) % 60

	sb.WriteString(styleTitle.Render("SYSTEM MONITOR"))
	sb.WriteString("  ")
	sb.WriteString(styleMuted.Render(
		fmt.Sprintf("uptime %02d:%02d:%02d  %s", h, min, s, time.Now().Format("15:04:05")),
	))
	sb.WriteString("\n\n")

	// ── Metric cards row ──
	cardW := (innerW - 6) / 4

	cpuCard := renderMetricCard("CPU", fmt.Sprintf("%.1f%%", met.CPUTotal), met.CPUTotal, cardW)
	ramCard := renderMetricCard("RAM",
		fmt.Sprintf("%.1f / %.1f GB", met.RAMUsedGB, met.RAMTotalGB),
		met.RAMPct, cardW)
	diskCard := renderMetricCard("DISK R/W",
		fmt.Sprintf("%d / %d KB/s", met.DiskReadKB, met.DiskWriteKB),
		0, cardW)
	netCard := renderMetricCard("NETWORK",
		fmt.Sprintf("↑%d ↓%d KB/s", met.NetTxKB, met.NetRxKB),
		0, cardW)

	sb.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, cpuCard, " ", ramCard, " ", diskCard, " ", netCard))
	sb.WriteString("\n\n")

	// ── CPU cores ──
	if len(met.CPUPerCore) > 0 {
		sb.WriteString(styleMuted.Render("CORES"))
		sb.WriteString("\n")
		coreBarW := max(4, (innerW/len(met.CPUPerCore))-2)
		var coreRow strings.Builder
		for i, v := range met.CPUPerCore {
			coreRow.WriteString(fmt.Sprintf("%s%s ",
				styleMuted.Render(fmt.Sprintf("%2d ", i)),
				bar(v, coreBarW),
			))
		}
		sb.WriteString(styleCard.Render(coreRow.String()))
		sb.WriteString("\n\n")
	}

	// ── Sparklines ──
	sparkW := innerW - 10
	sb.WriteString(styleMuted.Render("60s HISTORY"))
	sb.WriteString("\n")
	cpuSpark := styleGold.Render(sparkline(m.cpuHistory, sparkW))
	ramSpark := styleRAM.Render(sparkline(m.ramHistory, sparkW))
	sparkContent := fmt.Sprintf("%s %s\n%s %s",
		styleGold.Render("CPU"),
		cpuSpark,
		styleRAM.Render("RAM"),
		ramSpark,
	)
	sb.WriteString(styleCard.Render(sparkContent))
	sb.WriteString("\n\n")

	// ── Process table ──
	sb.WriteString(styleMuted.Render("TOP PROCESSES"))
	sb.WriteString("\n")

	nameW := max(16, innerW-40)
	header := fmt.Sprintf("%-*s %6s %7s %8s",
		nameW, "PROCESS", "PID", "CPU%", "MEM MB")
	sb.WriteString(styleMuted.Render(header))
	sb.WriteString("\n")
	sb.WriteString(styleMuted.Render(strings.Repeat("─", innerW)))
	sb.WriteString("\n")

	for _, p := range met.TopProcs {
		name := p.Name
		if len(name) > nameW {
			name = name[:nameW-1] + "…"
		}
		cpuStr := statusStyle(p.CPU).Render(fmt.Sprintf("%6.1f%%", p.CPU))
		line := fmt.Sprintf("%-*s %6d %s %8d",
			nameW, name, p.PID, cpuStr, p.MemMB)
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(styleMuted.Render("press q to quit"))

	return sb.String()
}

func renderMetricCard(label, val string, pct float64, width int) string {
	barW := max(4, width-2)
	var content string
	if pct > 0 {
		content = fmt.Sprintf("%s\n%s\n%s",
			styleMuted.Render(label),
			statusStyle(pct).Render(val),
			bar(pct, barW),
		)
	} else {
		content = fmt.Sprintf("%s\n%s",
			styleMuted.Render(label),
			styleGold.Render(val),
		)
	}
	return styleCard.Width(width).Render(content)
}

// ─── Command ─────────────────────────────────────────────────────────────────

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Live system monitor (CPU, RAM, disk, network, processes)",
	RunE: func(cmd *cobra.Command, args []string) error {
		p := tea.NewProgram(
			model{width: 100},
			tea.WithAltScreen(),
		)
		_, err := p.Run()
		return err
	},
}

func init() {
	rootCmd.AddCommand(monitorCmd)
}
