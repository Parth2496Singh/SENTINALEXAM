
import { Link, Outlet, useNavigate, Navigate } from "react-router-dom";
import { LogOut, ShieldCheck, UserCog } from "lucide-react";
import { useAuth } from "../../hooks/useAuth";
import { Button } from "../ui/button";
import { Badge } from "../ui/badge";

export function CandidateLayout() {
  const { user, token, role, loginAsDevAdmin, logout } = useAuth();
  const navigate = useNavigate();

  if (!token) {
    return <Navigate to="/" replace />;
  }
  if (role !== "candidate") {
    return <Navigate to="/admin" replace />;
  }

  const handleSwitchToAdmin = () => {
    loginAsDevAdmin();
    navigate("/admin");
  };

  const handleLogout = () => {
    logout();
    navigate("/", { replace: true });
  };

  return (
    <div className="min-h-screen bg-slate-950 flex flex-col text-slate-100 font-sans selection:bg-cyan-500/20">
      {/* Topbar */}
      <header className="h-16 border-b border-slate-800 bg-slate-900/80 backdrop-blur-md px-4 lg:px-8 flex items-center justify-between sticky top-0 z-40">
        <Link to="/candidate" className="flex items-center gap-2.5">
          <div className="h-9 w-9 rounded-lg bg-emerald-500/10 border border-emerald-500/30 flex items-center justify-center text-emerald-400">
            <ShieldCheck className="h-5 w-5" />
          </div>
          <div>
            <span className="font-bold tracking-tight text-white text-lg">SENTINEL</span>
            <span className="text-emerald-400 font-light text-sm ml-1">PORTAL</span>
            <Badge variant="outline" className="ml-2 text-[10px] py-0 px-1.5 border-emerald-500/30 text-emerald-400">
              CANDIDATE
            </Badge>
          </div>
        </Link>

        <div className="flex items-center gap-3">
          <Button
            variant="outline"
            size="sm"
            onClick={handleSwitchToAdmin}
            className="text-xs border-cyan-500/30 bg-cyan-500/10 text-cyan-300 hover:bg-cyan-500/20"
          >
            <UserCog className="h-3.5 w-3.5 mr-1.5" />
            Switch to Admin
          </Button>

          <div className="flex items-center gap-2 pl-2 border-l border-slate-800">
            <div className="hidden sm:flex flex-col text-right">
              <span className="text-xs font-semibold text-slate-200">{user?.name || "Candidate"}</span>
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

      {/* Main Container */}
      <main className="flex-1 p-4 lg:p-8 max-w-5xl mx-auto w-full">
        <Outlet />
      </main>

      {/* Subtle Footer */}
      <footer className="border-t border-slate-900 p-4 text-center text-xs text-slate-500">
        SentinelExam Security Operations Assessment Platform &copy; 2026. Authoritative session tracking active.
      </footer>
    </div>
  );
}
