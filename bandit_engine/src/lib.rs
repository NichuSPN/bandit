use ignore::WalkBuilder;
use rayon::prelude::*;
use std::ffi::{CStr, CString};
use std::os::raw::c_char;
use std::path::Path;
use std::sync::atomic::{AtomicUsize, Ordering};
use std::sync::Mutex;

#[no_mangle]
pub extern "C" fn rust_fast_search(
    query_ptr: *const c_char,
    dir_ptr: *const c_char,
    ignores_ptr: *const c_char,
) -> *mut c_char {
    if query_ptr.is_null() || dir_ptr.is_null() {
        return CString::new("Empty query or target directory.").unwrap().into_raw();
    }

    let query_str = unsafe { CStr::from_ptr(query_ptr).to_string_lossy() };
    let dir_str = unsafe { CStr::from_ptr(dir_ptr).to_string_lossy() };

    let ignores: Vec<String> = if !ignores_ptr.is_null() {
        let s = unsafe { CStr::from_ptr(ignores_ptr).to_string_lossy() };
        s.split(',')
            .map(|i| i.trim().to_lowercase())
            .filter(|i| !i.is_empty())
            .collect()
    } else {
        vec![
            ".git".into(),
            "node_modules".into(),
            "target".into(),
            "dist".into(),
            "build".into(),
            "venv".into(),
        ]
    };

    let terms: Vec<String> = query_str
        .split('|')
        .map(|t| t.trim().to_lowercase())
        .filter(|t| !t.is_empty())
        .collect();

    if terms.is_empty() {
        return CString::new("Empty search query.").unwrap().into_raw();
    }

    let search_path = Path::new(dir_str.as_ref());
    let mut builder = WalkBuilder::new(search_path);
    builder.hidden(false); // Include hidden files (e.g. .env.example, .github, etc.)
    builder.git_ignore(true); // Respect .gitignore rules!
    builder.filter_entry(move |entry| {
        let name = entry.file_name().to_string_lossy().to_lowercase();
        if name == ".git" {
            return false;
        }
        for ig in &ignores {
            if name == *ig || name.ends_with(ig) {
                return false;
            }
        }
        true
    });

    let entries: Vec<_> = builder
        .build()
        .filter_map(Result::ok)
        .filter(|e| e.file_type().map_or(false, |ft| ft.is_file()))
        .collect();

    let results = Mutex::new(Vec::new());
    let match_counter = AtomicUsize::new(0);

    entries.par_iter().for_each(|entry| {
        if match_counter.load(Ordering::Relaxed) >= 50 {
            return;
        }

        let path = entry.path();
        let path_str = path.to_string_lossy().to_lowercase();

        if path_str.ends_with(".min.js")
            || path_str.ends_with(".min.css")
            || path_str.ends_with(".map")
            || path_str.ends_with(".bundle.js")
            || path_str.ends_with(".lock")
            || path_str.ends_with("lock.json")
            || path_str.ends_with("lock.yaml")
        {
            return;
        }

        if let Ok(content) = std::fs::read_to_string(path) {
            for (idx, line) in content.lines().enumerate() {
                if match_counter.load(Ordering::Relaxed) >= 50 {
                    break;
                }

                let line_lower = line.to_lowercase();
                if terms.iter().any(|term| line_lower.contains(term)) {
                    let count = match_counter.fetch_add(1, Ordering::Relaxed);
                    if count >= 50 {
                        break;
                    }

                    let trimmed = line.trim();
                    let truncated = if trimmed.chars().count() > 150 {
                        format!("{}...", trimmed.chars().take(150).collect::<String>())
                    } else {
                        trimmed.to_string()
                    };

                    let match_str = format!("{}:{}: {}", path.display(), idx + 1, truncated);
                    if let Ok(mut lock) = results.lock() {
                        lock.push(match_str);
                    }
                }
            }
        }
    });

    let mut lock = results.into_inner().unwrap();
    lock.sort();

    if lock.is_empty() {
        let res = format!("No occurrences of '{}' found.", query_str);
        CString::new(res).unwrap().into_raw()
    } else {
        let res = format!("Search matches for '{}':\n{}", query_str, lock.join("\n"));
        CString::new(res).unwrap().into_raw()
    }
}

#[no_mangle]
pub extern "C" fn rust_compute_diff(
    old_ptr: *const c_char,
    new_ptr: *const c_char,
    label_ptr: *const c_char,
) -> *mut c_char {
    let old_str = if old_ptr.is_null() { "" } else { unsafe { CStr::from_ptr(old_ptr).to_str().unwrap_or("") } };
    let new_str = if new_ptr.is_null() { "" } else { unsafe { CStr::from_ptr(new_ptr).to_str().unwrap_or("") } };
    let label = if label_ptr.is_null() { "file" } else { unsafe { CStr::from_ptr(label_ptr).to_str().unwrap_or("file") } };

    let diff = similar::TextDiff::from_lines(old_str, new_str);
    let mut out = String::new();
    out.push_str(&format!("--- a/{}\n+++ b/{}\n", label, label));

    for change in diff.iter_all_changes() {
        let sign = match change.tag() {
            similar::ChangeTag::Delete => "-",
            similar::ChangeTag::Insert => "+",
            similar::ChangeTag::Equal => " ",
        };
        out.push_str(&format!("{}{}", sign, change));
    }

    CString::new(out).unwrap().into_raw()
}

