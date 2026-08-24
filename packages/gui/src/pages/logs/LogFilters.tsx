import {sourceKey, sourceLabel} from "./formatters";
import type {LogOverview} from "./types";

type LogFiltersProps = {
    overview: LogOverview;
    search: string;
    level: string;
    application: string;
    source: string;
    from: string;
    to: string;
    showDates: boolean;
    onSearchChange(value: string): void;
    onLevelChange(value: string): void;
    onApplicationChange(value: string): void;
    onSourceChange(value: string): void;
    onFromChange(value: string): void;
    onToChange(value: string): void;
    onShowDatesChange(): void;
    onClear(): void;
};

export function LogFilters(props: LogFiltersProps) {
    const hasFilters = props.search !== "" || props.level !== "" || props.application !== "" || props.source !== "" || props.from !== "" || props.to !== "";

    return <>
        <section className="filter-bar" aria-label="Filtros de logs">
            <label className="compact-field search-control">
                <span className="sr-only">Buscar na mensagem</span>
                <input type="search" value={props.search} onChange={(event) => props.onSearchChange(event.target.value)} placeholder="Buscar na mensagem…"/>
            </label>
            <label className="compact-field">
                <span className="sr-only">Nível</span>
                <select value={props.level} onChange={(event) => props.onLevelChange(event.target.value)}>
                    <option value="">Todos os níveis</option>
                    {props.overview.byLevel.map((item) => <option key={item.level} value={item.level}>{item.level} ({item.count})</option>)}
                </select>
            </label>
            <label className="compact-field application-control">
                <span className="sr-only">Aplicação</span>
                <select value={props.application} onChange={(event) => props.onApplicationChange(event.target.value)}>
                    <option value="">Todas as aplicações</option>
                    {props.overview.applications.map((item) => <option key={item} value={item}>{item}</option>)}
                </select>
            </label>
            <label className="compact-field source-control">
                <span className="sr-only">Origem</span>
                <select value={props.source} onChange={(event) => props.onSourceChange(event.target.value)}>
                    <option value="">Todas as origens</option>
                    {props.overview.sources.map((item) => <option key={sourceKey(item)} value={sourceKey(item)}>{sourceLabel(item)}</option>)}
                </select>
            </label>
            <button type="button" className={props.showDates ? "button primary" : "button"} aria-expanded={props.showDates} onClick={props.onShowDatesChange}>
                Período{props.from || props.to ? " · ativo" : ""}
            </button>
            {hasFilters && <button type="button" className="button quiet" onClick={props.onClear}>Limpar</button>}
        </section>
        {props.showDates && <div className="date-filters">
            <span id="date-help">Horário local, limites inclusivos.</span>
            <label>De <input type="datetime-local" step="1" aria-describedby="date-help" value={props.from} max={props.to || undefined} onChange={(event) => props.onFromChange(event.target.value)}/></label>
            <label>Até <input type="datetime-local" step="1" aria-describedby="date-help" value={props.to} min={props.from || undefined} onChange={(event) => props.onToChange(event.target.value)}/></label>
        </div>}
    </>;
}
