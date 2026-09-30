
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Clock, BookOpen, Play, ShieldAlert } from "lucide-react";
import { api, Exam } from "../../lib/api";
import { Card, CardHeader, CardTitle, CardContent, CardDescription, CardFooter } from "../../components/ui/card";
import { Button } from "../../components/ui/button";
import { Badge } from "../../components/ui/badge";
import { useAuth } from "../../hooks/useAuth";
import { Skeleton } from "../../components/ui/skeleton";

export function CandidateDashboard() {
  const { role } = useAuth();
  const { data: examsData, isLoading } = useQuery({
    queryKey: ["candidate-assigned-exams"],
    queryFn: () => api.listAssignedExams(),
    enabled: role === "candidate",
  });

  const assignedExams = (examsData?.items || []).filter(Boolean);

  return (
    <div className="space-y-8">
      {/* Header Banner */}
      <div className="border-b border-slate-800 pb-6 space-y-2">
        <h1 className="text-2xl font-bold tracking-tight text-white flex items-center gap-3">
          Candidate Assessment Hub
        </h1>
        <p className="text-slate-400 text-sm">
          Select an assigned assessment to begin. Ensure a stable network connection before starting.
        </p>
      </div>

      {/* Exam Grid */}
      {isLoading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          <Skeleton className="h-48 w-full rounded-xl" />
          <Skeleton className="h-48 w-full rounded-xl" />
        </div>
      ) : assignedExams.length === 0 ? (
        <Card className="border-slate-800 bg-slate-900/60 p-12 text-center text-slate-400 space-y-3">
          <ShieldAlert className="h-10 w-10 text-slate-500 mx-auto" />
          <h3 className="text-lg font-semibold text-slate-200">No Assigned Assessments</h3>
          <p className="text-xs text-slate-400 max-w-md mx-auto">
            You do not currently have any active assessment assignments. Check back later or contact your test administrator.
          </p>
        </Card>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {assignedExams.map((exam: Exam, idx: number) => (
            <Card
              key={exam.id || `assigned-${idx}`}
              className="border-slate-800 bg-slate-900/80 hover:border-slate-700 transition-all duration-200 shadow-xl flex flex-col justify-between"
            >
              <CardHeader className="space-y-3 pb-3">
                <div className="flex items-center justify-between">
                  <Badge variant="outline" className="font-mono text-cyan-400 border-cyan-500/30 text-xs">
                    {exam.code}
                  </Badge>
                  <Badge variant="success" className="text-xs">
                    {exam.status}
                  </Badge>
                </div>
                <CardTitle className="text-xl font-bold text-white leading-snug">
                  {exam.title}
                </CardTitle>
                <CardDescription className="line-clamp-2 text-xs text-slate-400">
                  {exam.description || "Standard CBT assessment module."}
                </CardDescription>
              </CardHeader>

              <CardContent className="space-y-4 pt-2">
                <div className="grid grid-cols-2 gap-3 text-xs bg-slate-950/60 p-3 rounded-lg border border-slate-800/80 font-mono">
                  <div className="flex items-center gap-2 text-slate-300">
                    <Clock className="h-4 w-4 text-cyan-400" />
                    <span>{exam.duration_minutes} Minutes</span>
                  </div>
                  <div className="flex items-center gap-2 text-slate-300">
                    <BookOpen className="h-4 w-4 text-purple-400" />
                    <span>Passing: {exam.pass_marks} pts</span>
                  </div>
                </div>
              </CardContent>

              <CardFooter className="pt-2 border-t border-slate-800/80 flex justify-between items-center">
                <span className="text-[11px] text-slate-500 font-mono">ID: {exam.id ? String(exam.id).slice(0, 8) : 'N/A'}</span>
                {exam.id && exam.id !== 'undefined' ? (
                  <Link to={`/candidate/exams/${exam.id}/instructions`}>
                    <Button className="bg-cyan-600 hover:bg-cyan-500 text-white font-medium text-xs glow-cyan">
                      <Play className="h-3.5 w-3.5 mr-1.5 fill-current" />
                      Start Assessment
                    </Button>
                  </Link>
                ) : (
                  <Button disabled className="text-xs">Unavailable</Button>
                )}
              </CardFooter>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
