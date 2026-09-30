import { useState } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import { useQuery, useMutation } from "@tanstack/react-query";
import { ShieldAlert, ArrowLeft, Play, AlertCircle } from "lucide-react";
import { api, ExamAttempt } from "../../lib/api";
import { Card, CardHeader, CardTitle, CardContent, CardFooter } from "../../components/ui/card";
import { Button } from "../../components/ui/button";
import { Badge } from "../../components/ui/badge";
import { Skeleton } from "../../components/ui/skeleton";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "../../components/ui/dialog";

export function ExamInstructionsPage() {
  const { examId } = useParams<{ examId: string }>();
  const navigate = useNavigate();
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const { data: exam, isLoading } = useQuery({
    queryKey: ["candidate-exam", examId],
    queryFn: () => api.getAssignedExam(examId!),
    enabled: !!examId && examId !== "undefined",
  });

  const startAttemptMutation = useMutation({
    mutationFn: () => api.startAttempt(examId!),
    onSuccess: (attempt: ExamAttempt) => {
      navigate(`/candidate/runner/${attempt.id}`);
    },
    onError: (err: any) => {
      setErrorMsg(err.message || "Failed to initiate exam attempt. Please check assignment.");
    },
  });

  if (isLoading) {
    return (
      <div className="space-y-4 max-w-3xl mx-auto py-8">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (!exam) {
    return (
      <div className="text-center py-12 text-slate-400">
        <p>Assessment not available.</p>
        <Link to="/candidate">
          <Button variant="outline" className="mt-4">Back to Dashboard</Button>
        </Link>
      </div>
    );
  }

  return (
    <div className="max-w-3xl mx-auto space-y-6 py-4">
      <Link to="/candidate" className="text-xs text-slate-400 hover:text-emerald-400 flex items-center gap-1">
        <ArrowLeft className="h-3.5 w-3.5" /> Back to Candidate Dashboard
      </Link>

      <Card className="border-slate-800 bg-slate-900/90 shadow-2xl">
        <CardHeader className="space-y-3 border-b border-slate-800 pb-6">
          <div className="flex items-center justify-between">
            <Badge variant="outline" className="font-mono text-cyan-400 border-cyan-500/30 text-xs">
              CODE: {exam.code}
            </Badge>
            <Badge variant="success">READY FOR CANDIDATE</Badge>
          </div>
          <CardTitle className="text-2xl font-bold text-white">{exam.title}</CardTitle>
          <p className="text-slate-400 text-sm">{exam.description || "Official assessment examination."}</p>
        </CardHeader>

        <CardContent className="space-y-6 pt-6">
          <div className="grid grid-cols-3 gap-4 p-4 rounded-xl bg-slate-950/80 border border-slate-800 text-center">
            <div>
              <span className="text-xs text-slate-400 block uppercase font-mono">Time Limit</span>
              <span className="text-xl font-bold text-cyan-400 font-mono">{exam.duration_minutes} Mins</span>
            </div>
            <div>
              <span className="text-xs text-slate-400 block uppercase font-mono">Passing Score</span>
              <span className="text-xl font-bold text-emerald-400 font-mono">{exam.pass_marks} pts</span>
            </div>
            <div>
              <span className="text-xs text-slate-400 block uppercase font-mono">Total Points</span>
              <span className="text-xl font-bold text-purple-400 font-mono">{exam.total_marks} pts</span>
            </div>
          </div>

          <div className="space-y-3">
            <h3 className="text-sm font-semibold text-slate-200 uppercase tracking-wider flex items-center gap-2">
              <ShieldAlert className="h-4 w-4 text-cyan-400" />
              Examination Rules & Instructions
            </h3>
            <div className="p-4 rounded-lg bg-slate-950/90 border border-slate-800 text-xs text-slate-300 leading-relaxed font-mono whitespace-pre-wrap">
              {exam.instructions ||
                `1. Once started, the timer will count down continuously.\n2. Server time is authoritative. Answers are automatically saved.\n3. Do not close or refresh the window during the exam.\n4. You can mark questions for review and return to them prior to submission.`}
            </div>
          </div>

          {errorMsg && (
            <div className="p-3 rounded-lg bg-red-500/10 border border-red-500/30 text-red-400 text-xs flex items-center gap-2">
              <AlertCircle className="h-4 w-4 shrink-0" />
              <span>{errorMsg}</span>
            </div>
          )}
        </CardContent>

        <CardFooter className="flex flex-col sm:flex-row items-center justify-between gap-4 border-t border-slate-800 pt-6">
          <div className="text-xs text-slate-400">
            By clicking "Begin Assessment", your attempt will be initialized on the server.
          </div>
          <Button
            onClick={() => setConfirmOpen(true)}
            size="lg"
            className="w-full sm:w-auto bg-cyan-600 hover:bg-cyan-500 text-white font-semibold glow-cyan px-8"
          >
            <Play className="h-4 w-4 mr-2 fill-current" />
            Begin Assessment
          </Button>
        </CardFooter>
      </Card>

      <Dialog open={confirmOpen} onOpenChange={setConfirmOpen}>
        <DialogContent onClose={() => setConfirmOpen(false)}>
          <DialogHeader>
            <DialogTitle>Confirm Assessment Start</DialogTitle>
            <DialogDescription>
              Are you ready to begin? The timer will start immediately upon confirmation.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setConfirmOpen(false)}>
              Cancel
            </Button>
            <Button
              onClick={() => {
                setConfirmOpen(false);
                startAttemptMutation.mutate();
              }}
              disabled={startAttemptMutation.isPending}
              className="bg-cyan-600 hover:bg-cyan-500"
            >
              {startAttemptMutation.isPending ? "Initializing..." : "Start Now"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
