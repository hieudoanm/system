#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use serde::Serialize;
use std::sync::Mutex;
use sysinfo::{Components, Disks, Networks, System};
use tauri::State;

pub struct AppState(pub Mutex<System>);

#[derive(Serialize)]
pub struct ProcessInfo {
    pub pid: u32,
    pub name: String,
    pub cpu: f32,
    pub mem_mb: u64,
}

#[derive(Serialize)]
pub struct Metrics {
    pub cpu_total: f32,
    pub cpu_per_core: Vec<f32>,
    pub ram_used_gb: f64,
    pub ram_total_gb: f64,
    pub ram_pct: f32,
    pub swap_used_gb: f64,
    pub swap_total_gb: f64,
    pub disk_read_kb: u64,
    pub disk_write_kb: u64,
    pub net_rx_kb: u64,
    pub net_tx_kb: u64,
    pub temp_cpu: Option<f32>,
    pub top_processes: Vec<ProcessInfo>,
}

#[tauri::command]
fn get_metrics(state: State<AppState>) -> Metrics {
    let mut sys = state.0.lock().unwrap();
    sys.refresh_all();

    let cpu_total = sys.global_cpu_usage();
    let cpu_per_core: Vec<f32> = sys.cpus().iter().map(|c| c.cpu_usage()).collect();

    let ram_used = sys.used_memory();
    let ram_total = sys.total_memory();
    let ram_pct = (ram_used as f32 / ram_total as f32) * 100.0;

    let swap_used = sys.used_swap();
    let swap_total = sys.total_swap();

    let disks = Disks::new_with_refreshed_list();
    let (disk_read_kb, disk_write_kb) = disks.iter().fold((0u64, 0u64), |acc, d| {
        (
            acc.0 + d.usage().read_bytes / 1024,
            acc.1 + d.usage().written_bytes / 1024,
        )
    });

    let mut networks = Networks::new_with_refreshed_list();
    networks.refresh(true);
    let (net_rx_kb, net_tx_kb) = networks.iter().fold((0u64, 0u64), |acc, (_, n)| {
        (acc.0 + n.received() / 1024, acc.1 + n.transmitted() / 1024)
    });

    let components = Components::new_with_refreshed_list();
    let temp_cpu: Option<f32> = components
        .iter()
        .find(|c| {
            c.label().to_lowercase().contains("cpu")
                || c.label().to_lowercase().contains("core")
        })
        .and_then(|c| c.temperature());

    let mut procs: Vec<ProcessInfo> = sys
        .processes()
        .values()
        .map(|p| ProcessInfo {
            pid: p.pid().as_u32(),
            name: p.name().to_string_lossy().to_string(),
            cpu: p.cpu_usage(),
            mem_mb: p.memory() / 1_048_576,
        })
        .collect();
    procs.sort_by(|a, b| b.cpu.partial_cmp(&a.cpu).unwrap());
    procs.truncate(10);

    Metrics {
        cpu_total,
        cpu_per_core,
        ram_used_gb: ram_used as f64 / 1_073_741_824.0,
        ram_total_gb: ram_total as f64 / 1_073_741_824.0,
        ram_pct,
        swap_used_gb: swap_used as f64 / 1_073_741_824.0,
        swap_total_gb: swap_total as f64 / 1_073_741_824.0,
        disk_read_kb,
        disk_write_kb,
        net_rx_kb,
        net_tx_kb,
        temp_cpu,
        top_processes: procs,
    }
}

fn main() {
    tauri::Builder::default()
        .manage(AppState(Mutex::new(System::new_all())))
        .invoke_handler(tauri::generate_handler![get_metrics])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}
