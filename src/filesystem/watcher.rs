// ABOUTME: File watching functionality for incremental updates

use crate::error::Result;
use notify::{Event, EventKind, RecommendedWatcher, RecursiveMode, Watcher, Config};
use std::collections::HashMap;
use std::path::PathBuf;
use std::sync::mpsc;
use std::sync::Arc;
use std::thread;
use std::time::Duration;

#[derive(Debug, Clone)]
pub enum FileChangeEvent {
    Created(PathBuf),
    Modified(PathBuf),
    Deleted(PathBuf),
}

pub struct FileWatcher {
    watcher: Option<RecommendedWatcher>,
    receiver: Option<mpsc::Receiver<FileChangeEvent>>,
    watched_directories: HashMap<PathBuf, RecursiveMode>,
}

impl FileWatcher {
    pub fn new() -> Result<Self> {
        Ok(Self {
            watcher: None,
            receiver: None,
            watched_directories: HashMap::new(),
        })
    }

    pub fn watch_directory(&mut self, path: PathBuf, recursive: bool) -> Result<()> {
        let (tx, rx) = mpsc::channel();
        let mode = if recursive {
            RecursiveMode::Recursive
        } else {
            RecursiveMode::NonRecursive
        };

        let config = Config::default();
        let mut watcher = RecommendedWatcher::new(
            move |res: notify::Result<Event>| {
                match res {
                    Ok(event) => {
                        for path in &event.paths {
                            let change_event = match event.kind {
                                EventKind::Create(_) => FileChangeEvent::Created(path.clone()),
                                EventKind::Modify(_) => FileChangeEvent::Modified(path.clone()),
                                EventKind::Remove(_) => FileChangeEvent::Deleted(path.clone()),
                                _ => return, // Ignore other events
                            };

                            // Ignore errors - if receiver is closed, we're shutting down
                            let _ = tx.send(change_event);
                        }
                    }
                    Err(e) => eprintln!("File watcher error: {:?}", e),
                }
            },
            config,
        )?;

        watcher.watch(&path, mode)?;
        self.watcher = Some(watcher);
        self.receiver = Some(rx);
        self.watched_directories.insert(path, mode);

        Ok(())
    }

    pub fn get_events(&mut self) -> Vec<FileChangeEvent> {
        let mut events = Vec::new();
        if let Some(ref receiver) = self.receiver {
            // Collect all available events with a timeout to avoid blocking
            while let Ok(event) = receiver.recv_timeout(Duration::from_millis(10)) {
                events.push(event);
            }
        }
        events
    }

    pub fn stop_watching(&mut self, path: &PathBuf) -> Result<()> {
        if let Some(ref mut watcher) = self.watcher {
            if let Some(&mode) = self.watched_directories.get(path) {
                watcher.unwatch(path)?;
                self.watched_directories.remove(path);
            }
        }
        Ok(())
    }

    pub fn stop_all(&mut self) -> Result<()> {
        if let Some(ref mut watcher) = self.watcher {
            for (path, _) in &self.watched_directories.clone() {
                watcher.unwatch(path)?;
            }
            self.watched_directories.clear();
        }
        self.watcher = None;
        self.receiver = None;
        Ok(())
    }
}

impl Default for FileWatcher {
    fn default() -> Self {
        Self::new().unwrap()
    }
}

impl Drop for FileWatcher {
    fn drop(&mut self) {
        let _ = self.stop_all();
    }
}

#[derive(Debug)]
pub struct DebouncedFileWatcher {
    events: Arc<std::sync::Mutex<Vec<FileChangeEvent>>>,
    _handle: thread::JoinHandle<()>,
}

impl DebouncedFileWatcher {
    pub fn new(watch_path: PathBuf, debounce_duration: Duration) -> Result<Self> {
        let events = Arc::new(std::sync::Mutex::new(Vec::new()));
        let events_clone = events.clone();

        let handle = thread::spawn(move || {
            let mut watcher = FileWatcher::new().unwrap();
            let mut pending_events: HashMap<PathBuf, (std::time::Instant, FileChangeEvent)> = HashMap::new();

            watcher.watch_directory(watch_path, true).unwrap();

            loop {
                let new_events = watcher.get_events();

                for event in new_events {
                    let path = match &event {
                        FileChangeEvent::Created(p) |
                        FileChangeEvent::Modified(p) |
                        FileChangeEvent::Deleted(p) => p.clone(),
                    };

                    pending_events.insert(path.clone(), (std::time::Instant::now(), event));
                }

                // Process events that have been pending longer than debounce duration
                let now = std::time::Instant::now();
                let mut ready_events = Vec::new();

                pending_events.retain(|_path, (timestamp, event)| {
                    if now.duration_since(*timestamp) > debounce_duration {
                        ready_events.push(event.clone());
                        false // Remove from pending
                    } else {
                        true // Keep in pending
                    }
                });

                if !ready_events.is_empty() {
                    let mut events_guard = events_clone.lock().unwrap();
                    events_guard.extend(ready_events);
                }

                thread::sleep(Duration::from_millis(100));
            }
        });

        Ok(Self {
            events,
            _handle: handle,
        })
    }

    pub fn get_events(&self) -> Vec<FileChangeEvent> {
        let mut events_guard = self.events.lock().unwrap();
        let events = events_guard.clone();
        events_guard.clear();
        events
    }
}