import { Navigate, Route, Routes } from "react-router-dom";
import AppLayout from "./components/layout/AppLayout.jsx";
import { DashboardPage, ModulePage } from "./routes.jsx";

export default function App() {
  return (
    <Routes>
      <Route element={<AppLayout />}>
        <Route index element={<DashboardPage />} />
        <Route path="/dashboard" element={<DashboardPage />} />
        <Route path="/students" element={<ModulePage title="Students" description="Student records, enrolment and class placement." />} />
        <Route path="/attendance" element={<ModulePage title="Attendance" description="Daily roll calls, lesson attendance and reconciliation." />} />
        <Route path="/academics" element={<ModulePage title="Academics" description="Terms, classes, subjects and academic progress." />} />
        <Route path="/staff" element={<ModulePage title="Staff" description="Staff directory, roles and assignments." />} />
        <Route path="/finance" element={<ModulePage title="Finance" description="Fee structures, student ledgers and payments." />} />
        <Route path="/reports" element={<ModulePage title="Reports" description="School performance and operational reports." />} />
        <Route path="/settings" element={<ModulePage title="Settings" description="School profile and system preferences." />} />
        <Route path="*" element={<Navigate to="/dashboard" replace />} />
      </Route>
    </Routes>
  );
}
