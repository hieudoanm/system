'use client';

import { useEffect, useState, useRef } from 'react';
import Link from 'next/link';
import { invoke } from '@tauri-apps/api/core';
import {
  AreaChart,
  Area,
  YAxis,
  XAxis,
  Tooltip,
  ResponsiveContainer,
  CartesianGrid,
} from 'recharts';
import { NameType } from 'recharts/types/component/DefaultTooltipContent';

// ─── Types ───────────────────────────────────────────────────────────────────

type Metrics = {
  cpu_total: number;
  cpu_per_core: number[];
  ram_used_gb: number;
  ram_total_gb: number;
  ram_pct: number;
  swap_used_gb: number;
  swap_total_gb: number;
  disk_read_kb: number;
  disk_write_kb: number;
  net_rx_kb: number;
  net_tx_kb: number;
  temp_cpu: number | null;
  top_processes: { pid: number; name: string; cpu: number; mem_mb: number }[];
};

type HistoryPoint = { cpu: number; ram: number };

// ─── Constants ───────────────────────────────────────────────────────────────

const MAX = 60;

const C = {
  error: '#e57373',
  warning: '#ffb74d',
  success: '#81c784',
  gold: '#c9a84c',
  ram: '#34d399',
  bg: '#1a1108',
  text: '#dca54c',
  muted: '#6b5a3e',
  grid: 'rgba(255,255,255,0.04)',
  border: 'rgba(255,255,255,0.06)',
} as const;

const MOCK_PROCESSES = [
  'Chrome',
  'node',
  'Xcode',
  'Slack',
  'Spotify',
  'kernel_task',
  'WindowServer',
  'Safari',
];

// ─── Helpers ─────────────────────────────────────────────────────────────────

const statusColor = (v: number) =>
  v > 85 ? C.error : v > 65 ? C.warning : C.success;

const statusBadgeClass = (v: number) =>
  v > 85 ? 'badge-error' : v > 65 ? 'badge-warning' : 'badge-success';

const makeMock = (): Metrics => ({
  cpu_total: 20 + Math.random() * 60,
  cpu_per_core: Array.from({ length: 8 }, () => 10 + Math.random() * 80),
  ram_used_gb: 6 + Math.random() * 4,
  ram_total_gb: 16,
  ram_pct: 40 + Math.random() * 40,
  swap_used_gb: 0.5 + Math.random(),
  swap_total_gb: 4,
  disk_read_kb: Math.floor(Math.random() * 500),
  disk_write_kb: Math.floor(Math.random() * 300),
  net_rx_kb: Math.floor(Math.random() * 1000),
  net_tx_kb: Math.floor(Math.random() * 500),
  temp_cpu: 45 + Math.random() * 35,
  top_processes: MOCK_PROCESSES.map((name, i) => ({
    pid: 1000 + i * 37,
    name,
    cpu: Math.random() * 25,
    mem_mb: Math.floor(200 + Math.random() * 1200),
  })),
});

const toHistoryPoint = (d: Metrics): HistoryPoint => ({
  cpu: +d.cpu_total.toFixed(1),
  ram: +d.ram_pct.toFixed(1),
});

// ─── Component ───────────────────────────────────────────────────────────────

