import { ArrowDownRight, ArrowUpRight, ArrowUpRight as OpenIcon, CalendarDays, ChevronRight, CircleAlert, Clock3, Download, Plus, Users, UserRoundCheck, WalletCards } from "lucide-react";
import { Link } from "react-router-dom";

const metrics = [
  { label: "Total students", value: "1,284", change: "+4.8%", up: true, icon: Users, tone: "blue", note: "compared to last term" },
  { label: "Attendance today", value: "94.6%", change: "+1.2%", up: true, icon: UserRoundCheck, tone: "green", note: "compared to last week" },
  { label: "Fee collection", value: "KES 8.42M", change: "−2.4%", up: false, icon: WalletCards, tone: "violet", note: "of KES 12.5M invoiced" },
  { label: "Open actions", value: "18", change: "6 urgent", up: false, icon: CircleAlert, tone: "amber", note: "need your attention" },
];

const week = [
  { day: "Mon", present: 93, absent: 7 }, { day: "Tue", present: 96, absent: 4 },
  { day: "Wed", present: 91, absent: 9 }, { day: "Thu", present: 95, absent: 5 },
  { day: "Fri", present: 89, absent: 11 }, { day: "Sat", present: 0, absent: 0 }, { day: "Sun", present: 0, absent: 0 },
];
const actions = [
  { title: "Morning roll incomplete", detail: "Form 2 West · Class teacher pending", time: "12 min ago", status: "Needs action", tone: "red" },
  { title: "Leave request awaiting review", detail: "Student #ST-2048 · 2 school days", time: "38 min ago", status: "Review", tone: "blue" },
  { title: "Fee payment received", detail: "M-Pesa · Receipt QK72P9 · KES 12,500", time: "1 hr ago", status: "Recorded", tone: "green" },
];

function PageHeading({ eyebrow, title, description, action }) {
  return <div className="page-heading"><div><div className="eyebrow">{eyebrow}</div><h1>{title}</h1><p>{description}</p></div>{action}</div>;
}

function MetricCard({ metric }) {
  const Icon = metric.icon;
  return <article className="metric-card"><div className="metric-top"><span>{metric.label}</span><span className={`metric-icon ${metric.tone}`}><Icon size={18} /></span></div><div className="metric-value">{metric.value}</div><div className="metric-foot"><span className={`change ${metric.up ? "positive" : "negative"}`}>{metric.up ? <ArrowUpRight size={14} /> : <ArrowDownRight size={14} />}{metric.change}</span><span>{metric.note}</span></div></article>;
}

function AttendanceChart() {
  return <div className="chart-area"><div className="chart-y"><span>100%</span><span>75%</span><span>50%</span><span>25%</span><span>0%</span></div><div className="chart-plot"><div className="chart-grid"><i /><i /><i /><i /><i /></div><div className="bar-list">{week.map((item) => <div className="bar-column" key={item.day}><div className="bar-stack">{item.present > 0 && <><div className="bar-present" style={{ height: `${item.present}%` }} /><div className="bar-absent" style={{ height: `${item.absent}%` }} /></>}</div><span>{item.day}</span></div>)}</div></div>
  </div>;
}

export function DashboardPage() {
  return <div className="page-content">
    <PageHeading eyebrow="MONDAY, 19 OCTOBER 2026" title="Good morning, Admin" description="Here’s what’s happening across your school today." action={<button className="button button-secondary"><Download size={16} /> Export report</button>} />
    <div className="notice-strip"><span className="notice-mark"><CalendarDays size={17} /></span><div><strong>Term 3 is in progress</strong><span>Week 7 · 9 weeks remaining in the academic year</span></div><Link to="/academics">View calendar <ChevronRight size={15} /></Link></div>
    <section className="metric-grid" aria-label="School overview">{metrics.map((metric) => <MetricCard metric={metric} key={metric.label} />)}</section>
    <section className="dashboard-grid">
      <article className="panel attendance-panel"><div className="panel-heading"><div><h2>Attendance overview</h2><p>Daily student attendance this week</p></div><select className="select-control" aria-label="Attendance period"><option>This week</option><option>This month</option></select></div><div className="chart-legend"><span><i className="legend-present" /> Present</span><span><i className="legend-absent" /> Absent or late</span></div><AttendanceChart /></article>
      <article className="panel actions-panel"><div className="panel-heading"><div><h2>Needs attention</h2><p>Items that may need a follow-up</p></div><Link className="panel-link" to="/attendance">View all <ChevronRight size={15} /></Link></div><div className="action-list">{actions.map((action) => <div className="action-row" key={action.title}><span className={`action-dot ${action.tone}`} /><div className="action-copy"><strong>{action.title}</strong><span>{action.detail}</span><small><Clock3 size={12} />{action.time}</small></div><span className={`status-pill ${action.tone}`}>{action.status}</span></div>)}</div><Link className="all-actions" to="/attendance">Open action centre <ChevronRight size={15} /></Link></article>
    </section>
    <section className="bottom-grid"><article className="panel enrollment-panel"><div className="panel-heading"><div><h2>Students by level</h2><p>Current enrolment distribution</p></div><Link className="icon-link" to="/students" aria-label="View students"><OpenIcon size={17} /></Link></div><div className="level-list">{[["Form 1",278,"blue"],["Form 2",304,"indigo"],["Form 3",336,"cyan"],["Form 4",366,"navy"]].map(([name,count,tone])=><div className="level-row" key={name}><span className="level-name"><i className={`level-dot ${tone}`} />{name}</span><div className="level-track"><i className={tone} style={{width:`${count/3.66}%`}} /></div><strong>{count}</strong></div>)}</div></article><article className="panel quick-panel"><div className="panel-heading"><div><h2>Quick actions</h2><p>Common school tasks</p></div></div><div className="quick-actions"><Link to="/students" className="quick-action"><span className="quick-icon blue"><Plus size={18} /></span><span><strong>Register student</strong><small>Add a new learner</small></span><ChevronRight size={16} /></Link><Link to="/attendance" className="quick-action"><span className="quick-icon green"><UserRoundCheck size={18} /></span><span><strong>Take attendance</strong><small>Open today’s class roll</small></span><ChevronRight size={16} /></Link><Link to="/finance" className="quick-action"><span className="quick-icon violet"><WalletCards size={18} /></span><span><strong>Record payment</strong><small>Update a student ledger</small></span><ChevronRight size={16} /></Link></div></article></section>
    <div className="demo-note">Dashboard figures shown here are sample data while the backend integration is being connected.</div>
  </div>;
}

export function ModulePage({ title, description }) {
  return <div className="page-content"><PageHeading eyebrow="SCHOOL MANAGEMENT" title={title} description={description} action={<button className="button button-primary"><Plus size={16} /> Add {title.slice(0, -1)}</button>} /><div className="panel module-empty"><span className="module-empty-icon"><ArrowUpRight size={20} /></span><h2>{title} workspace</h2><p>This section is ready for the {title.toLowerCase()} workflow. Connect it to the available API to load school records.</p><Link to="/dashboard" className="button button-secondary">Back to overview</Link></div></div>;
}
