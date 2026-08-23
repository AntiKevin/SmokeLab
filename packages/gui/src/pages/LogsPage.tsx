import {useEffect, useMemo, useRef, useState} from "react";
import {GetLogOverview, ListLogs} from "../../wailsjs/go/main/App";
import {logs as models} from "../../wailsjs/go/models";
import "./LogsPage.css";

type LogSource = models.LogSource;
type LogRecord = models.LogRecord;
type LogOverview = models.LogOverview;
type LogPage = models.LogPage;

const PAGE_SIZE = 25;
const EMPTY_PAGE = new models.LogPage({items: [], total: 0, page: 1, pageSize: PAGE_SIZE, totalPages: 0});
const EMPTY_OVERVIEW = new models.LogOverview({total: 0, byLevel: [], sources: []});

function sourceKey(source: LogSource): string {
    return JSON.stringify([source.kind, source.name, source.id]);
}

function sourceLabel(source: LogSource): string {
    const primary = source.name || source.id || "Origem sem nome";
    return source.kind ? `${primary} · ${source.kind}` : primary;
}

function formatDate(value?: string | null): string {
    if (!value) return "—";
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return value;
    return new Intl.DateTimeFormat("pt-BR", {dateStyle: "short", timeStyle: "medium"}).format(date);
}

function toRFC3339(value: string): string | undefined {
    if (!value) return undefined;
    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? undefined : date.toISOString();
}

