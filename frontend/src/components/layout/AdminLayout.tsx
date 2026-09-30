import { useState } from "react";
import { Link, useLocation, useNavigate, Outlet, Navigate } from "react-router-dom";
import {
  ShieldAlert,
  LayoutDashboard,
  FileSpreadsheet,
  Database,
  Users,
  Award,
  LogOut,
  UserCheck,
  Activity,
  Menu,
  X,
  ChevronRight,
} from "lucide-react";
import { useAuth } from "../../hooks/useAuth";
import { Button } from "../ui/button";
import { Badge } from "../ui/badge";

export function AdminLayout() {
  const location = useLocation();
  const navigate = useNavigate();
  const { user, token, role, loginAsDevCandidate, logout } = useAuth();
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  if (!token) {
    return <Navigate to="/" replace />;
  }
  if (role !== "exam_admin") {
    return <Navigate to="/candidate" replace />;
  }

  const handleLogout = () => {
    logout();
    navigate("/", { replace: true });
  };

  const navItems = [
    { label: "Dashboard", path: "/admin", icon: LayoutDashboard },
    { label: "Exams", path: "/admin/exams", icon: FileSpreadsheet },
    { label: "Question Banks", path: "/admin/question-banks", icon: Database },
    { label: "Candidates", path: "/admin/candidates", icon: Users },
    { label: "Attempts", path: "/admin/attempts", icon: Award },
  ];

  const handleSwitchToCandidate = () => {
    loginAsDevCandidate();
    navigate("/candidate");
  };

  return (
    <div className="min-h-screen bg-slate-950 flex flex-col text-slate-100 font-sans">
      {/* Topbar */}
      <header className="h-16 border-b border-slate-800 bg-slate-900/90 backdrop-blur-md px-4 lg:px-6 flex items-center justify-between sticky top-0 z-40">
        <div className="flex items-center gap-3">
          <button
            onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
            className="lg:hidden text-slate-400 hover:text-white p-1"
          >
            {mobileMenuOpen ? <X className="h-6 w-6" /> : <Menu className="h-6 w-6" />}
          </button>
          
          <Link to="/admin" className="flex items-center gap-2.5 group">
            <div className="h-9 w-9 rounded-lg bg-cyan-500/10 border border-cyan-500/30 flex items-center justify-center text-cyan-400 group-hover:border-cyan-400 transition-colors glow-cyan">
              <ShieldAlert className="h-5 w-5" />
            </div>
            <div>
              <span className="font-bold tracking-tight text-white text-lg">SENTINEL</span>
              <span className="text-cyan-400 font-light text-sm ml-1">EXAM</span>
              <Badge variant="outline" className="ml-2 text-[10px] py-0 px-1.5 border-slate-700 text-slate-400">
                ADMIN
              </Badge>
            </div>
          </Link>
        </div>

        {/* Topbar Right */}
        <div className="flex items-center gap-3">
          <div className="hidden sm:flex items-center gap-2 px-3 py-1.5 rounded-full bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-xs">
            <Activity className="h-3.5 w-3.5 animate-pulse" />
            <span>API Online</span>
          </div>

          <Button
            variant="outline"
            size="sm"
            onClick={handleSwitchToCandidate}
            className="text-xs border-indigo-500/30 bg-indigo-500/10 text-indigo-300 hover:bg-indigo-500/20 hover:text-indigo-200"
          >
            <UserCheck className="h-3.5 w-3.5 mr-1.5" />
            Switch to Candidate
          </Button>

          <div className="flex items-center gap-2 pl-2 border-l border-slate-800">
            <div className="hidden md:flex flex-col text-right">
              <span className="text-xs font-semibold text-slate-200">{user?.name || "Administrator"}</span>
              <span className="text-[10px] text-slate-400">{user?.email}</span>
            </div>
            <Button
              variant="ghost"
              size="icon"
              onClick={handleLogout}
              title="Logout"
              className="text-slate-400 hover:text-red-400"
            >
              <LogOut className="h-4 w-4" />
            </Button>
          </div>
        </div>
      </header>

      {/* Main Body */}
      <div className="flex-1 flex overflow-hidden">
        {/* Desktop Sidebar */}
        <aside className="hidden lg:flex w-64 flex-col border-r border-slate-800 bg-slate-900/60 p-4 shrink-0">
          <nav className="space-y-1">
            {navItems.map((item) => {
              const Icon = item.icon;
              const isActive =
                item.path === "/admin"
                  ? location.pathname === "/admin"
                  : location.pathname.startsWith(item.path);

              return (
                <Link
                  key={item.path}
                  to={item.path}
                  className={`flex items-center gap-3 px-3.5 py-2.5 rounded-lg text-sm font-medium transition-all ${
                    isActive
                      ? "bg-cyan-500/10 text-cyan-400 border border-cyan-500/30 shadow-sm"
                      : "text-slate-400 hover:text-slate-200 hover:bg-slate-800/60"
                  }`}
                >
                  <Icon className={`h-4 w-4 ${isActive ? "text-cyan-400" : "text-slate-400"}`} />
                  <span className="flex-1">{item.label}</span>
                  {isActive && <ChevronRight className="h-4 w-4 text-cyan-500/50" />}
                </Link>
              );
            })}
          </nav>

          <div className="mt-auto pt-4 border-t border-slate-800/80">
            <div className="p-3 rounded-lg bg-slate-900/90 border border-slate-800 text-xs text-slate-400 space-y-1">
              <div className="font-semibold text-slate-300">Phase 3 — Assessment Core</div>
              <div>Authoritative DB + JetStream</div>
              <div className="text-[10px] text-slate-500 font-mono">v1.0.0-PROD</div>
            </div>
          </div>
        </aside>

        {/* Mobile Navigation Drawer */}
        {mobileMenuOpen && (
          <div className="fixed inset-0 z-50 lg:hidden flex">
            <div className="fixed inset-0 bg-slate-950/80 backdrop-blur-sm" onClick={() => setMobileMenuOpen(false)} />
            <div className="relative w-64 bg-slate-900 border-r border-slate-800 p-4 flex flex-col z-10">
              <nav className="space-y-1">
                {navItems.map((item) => {
                  const Icon = item.icon;
                  const isActive =
                    item.path === "/admin"
                      ? location.pathname === "/admin"
                      : location.pathname.startsWith(item.path);

                  return (
                    <Link
                      key={item.path}
                      to={item.path}
                      onClick={() => setMobileMenuOpen(false)}
                      className={`flex items-center gap-3 px-3.5 py-2.5 rounded-lg text-sm font-medium transition-all ${
                        isActive
                          ? "bg-cyan-500/10 text-cyan-400 border border-cyan-500/30"
                          : "text-slate-400 hover:text-slate-200 hover:bg-slate-800"
                      }`}
                    >
                      <Icon className="h-4 w-4" />
                      <span>{item.label}</span>
                    </Link>
                  );
                })}
              </nav>
            </div>
          </div>
        )}

        {/* Main Content Area */}
        <main className="flex-1 overflow-y-auto p-4 lg:p-8 max-w-7xl mx-auto w-full">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
