interface HomePageProps {
    onOpenLogs: () => void;
}

function HomePage({onOpenLogs}: HomePageProps) {
    return (
        <section className="page home-page" aria-labelledby="home-title">
            <header className="home-header">
                <p className="terminal-kicker">$ smokelab workspace</p>
                <h1 id="home-title" className="home-title">Ferramentas locais para investigar o que seu software está fazendo.</h1>
                <p className="home-copy">
                    Escolha uma ferramenta no menu. Tudo roda localmente e compartilha o mesmo motor do SmokeLab.
                </p>
            </header>

            <section className="tools-section" aria-labelledby="tools-title">
                <span id="tools-title" className="section-label">Ferramentas disponíveis</span>
                <button type="button" className="tool-card" onClick={onOpenLogs}>
                    <span className="tool-icon" aria-hidden="true">&gt;_</span>
                    <span className="tool-content">
                        <strong>Explorador de logs</strong>
                        <span>Busque, filtre e inspecione os registros persistidos pela CLI.</span>
                    </span>
                    <span className="tool-action">abrir →</span>
                </button>
            </section>
        </section>
    );
}

export default HomePage;
