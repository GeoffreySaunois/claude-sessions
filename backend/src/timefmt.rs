//! Minimal RFC3339 formatting for file modification times, in UTC. The frontend
//! parses `lastActive` as a `Date`, for which a `Z`-suffixed UTC instant is an
//! equivalent, unambiguous RFC3339 representation. Kept dependency-free (no
//! chrono) to minimize the supply-chain surface.

/// rfc3339 renders a Unix instant (seconds + sub-second nanoseconds) as an
/// RFC3339 UTC timestamp, e.g. `2026-06-29T13:40:05.123456789Z`. Trailing zero
/// nanoseconds are omitted, matching Go's `time.Time` JSON encoding shape.
pub fn rfc3339(unix_secs: i64, nanos: u32) -> String {
    let days = unix_secs.div_euclid(86_400);
    let secs_of_day = unix_secs.rem_euclid(86_400);
    let (year, month, day) = civil_from_days(days);
    let hour = secs_of_day / 3600;
    let minute = (secs_of_day % 3600) / 60;
    let second = secs_of_day % 60;

    let mut out = format!("{year:04}-{month:02}-{day:02}T{hour:02}:{minute:02}:{second:02}");
    if nanos > 0 {
        // Trim trailing zeros from the fractional part.
        let frac = format!("{nanos:09}");
        let trimmed = frac.trim_end_matches('0');
        out.push('.');
        out.push_str(trimmed);
    }
    out.push('Z');
    out
}

/// civil_from_days converts a count of days since the Unix epoch into a
/// (year, month, day) Gregorian date. Algorithm from Howard Hinnant's
/// `chrono`-compatible `civil_from_days`.
fn civil_from_days(z: i64) -> (i64, u32, u32) {
    let z = z + 719_468;
    let era = if z >= 0 { z } else { z - 146_096 } / 146_097;
    let doe = z - era * 146_097; // [0, 146096]
    let yoe = (doe - doe / 1460 + doe / 36_524 - doe / 146_096) / 365; // [0, 399]
    let y = yoe + era * 400;
    let doy = doe - (365 * yoe + yoe / 4 - yoe / 100); // [0, 365]
    let mp = (5 * doy + 2) / 153; // [0, 11]
    let d = (doy - (153 * mp + 2) / 5 + 1) as u32; // [1, 31]
    let m = if mp < 10 { mp + 3 } else { mp - 9 } as u32; // [1, 12]
    let year = if m <= 2 { y + 1 } else { y };
    (year, m, d)
}