#[no_mangle]
pub extern "C" fn rust_generate_repomap(
    dir_ptr: *const c_char,
    max_tokens: usize,
) -> *mut c_char {
    let dir_str = if dir_ptr.is_null() {
        "."
    } else {
        unsafe { CStr::from_ptr(dir_ptr).to_str().unwrap_or(".") }
    };

    let max_chars = if max_tokens == 0 { 4096 } else { max_tokens * 4 };
    let search_path = Path::new(dir_str);

    let mut builder = WalkBuilder::new(search_path);
    builder.hidden(false);
    builder.git_ignore(true);
    builder.filter_entry(|entry| {
        let name = entry.file_name().to_string_lossy().to_lowercase();
        if name == ".git" || name == "node_modules" || name == "vendor" || name == "target" || name == "dist" || name == "build" || name == ".bandit" || name == ".claude" {
            return false;
        }
        true
    });

    let mut out = String::from("## Repository Outline (Tree-Sitter / Rust Symbol Map)\n");

    for entry in builder.build().filter_map(Result::ok) {
        if out.len() >= max_chars {
            out.push_str("\n... [Repository Map truncated to stay within VRAM/token budget] ...\n");
            break;
        }

        if !entry.file_type().map_or(false, |ft| ft.is_file()) {
            continue;
        }

        let path = entry.path();
        let ext = path.extension().and_then(|s| s.to_str()).unwrap_or("").to_lowercase();
        if !matches!(ext.as_str(), "go" | "rs" | "py" | "ts" | "tsx" | "js" | "jsx" | "c" | "cpp" | "h" | "hpp") {
            continue;
        }

        if let Ok(content) = std::fs::read_to_string(path) {
            let mut symbols = Vec::new();
            for (idx, line) in content.lines().enumerate() {
                let trimmed = line.trim();
                if trimmed.is_empty() || trimmed.starts_with("//") || trimmed.starts_with('#') || trimmed.starts_with("/*") {
                    continue;
                }

                let is_match = match ext.as_str() {
                    "go" => trimmed.starts_with("type ") || trimmed.starts_with("func ") || trimmed.starts_with("package "),
                    "py" => trimmed.starts_with("class ") || trimmed.starts_with("def "),
                    "ts" | "tsx" | "js" | "jsx" => trimmed.starts_with("export ") || trimmed.starts_with("class ") || trimmed.starts_with("function ") || trimmed.starts_with("interface ") || trimmed.starts_with("type "),
                    "rs" => trimmed.starts_with("pub ") || trimmed.starts_with("struct ") || trimmed.starts_with("fn ") || trimmed.starts_with("enum ") || trimmed.starts_with("trait ") || trimmed.starts_with("mod "),
                    "c" | "cpp" | "h" | "hpp" => trimmed.starts_with("typedef ") || trimmed.starts_with("struct ") || trimmed.starts_with("class ") || trimmed.starts_with("enum "),
                    _ => false,
                };

                if is_match {
                    let clean = if let Some(idx_brace) = trimmed.find('{') {
                        trimmed[..idx_brace].trim()
                    } else {
                        trimmed
                    };
                    symbols.push(format!("  - L{}: {}", idx + 1, clean));
                }
            }

            if !symbols.is_empty() {
                let rel_path = path.strip_prefix(search_path).unwrap_or(path).display().to_string();
                let block = format!("\n### {}\n{}\n", rel_path, symbols.join("\n"));
                if out.len() + block.len() > max_chars {
                    out.push_str("\n... [Repository Map truncated to stay within VRAM/token budget] ...\n");
                    break;
                }
                out.push_str(&block);
            }
        }
    }

    CString::new(out).unwrap().into_raw()
}

#[no_mangle]
pub extern "C" fn rust_free_string(ptr: *mut c_char) {
    if !ptr.is_null() {
        unsafe {
            let _ = CString::from_raw(ptr);
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_rust_compute_diff() {
        let old_cstr = CString::new("line1\nline2").unwrap();
        let new_cstr = CString::new("line1\nline_new\nline2").unwrap();
        let label_cstr = CString::new("test.txt").unwrap();

        let ptr = rust_compute_diff(old_cstr.as_ptr(), new_cstr.as_ptr(), label_cstr.as_ptr());
        let res = unsafe { CStr::from_ptr(ptr).to_string_lossy().to_string() };
        rust_free_string(ptr);

        assert!(res.contains("+line_new"));
    }

    #[test]
    fn test_rust_generate_repomap() {
        let dir_cstr = CString::new(".").unwrap();
        let ptr = rust_generate_repomap(dir_cstr.as_ptr(), 1024);
        let res = unsafe { CStr::from_ptr(ptr).to_string_lossy().to_string() };
        rust_free_string(ptr);

        assert!(res.contains("Repository Outline"));
    }
}
