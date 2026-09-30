
import { useQuery } from "@tanstack/react-query";
import { Award } from "lucide-react";
import { api, Exam, ExamAttempt } from "../../lib/api";
import { Card, CardContent } from "../../components/ui/card";
import { Badge } from "../../components/ui/badge";
import { Table, TableHeader, TableRow, TableHead, TableBody, TableCell } from "../../components/ui/table";
import { Skeleton } from "../../components/ui/skeleton";
import { useAuth } from "../../hooks/useAuth";
import { formatDate } from "../../lib/utils";

export function AttemptsPage() {
  const { role } = useAuth();
  const { data: examsData, isLoading } = useQuery({
    queryKey: ["exams"],
    queryFn: () => api.listExams(),
    enabled: role === "exam_admin",
  });

  const exams = examsData?.items || [];
  const allAttempts = exams.flatMap((e: Exam) =>
    (e.attempts || []).map((a: ExamAttempt) => ({
      ...a,
      examTitle: e.title,
      examCode: e.code,
    }))
  );

  return (
    <div className="space-y-6">
      <div className="border-b border-slate-800 pb-5">
        <h1 className="text-2xl font-bold text-white flex items-center gap-2">
          <Award className="h-6 w-6 text-emerald-400" />
          Assessment Attempts & Logs
        </h1>
        <p className="text-slate-400 text-sm mt-1">
          Authoritative record of candidate sessions, deadlines, and submissions.
        </p>
      </div>

      <Card className="border-slate-800 bg-slate-900/60">
        <CardContent className="p-0">
          {isLoading ? (
            <div className="p-6 space-y-3">
              <Skeleton className="h-12 w-full" />
              <Skeleton className="h-12 w-full" />
            </div>
          ) : allAttempts.length === 0 ? (
            <div className="p-12 text-center text-slate-400">
              <p className="text-base font-medium">No candidate attempts recorded yet.</p>
              <p className="text-xs text-slate-500 mt-1">
                Candidate attempts will automatically appear here as test sessions are initiated.
              </p>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Assessment</TableHead>
                  <TableHead>Candidate UUID</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Started At</TableHead>
                  <TableHead>Expires At</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {allAttempts.map((att: any) => (
                  <TableRow key={att.id}>
                    <TableCell className="font-semibold text-white">
                      {att.examTitle}
                      <div className="text-xs text-slate-400 font-mono font-normal">
                        Code: {att.examCode}
                      </div>
                    </TableCell>
                    <TableCell className="font-mono text-xs text-cyan-400">
                      {att.candidate_id}
                    </TableCell>
                    <TableCell>
                      <Badge
                        variant={
                          att.status === "SUBMITTED" || att.status === "EVALUATED"
                            ? "success"
                            : att.status === "IN_PROGRESS" || att.status === "ACTIVE"
                            ? "warning"
                            : "outline"
                        }
                      >
                        {att.status}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-slate-400 text-xs font-mono">
                      {formatDate(att.started_at)}
                    </TableCell>
                    <TableCell className="text-slate-400 text-xs font-mono">
                      {formatDate(att.expires_at)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
