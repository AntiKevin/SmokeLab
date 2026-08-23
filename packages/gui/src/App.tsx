import {useEffect, useRef, useState} from "react";
import TopBar from "./components/TopBar";
import HomePage from "./pages/HomePage";
import LogsPage from "./pages/LogsPage";
import "./App.css";

type Page = "home" | "logs";

const navigation: Array<{ id: Page; label: string; shortcut: string }> = [
    {id: "home", label: "Início", shortcut: "01"},
    {id: "logs", label: "Logs", shortcut: "02"},
];

function App() {
    const [page, setPage] = useState<Page>("home");
    const [menuOpen, setMenuOpen] = useState(false);
    const menu = useRef<HTMLDivElement>(null);

    useEffect(() => {
        function closeMenu(event: PointerEvent) {
            if (!menu.current?.contains(event.target as Node)) setMenuOpen(false);
        }

        function closeMenuWithKeyboard(event: KeyboardEvent) {
            if (event.key === "Escape") setMenuOpen(false);
        }

        document.addEventListener("pointerdown", closeMenu);
        document.addEventListener("keydown", closeMenuWithKeyboard);
        return () => {
            document.removeEventListener("pointerdown", closeMenu);
            document.removeEventListener("keydown", closeMenuWithKeyboard);
        };
    }, []);

    return (
        <div className="app-shell">
            <TopBar/>

            <div className="app-layout">
                <header className="main-menu">
                    <div className="menu-brand">
                        <span className="brand-mark" aria-hidden="true">SL</span>
                        <div>
                            <strong>SmokeLab</strong>
                        </div>
                    </div>

                    <div className="menu-dropdown" ref={menu}>
                        <button
                            type="button"
                            className={menuOpen ? "menu-trigger open" : "menu-trigger"}
                            aria-expanded={menuOpen}
                            aria-haspopup="menu"
                            onClick={() => setMenuOpen((value) => !value)}
                        >
                            Menu <span aria-hidden="true">{menuOpen ? "↑" : "↓"}</span>
                        </button>

                        {menuOpen && (
                            <nav className="menu-navigation" aria-label="Navegação principal" role="menu">
                                <span className="menu-label">Navegação</span>
                                {navigation.map((item) => (
                                    <button
                                        key={item.id}
                                        type="button"
                                        role="menuitem"
                                        className={page === item.id ? "menu-item active" : "menu-item"}
                                        aria-current={page === item.id ? "page" : undefined}
                                        onClick={() => {
                                            setPage(item.id);
                                            setMenuOpen(false);
                                        }}
                                    >
                                        <span className="menu-index" aria-hidden="true">{item.shortcut}</span>
                                        <span>{item.label}</span>
                                    </button>
                                ))}
                            </nav>
                        )}
                    </div>

                    <div className="current-location" aria-live="polite">
                        <span>Workspace</span>
                        <strong>{navigation.find((item) => item.id === page)?.label}</strong>
                    </div>

                    <div className="menu-footer">
                        <span className="status-dot" aria-hidden="true"/>
                        local
                    </div>
                </header>

                <main className="page-container">
                    {page === "home"
                        ? <HomePage onOpenLogs={() => setPage("logs")}/>
                        : <LogsPage/>}
                </main>
            </div>
        </div>
    );
}

export default App;