const AppPage = () => {
  const [latest, setLatest] = useState<Metrics | null>(null);
  const [history, setHistory] = useState<HistoryPoint[]>([]);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const chartWrapRef = useRef<HTMLDivElement>(null);

  const pushHistory = (point: HistoryPoint) =>
    setHistory((prev) => {
      const next = [...prev, point];
      return next.length > MAX ? next.slice(-MAX) : next;
    });

  useEffect(() => {
    const poll = async () => {
      try {
        const data = await invoke<Metrics>('get_metrics');
        setLatest(data);
        pushHistory(toHistoryPoint(data));
      } catch {
        const mock = makeMock();
        setLatest(mock);
        pushHistory(toHistoryPoint(mock));
      }
    };

    poll();
    intervalRef.current = setInterval(poll, 1000);
    return () => {
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, []);

  const tempVal = latest?.temp_cpu ?? null;

  const cards = [
    {
      label: 'CPU',
      val: `${latest?.cpu_total.toFixed(1) ?? 0}%`,
      raw: latest?.cpu_total ?? 0,
      showBar: true,
    },
    {
      label: 'RAM',
      val: `${latest?.ram_used_gb.toFixed(1) ?? '–'} / ${latest?.ram_total_gb.toFixed(1) ?? '–'} GB`,
      raw: latest?.ram_pct ?? 0,
      showBar: true,
    },
    {
      label: 'Disk I/O',
      val: `R ${latest?.disk_read_kb ?? 0} KB/s`,
      raw: 0,
      showBar: false,
    },
    {
      label: 'Temp',
      val: tempVal != null ? `${tempVal.toFixed(1)} °C` : 'N/A',
      raw: tempVal ?? 0,
      showBar: tempVal != null,
    },
  ];

  return (
    <main className="bg-base-300 text-base-content min-h-screen p-6 font-mono select-none">
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <span className="text-primary text-[10px] tracking-widest uppercase">
          system monitor
        </span>
        <div className="flex items-center gap-4">
          <Link
            href="/version"
            className="text-[10px] tabular-nums transition-opacity hover:opacity-60"
            style={{ color: C.muted }}>
            version
          </Link>
          <span className="text-[10px] tabular-nums" style={{ color: C.muted }}>
            {new Date().toLocaleTimeString()}
          </span>
        </div>
      </div>

      {/* Metric cards */}
      <div className="mb-4 grid grid-cols-2 gap-3 lg:grid-cols-4">
        {cards.map((c) => (
          <div
            key={c.label}
            className="card bg-base-200 border-base-100/10 border">
            <div className="card-body p-4">
              <p
                className="mb-1 text-[10px] tracking-widest uppercase"
                style={{ color: C.muted }}>
                {c.label}
              </p>
              <p
                className="text-lg font-medium tabular-nums"
                style={{ color: c.raw > 0 ? statusColor(c.raw) : C.text }}>
                {c.val}
              </p>
              {c.showBar && c.raw > 0 && (
                <progress
                  className={`progress mt-2 h-1 ${
                    c.raw > 85
                      ? 'progress-error'
                      : c.raw > 65
                        ? 'progress-warning'
                        : 'progress-success'
                  }`}
                  value={c.raw}
                  max={100}
                />
              )}
            </div>
          </div>
        ))}
      </div>

      {/* CPU cores */}
      {latest?.cpu_per_core && (
        <div className="card bg-base-200 border-base-100/10 mb-4 border">
          <div className="card-body p-4">
            <p
              className="mb-3 text-[10px] tracking-widest uppercase"
              style={{ color: C.muted }}>
              cores ({latest.cpu_per_core.length})
            </p>
            <div className="flex h-10 items-end gap-1">
              {latest.cpu_per_core.map((v, i) => (
                <div key={i} className="flex flex-1 flex-col items-center">
                  <div
                    className="w-full rounded-sm transition-all duration-300"
                    style={{
                      height: `${Math.max(2, v * 0.4)}px`,
                      background: statusColor(v),
                    }}
                  />
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* History chart */}
      <div className="card bg-base-200 border-base-100/10 mb-4 border">
        <div className="card-body p-4">
          <p
            className="mb-3 text-[10px] tracking-widest uppercase"
            style={{ color: C.muted }}>
            60s history
          </p>
          <div ref={chartWrapRef} style={{ width: '100%', height: 160 }}>
            <ResponsiveContainer width="99%" height={160}>
              <AreaChart
                data={history}
                margin={{ top: 4, right: 0, left: -20, bottom: 0 }}>
                <defs>
                  <linearGradient id="gcpu" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor={C.gold} stopOpacity={0.3} />
                    <stop offset="95%" stopColor={C.gold} stopOpacity={0} />
                  </linearGradient>
                  <linearGradient id="gram" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor={C.ram} stopOpacity={0.3} />
                    <stop offset="95%" stopColor={C.ram} stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke={C.grid} />
                <XAxis hide />
                <YAxis
                  domain={[0, 100]}
                  tick={{ fontSize: 10, fill: C.muted }}
                  axisLine={false}
                  tickLine={false}
                  tickFormatter={(v) => v + '%'}
                />
                <Tooltip
                  contentStyle={{
                    background: C.bg,
                    border: `1px solid ${C.border}`,
                    fontSize: 11,
                    borderRadius: 6,
                    color: C.text,
                  }}
                  labelStyle={{ display: 'none' }}
                  formatter={(v: any, k: NameType | undefined) => [
                    `${v}%`,
                    k?.toString().toUpperCase(),
                  ]}
                />
                <Area
                  type="monotone"
                  dataKey="cpu"
                  stroke={C.gold}
                  fill="url(#gcpu)"
                  strokeWidth={1.5}
                  dot={false}
                  isAnimationActive={false}
                />
                <Area
                  type="monotone"
                  dataKey="ram"
                  stroke={C.ram}
                  fill="url(#gram)"
                  strokeWidth={1.5}
                  dot={false}
                  isAnimationActive={false}
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>
          <div
            className="mt-2 flex gap-4 text-[11px]"
            style={{ color: C.muted }}>
            <span className="flex items-center gap-1.5">
              <span
                className="inline-block h-2 w-2 rounded-full"
                style={{ background: C.gold }}
              />
              CPU
            </span>
            <span className="flex items-center gap-1.5">
              <span
                className="inline-block h-2 w-2 rounded-full"
                style={{ background: C.ram }}
              />
              RAM
            </span>
          </div>
        </div>
      </div>

      {/* Process table */}
      <div className="card bg-base-200 border-base-100/10 border">
        <div className="card-body p-4">
          <p
            className="mb-3 text-[10px] tracking-widest uppercase"
            style={{ color: C.muted }}>
            top processes
          </p>
          <table className="table-xs table">
            <thead>
              <tr style={{ color: C.muted }} className="border-base-100/10">
                <th className="bg-transparent font-normal">process</th>
                <th className="bg-transparent text-right font-normal">pid</th>
                <th className="bg-transparent text-right font-normal">cpu</th>
                <th className="bg-transparent text-right font-normal">mem</th>
              </tr>
            </thead>
            <tbody>
              {(latest?.top_processes ?? []).map((p) => (
                <tr
                  key={p.pid}
                  className="border-base-100/10 hover:bg-base-100/5">
                  <td className="max-w-xs truncate" style={{ color: C.text }}>
                    {p.name}
                  </td>
                  <td className="text-right" style={{ color: C.muted }}>
                    {p.pid}
                  </td>
                  <td className="text-right tabular-nums">
                    <span
                      className={`badge badge-xs ${statusBadgeClass(p.cpu)}`}>
                      {p.cpu.toFixed(1)}%
                    </span>
                  </td>
                  <td
                    className="text-right tabular-nums"
                    style={{ color: C.muted }}>
                    {p.mem_mb} MB
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </main>
  );
};

export default AppPage;
