// ABOUTME: String processing utilities

pub fn truncate_string(s: &str, max_length: usize) -> String {
    if s.len() <= max_length {
        s.to_string()
    } else {
        format!("{}...", &s[..max_length.saturating_sub(3)])
    }
}

pub fn extract_snippet(content: &str, line_num: usize, context_lines: usize) -> String {
    let lines: Vec<&str> = content.lines().collect();
    let start = line_num.saturating_sub(context_lines);
    let end = (line_num + context_lines + 1).min(lines.len());

    if start >= lines.len() {
        return String::new();
    }

    lines[start..end].join("\n")
}

pub fn normalize_whitespace(s: &str) -> String {
    s.split_whitespace().collect::<Vec<_>>().join(" ")
}

pub fn is_likely_code(content: &str) -> bool {
    // Simple heuristic to determine if content looks like code
    let code_indicators = [
        "fn ", "def ", "function ", "class ", "import ", "use ",
        "var ", "let ", "const ", "if ", "for ", "while ",
        "{", "}", "(", ")", "[", "]", ";", "->", "=>"
    ];

    let content_lower = content.to_lowercase();
    code_indicators.iter().any(|&indicator| content_lower.contains(indicator))
}