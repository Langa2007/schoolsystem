import { useState } from "react";
import { Outlet } from "react-router-dom";
import Sidebar from "./Sidebar.jsx";
import Navbar from "./Navbar.jsx";

export default function AppLayout() {
  const [menuOpen, setMenuOpen] = useState(false);
  return <div className="app-shell"><Sidebar open={menuOpen} onNavigate={() => setMenuOpen(false)} /><div className="app-main"><Navbar onMenu={() => setMenuOpen((open) => !open)} /><main><Outlet /></main><footer className="app-footer"><span>© 2026 ElimuCore</span><span>School management, made clear.</span></footer></div>{menuOpen && <button className="mobile-scrim" aria-label="Close navigation" onClick={() => setMenuOpen(false)} />}</div>;
}
