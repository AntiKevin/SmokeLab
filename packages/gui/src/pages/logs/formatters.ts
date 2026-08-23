import type {LogSource} from "./types";

export function sourceKey(source: LogSource): string {
    return JSON.stringify([source.kind, source.name, source.id]);
}

export function sourceLabel(source: LogSource): string {
    const primary = source.name || source.id || "Origem sem nome";
    return source.kind ? `${primary} · ${source.kind}` : primary;
}

export function formatDate(value?: string | null): string {
    if (!value) return "—";
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return value;
    return new Intl.DateTimeFormat("pt-BR", {dateStyle: "short", timeStyle: "medium"}).format(date);
}

export function toRFC3339(value: string): string | undefined {
    if (!value) return undefined;
    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? undefined : date.toISOString();
}

export function errorMessage(error: unknown): string {
    if (error instanceof Error) return error.message;
    if (typeof error === "string") return error;
    if (error && typeof error === "object" && "message" in error && typeof error.message === "string") return error.message;
    return "Não foi possível ler os logs.";
}

export function formatJSONLossless(value: string): string {
    let output = "";
    let depth = 0;
    let inString = false;
    let escaped = false;

    for (const character of value) {
        if (inString) {
            output += character;
            if (escaped) escaped = false;
            else if (character === "\\") escaped = true;
            else if (character === '"') inString = false;
            continue;
        }

        if (character === '"') {
            inString = true;
            output += character;
        } else if (character === "{" || character === "[") {
            depth++;
            output += `${character}\n${"  ".repeat(depth)}`;
        } else if (character === "}" || character === "]") {
            depth = Math.max(0, depth - 1);
            output = `${output.trimEnd()}\n${"  ".repeat(depth)}${character}`;
        } else if (character === ",") {
            output += `,\n${"  ".repeat(depth)}`;
        } else if (character === ":") {
            output += ": ";
        } else if (!/\s/.test(character)) {
            output += character;
        }
    }

    return output;
}
