import {useEffect, useRef} from "react";
import {formatDate, formatJSONLossless, sourceLabel} from "./formatters";
import {LevelBadge} from "./LevelBadge";
import type {LogRecord} from "./types";

export function LogDetails({log, onClose}: { log: LogRecord; onClose(): void }) {
    const heading = useRef<HTMLHeadingElement>(null);

    useEffect(() => {
        heading.current?.focus();
    }, [log.id]);

    return <aside className="log-details" aria-labelledby="detail-title" onKeyDown={(event) => { if (event.key === "Escape") onClose(); }}>
        <header>
            <div>
                <span className="detail-id">#{log.id}</span>
                <h2 id="detail-title" ref={heading} tabIndex={-1}>Detalhes</h2>
            </div>
            <button type="button" className="icon-button" aria-label="Fechar detalhes" onClick={onClose}>×</button>
        </header>
        <div className="detail-scroll">
            <LevelBadge level={log.level}/>
            <p className="detail-message">{log.message}</p>
            <dl>
                <div><dt>Timestamp</dt><dd>{formatDate(log.timestamp)}</dd></div>
                <div><dt>Capturado</dt><dd>{formatDate(log.capturedAt)}</dd></div>
                <div><dt>Origem</dt><dd>{sourceLabel(log.source)}</dd></div>
                <div><dt>ID origem</dt><dd>{log.source.id || "—"}</dd></div>
                <div><dt>Linha</dt><dd>{log.lineNumber || "—"}</dd></div>
            </dl>
            <section className="params-block" aria-labelledby="params-title">
                <h3 id="params-title">Params</h3>
                {log.params && log.params !== "{}" ? <pre>{formatJSONLossless(log.params)}</pre> : <span>Sem parâmetros adicionais.</span>}
            </section>
        </div>
    </aside>;
}
