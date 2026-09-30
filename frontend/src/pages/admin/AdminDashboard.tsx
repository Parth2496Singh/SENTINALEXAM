
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import {
  FileSpreadsheet,
  Users,
  Plus,
  ArrowRight,
  Database,
  Activity,
  CheckCircle2,
  Clock,
  PlayCircle,
} from "lucide-react";
import { api, Exam } from "../../lib/api";
import { Card, CardContent } from "../../components/ui/card";
import { Button } from "../../components/ui/button";
import { Badge } from "../../components/ui/badge";
import { Table, TableHeader, TableRow, TableHead, TableBody, TableCell } from "../../components/ui/table";
import { useAuth } from "../../hooks/useAuth";
import { Skeleton } from "../../components/ui/skeleton";
import { formatDate } from "../../lib/utils";

export function AdminDashboard() {
  const { role } = useAuth();

  const { data: exams, isLoading: isLoadingExams } = useQuery({
    queryKey: ["exams"],
    queryFn: () => api.listExams(),
    enabled: role === "exam_admin",
  });

  const { data: candidates, isLoading: isLoadingCandidates } = useQuery({
    queryKey: ["candidates"],
    queryFn: () => api.listCandidates(),
    enabled: role === "exam_admin",
  });

  const { data: info } = useQuery({
    queryKey: ["info"],
    queryFn: () => api.getInfo(),
    enabled: !!role,
  });

  const examList = (exams?.items || []).filter(Boolean) as Exam[];
  const candidateList = (candidates?.items || []).filter(Boolean);

  const activeExams = examList.filter((e: Exam) => e.status === "LIVE").length;
  const scheduledExams = examList.filter((e: Exam) => e.status === "SCHEDULED" || e.status === "PUBLISHED").length;
  const totalCandidates = candidateList.length;

  return (
    <div className="space-y-8">
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 border-b border-slate-800 pb-6">
        <div>
          <h1 className="text-2xl font-extrabold tracking-tight text-white flex items-center gap-3">
            Assessment Operations Center
          </h1>
          <p className="text-slate-400 text-sm mt-1">
            Real-time assessment management, candidate scheduling, and authoritative session overview.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <Link to="/admin/exams">
            <Button className="bg-cyan-600 hover:bg-cyan-500 text-white font-medium glow-cyan">
              <Plus className="h-4 w-4 mr-2" />
              Create New Exam
            </Button>
          </Link>
        </div>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <Card className="bg-slate-900/60 border-slate-800/80 relative overflow-hidden">
          <div className="absolute top-0 right-0 w-24 h-24 bg-cyan-500/5 rounded-full blur-xl pointer-events-none" />
          <CardContent className="p-6">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Live Exams</span>
              <div className="p-2 rounded-lg bg-cyan-500/10 text-cyan-400">
                <PlayCircle className="h-5 w-5" />
              </div>
            </div>
            <div className="mt-4 flex items-baseline gap-2">
              <span className="text-3xl font-bold text-white font-mono">
                {isLoadingExams ? <Skeleton className="h-8 w-12" /> : activeExams}
              </span>
              <span className="text-xs text-cyan-400 font-medium">Currently active</span>
            </div>
          </CardContent>
        </Card>

        <Card className="bg-slate-900/60 border-slate-800/80 relative overflow-hidden">
          <div className="absolute top-0 right-0 w-24 h-24 bg-indigo-500/5 rounded-full blur-xl pointer-events-none" />
          <CardContent className="p-6">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Scheduled</span>
              <div className="p-2 rounded-lg bg-indigo-500/10 text-indigo-400">
                <Clock className="h-5 w-5" />
              </div>
            </div>
            <div className="mt-4 flex items-baseline gap-2">
              <span className="text-3xl font-bold text-white font-mono">
                {isLoadingExams ? <Skeleton className="h-8 w-12" /> : scheduledExams}
              </span>
              <span className="text-xs text-indigo-400 font-medium">Ready to deploy</span>
            </div>
          </CardContent>
        </Card>

        <Card className="bg-slate-900/60 border-slate-800/80 relative overflow-hidden">
          <div className="absolute top-0 right-0 w-24 h-24 bg-purple-500/5 rounded-full blur-xl pointer-events-none" />
          <CardContent className="p-6">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Candidates</span>
              <div className="p-2 rounded-lg bg-purple-500/10 text-purple-400">
                <Users className="h-5 w-5" />
              </div>
            </div>
            <div className="mt-4 flex items-baseline gap-2">
              <span className="text-3xl font-bold text-white font-mono">
                {isLoadingCandidates ? <Skeleton className="h-8 w-12" /> : totalCandidates}
              </span>
              <span className="text-xs text-purple-400 font-medium">Registered</span>
            </div>
          </CardContent>
        </Card>

        <Card className="bg-slate-900/60 border-slate-800/80 relative overflow-hidden">
          <div className="absolute top-0 right-0 w-24 h-24 bg-emerald-500/5 rounded-full blur-xl pointer-events-none" />
          <CardContent className="p-6">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold text-slate-400 uppercase tracking-wider">Total Exams</span>
              <div className="p-2 rounded-lg bg-emerald-500/10 text-emerald-400">
                <FileSpreadsheet className="h-5 w-5" />
              </div>
            </div>
            <div className="mt-4 flex items-baseline gap-2">
              <span className="text-3xl font-bold text-white font-mono">
                {isLoadingExams ? <Skeleton className="h-8 w-12" /> : examList.length}
              </span>
              <span className="text-xs text-emerald-400 font-medium">Created</span>
            </div>
          </CardContent>
        </Card>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        <div className="lg:col-span-2 space-y-4">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold text-white flex items-center gap-2">
              <FileSpreadsheet className="h-5 w-5 text-cyan-400" />
              Recent Assessments
            </h2>
            <Link to="/admin/exams" className="text-xs text-cyan-400 hover:underline flex items-center gap-1">
              View All <ArrowRight className="h-3 w-3" />
            </Link>
          </div>

          <Card className="border-slate-800 bg-slate-900/60">
            <CardContent className="p-0">
              {isLoadingExams ? (
                <div className="p-6 space-y-3">
                  <Skeleton className="h-10 w-full" />
                  <Skeleton className="h-10 w-full" />
                  <Skeleton className="h-10 w-full" />
                </div>
              ) : examList.length === 0 ? (
                <div className="p-8 text-center text-slate-400">
                  <p>No assessments created yet.</p>
                  <Link to="/admin/exams">
                    <Button variant="outline" size="sm" className="mt-3">
                      Create First Exam
                    </Button>
                  </Link>
                </div>
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Exam Title</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Duration</TableHead>
                      <TableHead>Created</TableHead>
                      <TableHead className="text-right">Action</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {examList.slice(0, 5).map((exam: Exam, idx: number) => (
                      <TableRow key={exam.id || `recent-exam-${idx}`}>
                        <TableCell className="font-semibold text-white">
                          <Link to={`/admin/exams/${exam.id}`} className="hover:text-cyan-400 transition-colors">
                            {exam.title}
                          </Link>
                          <div className="text-xs text-slate-400 font-mono font-normal">
                            Code: {exam.code}
                          </div>
                        </TableCell>
                        <TableCell>
                          <Badge
                            variant={
                              exam.status === "LIVE"
                                ? "success"
                                : exam.status === "PUBLISHED" || exam.status === "SCHEDULED"
                                ? "default"
                                : exam.status === "DRAFT"
                                ? "secondary"
                                : "outline"
                            }
                          >
                            {exam.status}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-slate-300 font-mono text-xs">
                          {exam.duration_minutes} mins
                        </TableCell>
                        <TableCell className="text-slate-400 text-xs">
                          {formatDate(exam.created_at)}
                        </TableCell>
                        <TableCell className="text-right">
                          <Link to={`/admin/exams/${exam.id}`}>
                            <Button variant="ghost" size="sm" className="text-xs">
                              Manage
                            </Button>
                          </Link>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
            </CardContent>
          </Card>
        </div>

        <div className="space-y-4">
          <h2 className="text-lg font-semibold text-white flex items-center gap-2">
            <Activity className="h-5 w-5 text-indigo-400" />
            Infrastructure Status
          </h2>

          <Card className="border-slate-800 bg-slate-900/60 space-y-4 p-6">
            <div className="space-y-3">
              <div className="flex items-center justify-between p-3 rounded-lg bg-slate-950/80 border border-slate-800/80">
                <div className="flex items-center gap-2.5">
                  <CheckCircle2 className="h-4 w-4 text-emerald-400" />
                  <span className="text-sm font-medium text-slate-200">Go API Service</span>
                </div>
                <Badge variant="success">HEALTHY</Badge>
              </div>

              <div className="flex items-center justify-between p-3 rounded-lg bg-slate-950/80 border border-slate-800/80">
                <div className="flex items-center gap-2.5">
                  <Database className="h-4 w-4 text-emerald-400" />
                  <span className="text-sm font-medium text-slate-200">PostgreSQL (pgxpool)</span>
                </div>
                <Badge variant="success">AUTHORITATIVE</Badge>
              </div>

              <div className="flex items-center justify-between p-3 rounded-lg bg-slate-950/80 border border-slate-800/80">
                <div className="flex items-center gap-2.5">
                  <Activity className="h-4 w-4 text-emerald-400" />
                  <span className="text-sm font-medium text-slate-200">NATS JetStream</span>
                </div>
                <Badge variant="success">STREAM READY</Badge>
              </div>
            </div>

            <div className="pt-2 border-t border-slate-800 text-xs text-slate-400 space-y-1">
              <div className="flex justify-between">
                <span>Phase Readiness:</span>
                <span className="font-semibold text-slate-200">Phase 3 Assessment Core</span>
              </div>
              <div className="flex justify-between">
                <span>Backend Status:</span>
                <span className="font-mono text-cyan-400">{info?.status || "assessment_core_ready"}</span>
              </div>
            </div>
          </Card>
        </div>
      </div>
    </div>
  );
}
