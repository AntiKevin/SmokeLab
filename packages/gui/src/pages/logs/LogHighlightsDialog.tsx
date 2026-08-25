import {useEffect, useRef, useState} from "react";
import type {FormEvent} from "react";
import type {ApplicationHighlight, HighlightSetting} from "./types";

type LogHighlightsDialogProps = {
    configuration: ApplicationHighlight[];
    loading: boolean;
    saving: boolean;
    error: string;
    onClose(): void;
    onSave(settings: HighlightSetting[]): void;
};

export function LogHighlightsDialog(props: LogHighlightsDialogProps) {
    const [fieldPaths, setFieldPaths] = useState<Record<string, string>>({});
    const heading = useRef<HTMLHeadingElement>(null);

    useEffect(() => {
        setFieldPaths(Object.fromEntries(props.configuration.map((item) => [item.application, item.fieldPath || ""])));
    }, [props.configuration]);

    useEffect(() => {
        heading.current?.focus();
    }, []);

    function submit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        props.onSave(props.configuration.map((item) => ({
            application: item.application,
            fieldPath: fieldPaths[item.application] || "",
        })) as HighlightSetting[]);
    }

    return <div
        className="dialog-backdrop"
        onMouseDown={(event) => {
            if (!props.saving && event.target === event.currentTarget) props.onClose();
        }}
    >
        <section
            className="highlight-dialog"
            role="dialog"
            aria-modal="true"
            aria-labelledby="highlight-dialog-title"
            onKeyDown={(event) => {
                if (event.key === "Escape" && !props.saving) props.onClose();
            }}
        >
            <header>
                <div>
                    <h2 id="highlight-dialog-title" ref={heading} tabIndex={-1}>Colunas destacadas</h2>
                    <p>Escolha um campo dos parâmetros para cada aplicação.</p>
                </div>
                <button type="button" className="icon-button" aria-label="Fechar configuração" disabled={props.saving} onClick={props.onClose}>×</button>
            </header>

            <form onSubmit={submit}>
                <div className="highlight-dialog-content">
                    {props.loading ? <div className="dialog-state" role="status">Detectando campos…</div>
                        : props.configuration.length === 0 ? <div className="dialog-state">Nenhuma aplicação encontrada.</div>
                        : props.configuration.map((item) => <label className="highlight-setting" key={item.application}>
                            <span title={item.application}>{item.application}</span>
                            <select
                                value={fieldPaths[item.application] || ""}
                                disabled={props.saving || item.fields.length === 0}
                                onChange={(event) => setFieldPaths((current) => ({...current, [item.application]: event.target.value}))}
                            >
                                <option value="">{item.fields.length === 0 ? "Nenhum campo detectado" : "Sem destaque"}</option>
                                {item.fields.map((field) => <option key={field.path} value={field.path}>{field.label}</option>)}
                            </select>
                        </label>)}
                </div>
                {props.error && <p className="dialog-error" role="alert">{props.error}</p>}
                <footer>
                    <button type="button" className="button quiet" disabled={props.saving} onClick={props.onClose}>Cancelar</button>
                    <button type="submit" className="button primary" disabled={props.loading || props.saving || props.configuration.length === 0}>
                        {props.saving ? "Salvando…" : "Salvar"}
                    </button>
                </footer>
            </form>
        </section>
    </div>;
}
