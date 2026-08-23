function levelClass(level: string): string {
    const normalized = level.toLocaleLowerCase();
    if (["error", "fatal", "critical", "crit"].includes(normalized)) return "danger";
    if (["warn", "warning"].includes(normalized)) return "warning";
    if (["debug", "trace"].includes(normalized)) return "muted";
    return "info";
}

export function LevelBadge({level}: { level: string }) {
    return <span className={`level-badge ${levelClass(level)}`}>{level}</span>;
}
