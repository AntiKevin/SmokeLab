import {useEffect, useMemo, useRef, useState} from "react";
import {GetLogOverview, ListLogs} from "../../wailsjs/go/main/App";
import {logs as models} from "../../wailsjs/go/models";
import {LogDetails} from "./logs/LogDetails";
import {LogFilters} from "./logs/LogFilters";
import {errorMessage, formatDate, sourceKey, toRFC3339} from "./logs/formatters";
import {LogTable} from "./logs/LogTable";
import type {LogOverview, LogPage, LogRecord} from "./logs/types";
import "./LogsPage.css";

const PAGE_SIZE = 25;
const EMPTY_PAGE = new models.LogPage({items: [], total: 0, page: 1, pageSize: PAGE_SIZE, totalPages: 0});
const EMPTY_OVERVIEW = new models.LogOverview({total: 0, byLevel: [], applications: [], sources: []});

function LogsPage() {
    const [overview, setOverview] = useState<LogOverview>(EMPTY_OVERVIEW);
    const [pageData, setPageData] = useState<LogPage>(EMPTY_PAGE);
    const [searchInput, setSearchInput] = useState("");
    const [search, setSearch] = useState("");
    const [level, setLevel] = useState("");
    const [application, setApplication] = useState("");
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
                applications: application ? [application] : [],
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
    }, [search, level, application, selectedSource, from, to, page, refreshToken]);

    const hasFilters = searchInput !== "" || level !== "" || application !== "" || source !== "" || from !== "" || to !== "";

    function clearFilters() {
        setSearchInput("");
        setSearch("");
        setLevel("");
        setApplication("");
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

    function refresh() {
        setRefreshToken((value) => value + 1);
    }

    return <section className="page logs-page" aria-labelledby="logs-title">
        <header className="logs-header">
            <div>
                <h1 id="logs-title" className="page-title">Logs</h1>
                <p className="page-description">Registros persistidos localmente, do mais recente para o mais antigo.</p>
            </div>
            <div className="header-actions">
                <span className="logs-total"><strong>{overview.total.toLocaleString("pt-BR")}</strong> registros</span>
                <button type="button" className="button" disabled={loading || refreshing} onClick={refresh}>{refreshing ? "Atualizando…" : "Atualizar"}</button>
            </div>
        </header>

        <div className="compact-summary" aria-label="Resumo dos logs">
            <span><strong>{overview.sources.length}</strong> fontes</span>
            <span><strong>{overview.byLevel.length}</strong> níveis</span>
            <span className={overviewError ? "summary-error" : ""}>{overviewError || (overview.total > 0 ? `${formatDate(overview.oldestTimestamp)} → ${formatDate(overview.newestTimestamp)}` : "Banco vazio")}</span>
        </div>

        <LogFilters
            overview={overview}
            search={searchInput}
            level={level}
            application={application}
            source={source}
            from={from}
            to={to}
            showDates={showDates}
            onSearchChange={setSearchInput}
            onLevelChange={(value) => { setLevel(value); setPage(1); }}
            onApplicationChange={(value) => { setApplication(value); setPage(1); }}
            onSourceChange={(value) => { setSource(value); setPage(1); }}
            onFromChange={(value) => { setFrom(value); setPage(1); }}
            onToChange={(value) => { setTo(value); setPage(1); }}
            onShowDatesChange={() => setShowDates((value) => !value)}
            onClear={clearFilters}
        />

        <div className="logs-content">
            <LogTable
                overview={overview}
                page={pageData}
                selected={selected}
                loading={loading}
                refreshing={refreshing}
                error={error}
                hasFilters={hasFilters}
                onSelect={openDetails}
                onPageChange={setPage}
                onRetry={refresh}
                onClearFilters={clearFilters}
            />
            {selected && <LogDetails log={selected} onClose={closeDetails}/>}
        </div>
    </section>;
}

export default LogsPage;
