use serde::{Deserialize, Serialize};
use std::sync::Mutex;
use tauri::{
    menu::{Menu, MenuItem},
    tray::TrayIconBuilder,
    Manager,
};

#[derive(Debug, Serialize, Deserialize, Clone)]
pub struct BackendState {
    pub running: bool,
    pub port: u16,
    pub pid: Option<u32>,
}

pub struct SidecarState {
    pub child: Mutex<Option<u32>>,
}

#[tauri::command]
fn get_backend_status(state: tauri::State<SidecarState>) -> BackendState {
    let child = state.child.lock().unwrap();
    match *child {
        Some(pid) => BackendState {
            running: true,
            port: 18790,
            pid: Some(pid),
        },
        None => BackendState {
            running: false,
            port: 0,
            pid: None,
        },
    }
}

#[tauri::command]
fn get_backend_port() -> u16 {
    18790
}

#[tauri::command]
fn get_app_version() -> String {
    env!("CARGO_PKG_VERSION").to_string()
}

pub fn run() {
    tauri::Builder::default()
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_updater::Builder::new().build())
        .plugin(tauri_plugin_process::init())
        .plugin(tauri_plugin_notification::init())
        .plugin(tauri_plugin_autostart::init(
            tauri_plugin_autostart::MacosLauncher::LaunchAgent,
            Some(vec![]),
        ))
        .manage(SidecarState {
            child: Mutex::new(None),
        })
        .setup(|app| {
            let show_item = MenuItem::with_id(app, "show", "Show OctAi", true, None::<&str>)?;
            let check_update_item =
                MenuItem::with_id(app, "check_update", "Check for Updates", true, None::<&str>)?;
            let quit_item = MenuItem::with_id(app, "quit", "Quit", true, None::<&str>)?;

            let menu = Menu::with_items(app, &[&show_item, &check_update_item, &quit_item])?;

            let _tray = TrayIconBuilder::new()
                .icon(app.default_window_icon().unwrap().clone())
                .menu(&menu)
                .tooltip("OctAi")
                .on_menu_event(|app, event| match event.id.as_ref() {
                    "show" => {
                        if let Some(window) = app.get_webview_window("main") {
                            let _ = window.show();
                            let _ = window.set_focus();
                        }
                    }
                    "check_update" => {
                        log::info!("Checking for updates...");
                    }
                    "quit" => {
                        app.exit(0);
                    }
                    _ => {}
                })
                .build(app)?;

            spawn_backend(app.handle().clone());

            Ok(())
        })
        .invoke_handler(tauri::generate_handler![
            get_backend_status,
            get_backend_port,
            get_app_version,
        ])
        .run(tauri::generate_context!())
        .expect("error while running OctAi");
}

fn spawn_backend(app: tauri::AppHandle) {
    use tauri_plugin_shell::ShellExt;

    tauri::async_runtime::spawn(async move {
        use tauri_plugin_shell::process::CommandEvent;

        let sidecar_command = match app.shell().sidecar("octai-backend") {
            Ok(cmd) => cmd,
            Err(e) => {
                log::error!("Failed to create sidecar command: {}", e);
                return;
            }
        };

        let sidecar_command = sidecar_command.args(["--port", "18790", "--console"]);

        let (mut rx, child) = match sidecar_command.spawn() {
            Ok(result) => result,
            Err(e) => {
                log::error!("Failed to spawn backend: {}", e);
                return;
            }
        };

        let pid = child.pid();
        let state = app.state::<SidecarState>();
        *state.child.lock().unwrap() = Some(pid);
        log::info!("Backend spawned with PID: {}", pid);

        while let Some(event) = rx.recv().await {
            match event {
                CommandEvent::Stdout(line) => {
                    log::info!("[backend] {}", String::from_utf8_lossy(&line));
                }
                CommandEvent::Stderr(line) => {
                    log::error!("[backend] {}", String::from_utf8_lossy(&line));
                }
                CommandEvent::Terminated(status) => {
                    log::info!("Backend exited with status: {:?}", status);
                    let state = app.state::<SidecarState>();
                    *state.child.lock().unwrap() = None;
                }
                CommandEvent::Error(err) => {
                    log::error!("Backend error: {}", err);
                }
                _ => {}
            }
        }
    });
}
