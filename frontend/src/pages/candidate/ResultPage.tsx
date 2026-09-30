
import { useParams, Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { ArrowRight, ShieldCheck } from "lucide-react";
import { api } from "../../lib/api";
import { Card, CardHeader, CardTitle, CardContent, CardFooter } from "../../components/ui/card";
import { Button } from "../../components/ui/button";
import { Badge } from "../../components/ui/badge";
import { Skeleton } from "../../components/ui/skeleton";
import { formatDate } from "../../lib/utils";

export function ResultPage() {
  const { attemptId } = useParams<{ attemptId: string }>();

  const { data: runtime, isLoading } = useQuery({
    queryKey: ["attempt-result", attemptId],
    queryFn: () => api.getAttemptRuntime(attemptId!),
    enabled: !!attemptId && attemptId !== "undefined",
  });

  if (isLoading) {
    return (
      <div className="max-w-xl mx-auto py-12 space-y-4">
        <Skeleton className="h-64 w-full rounded-2xl" />
      </div>
    );
  }

  const attempt = runtime?.attempt;
  const exam = runtime?.exam;

  return (
    <div className="max-w-xl mx-auto py-8 space-y-6">
      <Card className="border-slate-800 bg-slate-900/90 shadow-2xl text-center overflow-hidden relative">
        <div className="absolute -top-24 left-1/2 -translate-x-1/2 w-72 h-72 bg-emerald-500/10 rounded-full blur-3xl pointer-events-none" />

        <CardHeader className="space-y-3 pt-8 pb-4">
          <div className="inline-flex h-16 w-16 items-center justify-center rounded-2xl bg-emerald-500/10 border border-emerald-500/30 text-emerald-400 mx-auto glow-success">
            <ShieldCheck className="h-10 w-10" />
          </div>
          <Badge variant="success" className="mx-auto text-xs">
            ASSESSMENT SUBMITTED
          </Badge>
          <CardTitle className="text-2xl font-bold text-white">
            {exam?.title || "Assessment Completed"}
          </CardTitle>
          <p className="text-slate-400 text-xs font-mono">
            Attempt ID: {attemptId} &bull; Finalized State Verified
          </p>
        </CardHeader>

        <CardContent className="space-y-6 pt-2">
          {/* Submission Info Box */}
          <div className="p-4 rounded-xl bg-slate-950/80 border border-slate-800 space-y-3 text-xs font-mono">
            <div className="flex justify-between border-b border-slate-800 pb-2">
              <span className="text-slate-400">Submission Status:</span>
              <span className="text-emerald-400 font-bold">{attempt?.status || "SUBMITTED"}</span>
            </div>
            <div className="flex justify-between border-b border-slate-800 pb-2">
              <span className="text-slate-400">Started At:</span>
              <span className="text-slate-200">{formatDate(attempt?.started_at)}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-slate-400">Submitted At:</span>
              <span className="text-slate-200">{formatDate(attempt?.submitted_at || new Date().toISOString())}</span>
            </div>
          </div>

          <div className="text-xs text-slate-400 leading-relaxed bg-slate-950/50 p-3 rounded-lg border border-slate-800/80">
            ✓ Your responses have been safely recorded in authoritative PostgreSQL storage and broadcast to the NATS JetStream assessment stream.
          </div>
        </CardContent>

        <CardFooter className="pt-2 pb-6 flex justify-center">
          <Link to="/candidate">
            <Button className="bg-cyan-600 hover:bg-cyan-500 text-white glow-cyan">
              Return to Candidate Portal <ArrowRight className="h-4 w-4 ml-2" />
            </Button>
          </Link>
        </CardFooter>
      </Card>
    </div>
  );
}
