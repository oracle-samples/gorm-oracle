The fix is in the `buildOracleDefault` function. Two changes are needed:

**1. Line 683-684 — "already quoted strings" branch:** The original code passed through pre-quoted strings without escaping their interior. The fix strips the outer quotes, escapes any single quotes inside using Oracle's `''` doubling convention, then re-wraps:

```go
// Handle already quoted strings — strip outer quotes and re-escape
if strings.HasPrefix(defaultValue, "'") && strings.HasSuffix(defaultValue, "'") {
    inner := defaultValue[1 : len(defaultValue)-1]
    return "DEFAULT '" + strings.ReplaceAll(inner, "'", "''") + "'"
}
```

**2. Line 687 — unquoted string fallback:** The original code directly interpolated the value. The fix escapes single quotes before interpolation:

```go
// Handle string values that need quoting — escape single quotes
return "DEFAULT '" + strings.ReplaceAll(defaultValue, "'", "''") + "'"
```

With these changes, a malicious default like `'; DROP TABLE users; --` becomes the safe SQL literal `DEFAULT '''; DROP TABLE users; --'` — the single quote is doubled, keeping it inside the string literal instead of breaking out.