function formatJSONLossless(value: string): string {
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

function errorMessage(error: unknown): string {
    if (error instanceof Error) return error.message;
    if (typeof error === "string") return error;
    if (error && typeof error === "object" && "message" in error && typeof error.message === "string") return error.message;
    return "Não foi possível ler os logs.";
}

function levelClass(level: string): string {
    const normalized = level.toLocaleLowerCase();
    if (["error", "fatal", "critical", "crit"].includes(normalized)) return "danger";
    if (["warn", "warning"].includes(normalized)) return "warning";
    if (["debug", "trace"].includes(normalized)) return "muted";
    return "info";
}

function LevelBadge({level}: { level: string }) {
    return <span className={`level-badge ${levelClass(level)}`}>{level}</span>;
}

function LogsPage() {
    const [overview, setOverview] = useState<LogOverview>(EMPTY_OVERVIEW);
    const [pageData, setPageData] = useState<LogPage>(EMPTY_PAGE);
    const [searchInput, setSearchInput] = useState("");
    const [search, setSearch] = useState("");
    const [level, setLevel] = useState("");
    const [source, setSource] = useState("");
    const [from, setFrom] = useState("");
    const [to, setTo] = useState("");
    const [page, setPage] = useState(1);
    const [selected, setSelected] = useState<LogRecord | null>(null);
    const [showDates, setShowDates] = useState(false);
    const [loading, setLoading] = useState(true);
    const [refreshing, setRefreshing] = useState(false);
    const [error, setError] = useState("");
    const [overviewError, setOverviewError] = useState("");
    const [refreshToken, setRefreshToken] = useState(0);
    const listRequestID = useRef(0);
    const overviewRequestID = useRef(0);
    const detailsHeading = useRef<HTMLHeadingElement>(null);
    const selectedRow = useRef<HTMLTableRowElement>(null);

    const selectedSource = useMemo(
        () => overview.sources.find((item) => sourceKey(item) === source),
        [overview.sources, source],
    );

    useEffect(() => {
        const timeout = window.setTimeout(() => {
            setSearch(searchInput.trim());
            setPage(1);
        }, 300);
        return () => window.clearTimeout(timeout);
    }, [searchInput]);

    useEffect(() => {
        const requestID = ++overviewRequestID.current;
        let active = true;
        setOverviewError("");
        GetLogOverview()
            .then((result) => {
                if (active && requestID === overviewRequestID.current) setOverview((result ?? EMPTY_OVERVIEW) as LogOverview);
            })
            .catch((reason) => {
                if (active && requestID === overviewRequestID.current) setOverviewError(errorMessage(reason));
            });
        return () => {
            active = false;
            overviewRequestID.current++;
        };
    }, [refreshToken]);

    useEffect(() => {
        const requestID = ++listRequestID.current;
        let active = true;
        setError("");
        if (pageData.items.length === 0) setLoading(true);
        else setRefreshing(true);

        const request = new models.ListLogsRequest({
            filter: {
                search,
                levels: level ? [level] : [],
                sources: selectedSource ? [selectedSource] : [],
                from: toRFC3339(from),
                to: toRFC3339(to),
            },
            page,
            pageSize: PAGE_SIZE,
            sortBy: "timestamp",
            sortDirection: "desc",
        });

        ListLogs(request)
            .then((result) => {
                if (!active || requestID !== listRequestID.current) return;
                const nextPage = (result ?? EMPTY_PAGE) as LogPage;
                setPageData(nextPage);
                setSelected((current) => current && nextPage.items.some((item) => item.id === current.id) ? current : null);
            })
            .catch((reason) => {
                if (!active || requestID !== listRequestID.current) return;
                setError(errorMessage(reason));
                setPageData(EMPTY_PAGE);
                setSelected(null);
            })
            .finally(() => {
                if (!active || requestID !== listRequestID.current) return;
                setLoading(false);
                setRefreshing(false);
            });

        return () => {
            active = false;
            listRequestID.current++;
        };
    // pageData is output, not a query input.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [search, level, selectedSource, from, to, page, refreshToken]);

    useEffect(() => {
        if (selected) detailsHeading.current?.focus();
    }, [selected]);

    const hasFilters = searchInput !== "" || level !== "" || source !== "" || from !== "" || to !== "";
    const rangeStart = pageData.total === 0 ? 0 : (pageData.page - 1) * pageData.pageSize + 1;
    const rangeEnd = Math.min(pageData.page * pageData.pageSize, pageData.total);

    function clearFilters() {
        setSearchInput("");
        setSearch("");
        setLevel("");
        setSource("");
        setFrom("");
        setTo("");
        setShowDates(false);
        setPage(1);
    }

    function openDetails(log: LogRecord, row: HTMLTableRowElement) {
        selectedRow.current = row;
        setSelected(log);
    }

    function closeDetails() {
        setSelected(null);
        window.requestAnimationFrame(() => selectedRow.current?.focus());
    }

    return (
        <section className="page logs-page" aria-labelledby="logs-title">
            <header className="logs-header">
                <div>
                    <h1 id="logs-title" className="page-title">Logs</h1>
                    <p className="page-description">Registros persistidos localmente, do mais recente para o mais antigo.</p>
                </div>
                <div className="header-actions">
                    <span className="logs-total"><strong>{overview.total.toLocaleString("pt-BR")}</strong> registros</span>
                    <button type="button" className="button" disabled={loading || refreshing} onClick={() => setRefreshToken((value) => value + 1)}>
                        {refreshing ? "Atualizando…" : "Atualizar"}
                    </button>
                </div>
            </header>

            <div className="compact-summary" aria-label="Resumo dos logs">
                <span><strong>{overview.sources.length}</strong> fontes</span>
                <span><strong>{overview.byLevel.length}</strong> níveis</span>
                <span className={overviewError ? "summary-error" : ""}>
                    {overviewError || (overview.total > 0 ? `${formatDate(overview.oldestTimestamp)} → ${formatDate(overview.newestTimestamp)}` : "Banco vazio")}
                </span>
            </div>

            <section className="filter-bar" aria-label="Filtros de logs">
                <label className="compact-field search-control">
                    <span className="sr-only">Buscar na mensagem</span>
                    <input type="search" value={searchInput} onChange={(event) => setSearchInput(event.target.value)} placeholder="Buscar na mensagem…"/>
                </label>

                <label className="compact-field">
                    <span className="sr-only">Nível</span>
                    <select value={level} onChange={(event) => { setLevel(event.target.value); setPage(1); }}>
                        <option value="">Todos os níveis</option>
                        {overview.byLevel.map((item) => <option key={item.level} value={item.level}>{item.level} ({item.count})</option>)}
                    </select>
                </label>

                <label className="compact-field source-control">
                    <span className="sr-only">Origem</span>
                    <select value={source} onChange={(event) => { setSource(event.target.value); setPage(1); }}>
                        <option value="">Todas as origens</option>
                        {overview.sources.map((item) => <option key={sourceKey(item)} value={sourceKey(item)}>{sourceLabel(item)}</option>)}
                    </select>
                </label>

                <button type="button" className={showDates ? "button primary" : "button"} aria-expanded={showDates} onClick={() => setShowDates((value) => !value)}>
                    Período{from || to ? " · ativo" : ""}
                </button>
                {hasFilters && <button type="button" className="button quiet" onClick={clearFilters}>Limpar</button>}
            </section>

            {showDates && (
                <div className="date-filters">
                    <span id="date-help">Horário local, limites inclusivos.</span>
                    <label>De <input type="datetime-local" step="1" aria-describedby="date-help" value={from} max={to || undefined} onChange={(event) => { setFrom(event.target.value); setPage(1); }}/></label>
                    <label>Até <input type="datetime-local" step="1" aria-describedby="date-help" value={to} min={from || undefined} onChange={(event) => { setTo(event.target.value); setPage(1); }}/></label>
                </div>
            )}

            <div className="logs-content">
                <div className="logs-table-panel">
                    <div className="table-meta" aria-live="polite">
                        <span>{refreshing ? "Atualizando…" : `${pageData.total.toLocaleString("pt-BR")} resultados`}</span>
                        <span>{pageData.total > 0 ? `${rangeStart}–${rangeEnd}` : "—"}</span>
                    </div>

                    <div className="table-scroll">
                        {loading ? (
                            <div className="content-state" role="status">Carregando logs…</div>
                        ) : error ? (
                            <div className="content-state error-state" role="alert">
                                <strong>Falha ao carregar</strong>
                                <span>{error}</span>
                                <button type="button" className="button" onClick={() => setRefreshToken((value) => value + 1)}>Tentar novamente</button>
                            </div>
                        ) : pageData.items.length === 0 ? (
                            <div className="content-state">
                                <strong>{overview.total === 0 ? "Banco vazio" : "Nenhum resultado"}</strong>
                                <span>{overview.total === 0 ? "Importe logs pela CLI para começar." : "Tente remover alguns filtros."}</span>
                                {hasFilters && <button type="button" className="button" onClick={clearFilters}>Limpar filtros</button>}
                            </div>
                        ) : (
                            <table className="logs-table">
                                <caption className="sr-only">Lista de logs. Pressione Enter em uma linha para ver detalhes.</caption>
                                <thead><tr><th>Horário</th><th>Nível</th><th>Mensagem</th><th>Origem</th></tr></thead>
                                <tbody>
                                {pageData.items.map((log) => (
                                    <tr
                                        key={log.id}
                                        tabIndex={0}
                                        aria-selected={selected?.id === log.id}
                                        className={selected?.id === log.id ? "selected" : ""}
                                        onClick={(event) => openDetails(log, event.currentTarget)}
                                        onKeyDown={(event) => {
                                            if (event.key === "Enter" || event.key === " ") {
                                                event.preventDefault();
                                                openDetails(log, event.currentTarget);
                                            }
                                        }}
                                    >
                                        <td><time dateTime={log.timestamp}>{formatDate(log.timestamp)}</time></td>
                                        <td><LevelBadge level={log.level}/></td>
                                        <td className="truncate" title={log.message}>{log.message}</td>
                                        <td className="truncate muted-cell" title={sourceLabel(log.source)}>{sourceLabel(log.source)}</td>
                                    </tr>
                                ))}
                                </tbody>
                            </table>
                        )}
                    </div>

                    <nav className="pagination" aria-label="Paginação dos logs">
                        <button type="button" className="button" disabled={pageData.page <= 1 || loading} onClick={() => setPage((value) => Math.max(1, value - 1))}>Anterior</button>
                        <span>{Math.max(1, pageData.page)} / {Math.max(1, pageData.totalPages)}</span>
                        <button type="button" className="button" disabled={pageData.totalPages === 0 || pageData.page >= pageData.totalPages || loading} onClick={() => setPage((value) => value + 1)}>Próxima</button>
                    </nav>
                </div>

                {selected && (
                    <aside className="log-details" aria-labelledby="detail-title" onKeyDown={(event) => { if (event.key === "Escape") closeDetails(); }}>
                        <header>
                            <div>
                                <span className="detail-id">#{selected.id}</span>
                                <h2 id="detail-title" ref={detailsHeading} tabIndex={-1}>Detalhes</h2>
                            </div>
                            <button type="button" className="icon-button" aria-label="Fechar detalhes" onClick={closeDetails}>×</button>
                        </header>
                        <div className="detail-scroll">
                            <LevelBadge level={selected.level}/>
                            <p className="detail-message">{selected.message}</p>
                            <dl>
                                <div><dt>Timestamp</dt><dd>{formatDate(selected.timestamp)}</dd></div>
                                <div><dt>Capturado</dt><dd>{formatDate(selected.capturedAt)}</dd></div>
                                <div><dt>Origem</dt><dd>{sourceLabel(selected.source)}</dd></div>
                                <div><dt>ID origem</dt><dd>{selected.source.id || "—"}</dd></div>
                                <div><dt>Linha</dt><dd>{selected.lineNumber || "—"}</dd></div>
                            </dl>
                            <section className="params-block" aria-labelledby="params-title">
                                <h3 id="params-title">Params</h3>
                                {selected.params && selected.params !== "{}"
                                    ? <pre>{formatJSONLossless(selected.params)}</pre>
                                    : <span>Sem parâmetros adicionais.</span>}
                            </section>
                        </div>
                    </aside>
                )}
            </div>
        </section>
    );
}

export default LogsPage;
