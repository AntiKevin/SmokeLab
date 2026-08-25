import {formatDate, formatHighlightValue, sourceLabel} from "./formatters";
import {LevelBadge, levelClass} from "./LevelBadge";
import type {LogOverview, LogPage, LogRecord} from "./types";

type LogTableProps = {
    overview: LogOverview;
    page: LogPage;
    selected: LogRecord | null;
    loading: boolean;
    refreshing: boolean;
    error: string;
    hasFilters: boolean;
    onSelect(log: LogRecord, row: HTMLTableRowElement): void;
    onPageChange(page: number): void;
    onRetry(): void;
    onClearFilters(): void;
};

export function LogTable(props: LogTableProps) {
    const rangeStart = props.page.total === 0 ? 0 : (props.page.page - 1) * props.page.pageSize + 1;
    const rangeEnd = Math.min(props.page.page * props.page.pageSize, props.page.total);
    const highlightColumns = props.page.highlightColumns ?? [];
    const tableMinWidth = 707 + highlightColumns.length * 180;

    return <div className="logs-table-panel">
        <div className="table-meta" aria-live="polite">
            <span>{props.refreshing ? "Atualizando…" : `${props.page.total.toLocaleString("pt-BR")} resultados`}</span>
            <span>{props.page.total > 0 ? `${rangeStart}–${rangeEnd}` : "—"}</span>
        </div>
        <div className="table-scroll">
            {props.loading ? <div className="content-state" role="status">Carregando logs…</div>
                : props.error ? <div className="content-state error-state" role="alert">
                    <strong>Falha ao carregar</strong>
                    <span>{props.error}</span>
                    <button type="button" className="button" onClick={props.onRetry}>Tentar novamente</button>
                </div>
                : props.page.items.length === 0 ? <div className="content-state">
                    <strong>{props.overview.total === 0 ? "Banco vazio" : "Nenhum resultado"}</strong>
                    <span>{props.overview.total === 0 ? "Importe logs pela CLI para começar." : "Tente remover alguns filtros."}</span>
                    {props.hasFilters && <button type="button" className="button" onClick={props.onClearFilters}>Limpar filtros</button>}
                </div>
                : <table className="logs-table" style={{minWidth: `${tableMinWidth}px`}}>
                    <caption className="sr-only">Lista de logs. Pressione Enter em uma linha para ver detalhes.</caption>
                    <thead><tr>
                        <th className="time-column">Horário</th>
                        <th className="level-column">Nível</th>
                        {highlightColumns.map((column) => <th className="highlight-column" key={column.path} title={column.label}>{column.label}</th>)}
                        <th>Mensagem</th>
                        <th className="source-column">Origem</th>
                    </tr></thead>
                    <tbody>{props.page.items.map((log) => <LogRow
                        key={log.id}
                        log={log}
                        highlightColumns={highlightColumns}
                        selected={props.selected?.id === log.id}
                        onSelect={props.onSelect}
                    />)}</tbody>
                </table>}
        </div>
        <nav className="pagination" aria-label="Paginação dos logs">
            <button type="button" className="button" disabled={props.page.page <= 1 || props.loading} onClick={() => props.onPageChange(Math.max(1, props.page.page - 1))}>Anterior</button>
            <span>{Math.max(1, props.page.page)} / {Math.max(1, props.page.totalPages)}</span>
            <button type="button" className="button" disabled={props.page.totalPages === 0 || props.page.page >= props.page.totalPages || props.loading} onClick={() => props.onPageChange(props.page.page + 1)}>Próxima</button>
        </nav>
    </div>;
}

function LogRow({log, highlightColumns, selected, onSelect}: {
    log: LogRecord;
    highlightColumns: NonNullable<LogPage["highlightColumns"]>;
    selected: boolean;
    onSelect: LogTableProps["onSelect"];
}) {
    function select(row: HTMLTableRowElement) {
        onSelect(log, row);
    }

    return <tr
        tabIndex={0}
        aria-selected={selected}
        className={selected ? "selected" : ""}
        onClick={(event) => select(event.currentTarget)}
        onKeyDown={(event) => {
            if (event.key === "Enter" || event.key === " ") {
                event.preventDefault();
                select(event.currentTarget);
            }
        }}
    >
        <td className="time-column"><time dateTime={log.timestamp}>{formatDate(log.timestamp)}</time></td>
        <td className="level-column"><LevelBadge level={log.level}/></td>
        {highlightColumns.map((column) => {
            const value = formatHighlightValue(log.highlightValues?.[column.path]);
            return <td className="truncate highlight-cell" data-label={column.label} key={column.path} title={value}>
                <span className={`highlight-badge ${levelClass(log.level)}`}>{value}</span>
            </td>;
        })}
        <td className="truncate message-column" title={log.message}>{log.message}</td>
        <td className="truncate muted-cell source-column" title={sourceLabel(log.source)}>{sourceLabel(log.source)}</td>
    </tr>;
}
