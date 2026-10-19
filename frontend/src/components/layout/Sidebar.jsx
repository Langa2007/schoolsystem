import { BookOpenCheck, CalendarDays, ChartNoAxesCombined, GraduationCap, LayoutDashboard, Settings2, Users, WalletCards, X } from "lucide-react";
import { NavLink } from "react-router-dom";

const navigation = [
  { label: "OVERVIEW", links: [{ to: "/dashboard", label: "Dashboard", icon: LayoutDashboard }] },
  { label: "SCHOOL", links: [
    { to: "/students", label: "Students", icon: GraduationCap },
    { to: "/attendance", label: "Attendance", icon: CalendarDays },
    { to: "/academics", label: "Academics", icon: BookOpenCheck },
    { to: "/staff", label: "Staff", icon: Users },
  ] },
  { label: "MANAGEMENT", links: [
    { to: "/finance", label: "Finance", icon: WalletCards },
    { to: "/reports", label: "Reports", icon: ChartNoAxesCombined },
  ] },
];

export default function Sidebar({ open, onNavigate }) {
  return <aside className={`sidebar ${open ? "sidebar-open" : ""}`}>
    <div className="brand-row"><div className="brand-mark"><BookOpenCheck size={20} strokeWidth={2.3} /></div><div className="brand-name">elimu<span>core</span><small>SCHOOL MANAGEMENT</small></div><button className="sidebar-close" onClick={onNavigate} aria-label="Close navigation"><X size={19} /></button></div>
    <div className="school-switch"><div className="school-avatar">KM</div><div className="school-details"><strong>Kijani Memorial</strong><span>Secondary School</span></div><span className="switch-chevron">⌄</span></div>
    <nav className="side-nav" aria-label="Main navigation">{navigation.map((group)=><div className="nav-group" key={group.label}><div className="nav-label">{group.label}</div>{group.links.map(({to,label,icon:Icon})=><NavLink key={to} to={to} onClick={onNavigate} className={({isActive})=>`nav-link ${isActive ? "active" : ""}`}><Icon size={18} strokeWidth={1.9} /><span>{label}</span></NavLink>)}</div>)}</nav>
    <div className="sidebar-bottom"><NavLink to="/settings" onClick={onNavigate} className={({isActive})=>`nav-link settings-link ${isActive ? "active" : ""}`}><Settings2 size={18} /><span>Settings</span></NavLink><div className="help-card"><div className="help-badge">?</div><strong>Need a hand?</strong><p>Visit the help centre for guides and support.</p><button type="button">Get support <span>↗</span></button></div><div className="profile-card"><div className="profile-avatar">JM</div><div className="profile-details"><strong>Jane Mwangi</strong><span>School Administrator</span></div><button className="profile-more" aria-label="Profile options">···</button></div></div>
  </aside>;
}
