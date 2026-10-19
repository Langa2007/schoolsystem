import { Bell, ChevronDown, Menu, Search } from "lucide-react";
import { useLocation } from "react-router-dom";

const pageNames = { "/dashboard": "Dashboard", "/students": "Students", "/attendance": "Attendance", "/academics": "Academics", "/staff": "Staff", "/finance": "Finance", "/reports": "Reports", "/settings": "Settings", "/": "Dashboard" };

export default function Navbar({ onMenu }) {
  const { pathname } = useLocation();
  return <header className="topbar"><button className="menu-toggle" onClick={onMenu} aria-label="Open navigation"><Menu size={21} /></button><div className="breadcrumb"><span>ElimuCore</span><span className="crumb-divider">/</span><strong>{pageNames[pathname] || "Dashboard"}</strong></div><div className="topbar-tools"><label className="search-box"><Search size={17} /><input placeholder="Search anything..." aria-label="Search" /><kbd>⌘ K</kbd></label><button className="notification-button" aria-label="Notifications"><Bell size={19} /><i /></button><div className="topbar-date">Mon, 19 Oct 2026 <ChevronDown size={14} /></div></div></header>;
}
