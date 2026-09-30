import { useState, useEffect } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useQuery, useMutation } from "@tanstack/react-query";
import {
  ShieldAlert,
  Clock,
  AlertTriangle,
  ChevronLeft,
  ChevronRight,
  Send,
  Wifi,
  RefreshCw,
  Star,
  Check,
} from "lucide-react";
import { api, RuntimeQuestion, CandidateResponse } from "../../lib/api";
import { eventWal } from "../../lib/eventWal";
import { runtimeCache } from "../../lib/runtimeCache";
import { Button } from "../../components/ui/button";
import { Badge } from "../../components/ui/badge";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "../../components/ui/dialog";
import { formatSeconds, cn } from "../../lib/utils";

export function AssessmentRunnerPage() {
  const { attemptId } = useParams<{ attemptId: string }>();
  const navigate = useNavigate();

  // State
  const [currentIdx, setCurrentIdx] = useState(0);
  const [answers, setAnswers] = useState<Record<string, any>>({});
  const [reviewFlags, setReviewFlags] = useState<Record<string, boolean>>({});
  const [saveStatus, setSaveStatus] = useState<"saved" | "saving" | "error">("saved");

  // Timer state
  const [secondsRemaining, setSecondsRemaining] = useState<number | null>(null);

  // Heartbeat & Session state
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [connectionStatus, setConnectionStatus] = useState<"connected" | "reconnecting">("connected");

  // Submit modal
  const [submitDialogOpen, setSubmitDialogOpen] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Fetch Attempt Runtime
  const { data: runtime, isLoading, isError, error } = useQuery({
    queryKey: ["attempt-runtime", attemptId],
    queryFn: () => api.getAttemptRuntime(attemptId!),
    enabled: !!attemptId && attemptId !== "undefined",
    refetchOnWindowFocus: false,
  });

  // 1. Initialize session on load
  useEffect(() => {
    if (!attemptId) return;
    let mounted = true;

    api
      .createSession(attemptId, { client_info: { userAgent: navigator.userAgent } })
      .then((sess: any) => {
        if (mounted && sess?.id) setSessionId(sess.id);
      })
      .catch((err: any) => {
        console.warn("Session creation warn:", err);
      });

    return () => {
      mounted = false;
    };
  }, [attemptId]);

  // 2. Heartbeat interval
  useEffect(() => {
    if (!sessionId) return;
    const interval = setInterval(() => {
      api
        .sendHeartbeat(sessionId, { network_quality: "GOOD" })
        .then(() => setConnectionStatus("connected"))
        .catch(() => setConnectionStatus("reconnecting"));
    }, 15000);

    return () => clearInterval(interval);
  }, [sessionId]);

  // 3. Sync existing responses & server timer + Initialize WAL & Cache
  useEffect(() => {
    if (!runtime || !attemptId) return;

    // Save fetched runtime to IndexedDB runtime cache
    runtimeCache.saveRuntime(runtime).catch(console.error);

    // Initialize local event sequence and recover any stale SYNCING events
    eventWal.initSequence(attemptId).then(() => {
      return eventWal.recoverOnStartup(attemptId);
    }).catch(console.error);

    // Load initial responses
    const initialAnswers: Record<string, any> = {};
    (runtime.responses || []).forEach((resp: CandidateResponse) => {
      initialAnswers[resp.question_version_id] = resp.value;
    });
    setAnswers(initialAnswers);

    // Calculate remaining time from server's expires_at
    if (runtime.attempt?.expires_at) {
      const expiresAtMs = new Date(runtime.attempt.expires_at).getTime();
      const diffSec = Math.max(0, Math.floor((expiresAtMs - Date.now()) / 1000));
      setSecondsRemaining(diffSec);
    }
  }, [runtime, attemptId]);

  // 4. Timer Countdown Tick
  useEffect(() => {
    if (secondsRemaining === null || secondsRemaining <= 0) return;

    const timer = setInterval(() => {
      setSecondsRemaining((prev) => {
        if (prev === null || prev <= 1) {
          clearInterval(timer);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);

    return () => clearInterval(timer);
  }, [secondsRemaining]);

  // Save response mutation with IndexedDB WAL write before network request
  const saveResponseMutation = useMutation({
    mutationFn: async ({ qvId, value }: { qvId: string; value: any }) => {
      // 1. Durably record event in IndexedDB WAL FIRST
      if (attemptId && sessionId) {
        await eventWal.recordEvent({
          attemptId,
          sessionId,
          sessionEpoch: runtime?.session?.session_epoch || 1,
          eventType: 'sentinel.candidate.option_selected',
          payload: { question_version_id: qvId, value },
        });
        // Update local runtime cache
        await runtimeCache.updateCachedResponse(attemptId, qvId, value);
      }
      // 2. Attempt network transmission
      return api.saveResponse(attemptId!, { question_version_id: qvId, value });
    },
    onMutate: () => setSaveStatus("saving"),
    onSuccess: () => setSaveStatus("saved"),
    onError: () => setSaveStatus("error"),
  });

  const handleAnswerChange = (qvId: string, val: any) => {
    setAnswers((prev) => ({ ...prev, [qvId]: val }));
    saveResponseMutation.mutate({ qvId, value: val });
  };

  const toggleReview = (qvId: string) => {
    const nextFlag = !reviewFlags[qvId];
    setReviewFlags((prev) => ({ ...prev, [qvId]: nextFlag }));

    if (attemptId && sessionId) {
      eventWal.recordEvent({
        attemptId,
        sessionId,
        sessionEpoch: runtime?.session?.session_epoch || 1,
        eventType: 'sentinel.candidate.review_flag_toggled',
        payload: { question_version_id: qvId, value: { flagged: nextFlag } },
      }).catch(console.error);
    }
  };

  const handleSubmitExam = async () => {
    setIsSubmitting(true);
    try {
      // Record submission intent event locally in WAL first
      if (attemptId && sessionId) {
        await eventWal.recordEvent({
          attemptId,
          sessionId,
          sessionEpoch: runtime?.session?.session_epoch || 1,
          eventType: 'sentinel.candidate.submission_intended',
          payload: { action: 'submit_clicked' },
        });
      }
      await api.submitAttempt(attemptId!);
      navigate(`/candidate/attempts/${attemptId}/result`);
    } catch (err: any) {
      console.error(err);
      setIsSubmitting(false);
      alert(err.message || "Failed to submit exam.");
    }
  };

  if (isLoading) {
    return (
      <div className="min-h-screen bg-slate-950 flex flex-col items-center justify-center space-y-4">
        <RefreshCw className="h-8 w-8 text-cyan-400 animate-spin" />
        <span className="text-slate-300 text-sm font-mono">Loading Secure Runtime Environment...</span>
      </div>
    );
  }

  if (isError || !runtime || !runtime.questions || runtime.questions.length === 0) {
    return (
      <div className="min-h-screen bg-slate-950 flex flex-col items-center justify-center p-4">
        <div className="max-w-md text-center space-y-4">
          <ShieldAlert className="h-12 w-12 text-red-500 mx-auto" />
          <h2 className="text-xl font-bold text-white">Runtime Error</h2>
          <p className="text-slate-400 text-xs font-mono">
            {(error as any)?.message || "Unable to load exam runtime questions."}
          </p>
          <Button variant="outline" onClick={() => navigate("/candidate")}>
            Return to Candidate Portal
          </Button>
        </div>
      </div>
    );
  }

  const questions = runtime.questions;
  const currentQuestion: RuntimeQuestion = questions[currentIdx];
  const currentQVId = currentQuestion.question_version_id;
  const currentAnswer = answers[currentQVId];

  // Palette counts
  const totalQuestions = questions.length;
  const answeredCount = Object.keys(answers).filter(
    (k) => answers[k] !== undefined && answers[k] !== null && answers[k] !== ""
  ).length;
  const reviewCount = Object.values(reviewFlags).filter(Boolean).length;
  const unansweredCount = totalQuestions - answeredCount;

  // Timer color threshold
  const isTimerWarning = secondsRemaining !== null && secondsRemaining < 300;
  const isTimerCritical = secondsRemaining !== null && secondsRemaining < 60;

  return (
    <div className="min-h-screen bg-slate-950 flex flex-col text-slate-100 font-sans select-none">
      {/* Assessment Header Bar */}
      <header className="h-16 border-b border-slate-800 bg-slate-900/90 backdrop-blur-md px-4 lg:px-6 flex items-center justify-between sticky top-0 z-40">
        <div className="flex items-center gap-3">
          <div className="h-8 w-8 rounded-lg bg-cyan-500/10 border border-cyan-500/30 flex items-center justify-center text-cyan-400">
            <ShieldAlert className="h-4 w-4" />
          </div>
          <div>
            <h1 className="font-bold text-white text-base leading-tight truncate max-w-xs md:max-w-md">
              {runtime.exam.title}
            </h1>
            <div className="text-[10px] text-slate-400 font-mono">
              CODE: {runtime.exam.code} &bull; ATTEMPT: {attemptId?.slice(0, 8)}
            </div>
          </div>
        </div>

        {/* Center/Right Info: Timer, Connection, Save state */}
        <div className="flex items-center gap-4">
          <div className="hidden sm:flex items-center gap-1.5 text-xs text-slate-400">
            <Wifi className={cn("h-3.5 w-3.5", connectionStatus === "connected" ? "text-emerald-400" : "text-amber-400 animate-pulse")} />
            <span className="text-[11px] font-mono">{connectionStatus === "connected" ? "Connected" : "Reconnecting"}</span>
          </div>

          <div className="hidden md:flex items-center gap-1.5 text-xs px-2.5 py-1 rounded bg-slate-950 border border-slate-800 font-mono">
            {saveStatus === "saving" && (
              <>
                <RefreshCw className="h-3 w-3 text-cyan-400 animate-spin" />
                <span className="text-cyan-400">Saving...</span>
              </>
            )}
            {saveStatus === "saved" && (
              <>
                <Check className="h-3 w-3 text-emerald-400" />
                <span className="text-emerald-400">Saved</span>
              </>
            )}
            {saveStatus === "error" && (
              <>
                <AlertTriangle className="h-3 w-3 text-red-400" />
                <span className="text-red-400">Save failed</span>
              </>
            )}
          </div>

          <div
            className={cn(
              "flex items-center gap-2 px-3 py-1.5 rounded-lg border font-mono font-bold text-sm transition-all",
              isTimerCritical
                ? "bg-red-500/20 text-red-400 border-red-500/40 glow-destructive animate-pulse"
                : isTimerWarning
                ? "bg-amber-500/15 text-amber-300 border-amber-500/30"
                : "bg-slate-950 text-cyan-400 border-slate-800"
            )}
          >
            <Clock className="h-4 w-4" />
            <span>{secondsRemaining !== null ? formatSeconds(secondsRemaining) : "--:--"}</span>
          </div>

          <Button
            onClick={() => setSubmitDialogOpen(true)}
            className="bg-emerald-600 hover:bg-emerald-500 text-white font-semibold text-xs glow-success px-4"
          >
            <Send className="h-3.5 w-3.5 mr-1.5" /> Submit Exam
          </Button>
        </div>
      </header>

      {/* Main Runner Body */}
      <div className="flex-1 flex flex-col md:flex-row overflow-hidden">
        {/* Left Side: Question Navigation Palette */}
        <aside className="w-full md:w-72 border-r border-slate-800 bg-slate-900/60 p-4 flex flex-col gap-4 shrink-0 overflow-y-auto max-h-48 md:max-h-none">
          <div className="flex items-center justify-between border-b border-slate-800/80 pb-2">
            <span className="text-xs font-semibold uppercase text-slate-400 tracking-wider">
              Question Palette
            </span>
            <span className="text-xs font-mono text-cyan-400 font-bold">
              {answeredCount}/{totalQuestions} Answered
            </span>
          </div>

          <div className="grid grid-cols-5 gap-2">
            {questions.map((q: RuntimeQuestion, idx: number) => {
              const qvId = q.question_version_id;
              const isCurrent = idx === currentIdx;
              const hasAnswer = answers[qvId] !== undefined && answers[qvId] !== null && answers[qvId] !== "";
              const isFlagged = reviewFlags[qvId];

              return (
                <button
                  key={qvId}
                  onClick={() => setCurrentIdx(idx)}
                  className={cn(
                    "h-9 rounded-md text-xs font-mono font-semibold transition-all relative flex items-center justify-center border",
                    isCurrent
                      ? "ring-2 ring-cyan-400 border-cyan-500 bg-slate-800 text-white shadow-lg"
                      : hasAnswer && isFlagged
                      ? "bg-amber-500/20 border-amber-500/50 text-amber-300"
                      : hasAnswer
                      ? "bg-cyan-500/20 border-cyan-500/40 text-cyan-300"
                      : isFlagged
                      ? "bg-amber-500/10 border-amber-500/30 text-amber-400"
                      : "bg-slate-950 border-slate-800 text-slate-400 hover:border-slate-700 hover:text-slate-200"
                  )}
                >
                  {(idx + 1).toString().padStart(2, "0")}
                  {isFlagged && (
                    <Star className="h-2.5 w-2.5 absolute top-0.5 right-0.5 fill-amber-400 text-amber-400" />
                  )}
                </button>
              );
            })}
          </div>

          <div className="mt-auto pt-4 border-t border-slate-800/80 space-y-2 text-[11px] text-slate-400 font-mono">
            <div className="flex items-center gap-2">
              <span className="h-2.5 w-2.5 rounded-full bg-cyan-500/40 border border-cyan-400" />
              <span>Answered</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="h-2.5 w-2.5 rounded-full bg-slate-950 border border-slate-800" />
              <span>Unanswered</span>
            </div>
            <div className="flex items-center gap-2">
              <Star className="h-3 w-3 text-amber-400 fill-amber-400" />
              <span>Marked for Review</span>
            </div>
          </div>
        </aside>

        {/* Right Side: Main Question Display Container */}
        <main className="flex-1 flex flex-col overflow-y-auto p-4 lg:p-8 max-w-4xl mx-auto w-full">
          <div className="space-y-6 flex-1 flex flex-col justify-between">
            <div className="space-y-4">
              <div className="flex items-center justify-between border-b border-slate-800 pb-3">
                <div className="flex items-center gap-2">
                  <span className="text-lg font-bold text-white font-mono">
                    Question {currentIdx + 1} of {totalQuestions}
                  </span>
                  <Badge variant="outline" className="text-xs text-slate-400 border-slate-700">
                    {currentQuestion.marks} points
                  </Badge>
                  <Badge variant="secondary" className="text-xs">
                    {currentQuestion.type}
                  </Badge>
                </div>

                <Button
                  variant={reviewFlags[currentQVId] ? "secondary" : "outline"}
                  size="sm"
                  onClick={() => toggleReview(currentQVId)}
                  className={cn(
                    "text-xs transition-colors",
                    reviewFlags[currentQVId]
                      ? "bg-amber-500/20 text-amber-300 border-amber-500/40"
                      : "text-slate-400 hover:text-white"
                  )}
                >
                  <Star className={cn("h-3.5 w-3.5 mr-1.5", reviewFlags[currentQVId] && "fill-amber-400")} />
                  {reviewFlags[currentQVId] ? "Marked for Review" : "Mark Review"}
                </Button>
              </div>

              <div className="text-base lg:text-lg text-slate-100 font-medium leading-relaxed bg-slate-900/60 p-6 rounded-xl border border-slate-800">
                {currentQuestion.content?.stem || "Question text unavailable."}
              </div>

              <div className="pt-2">
                {currentQuestion.type === "SINGLE_CHOICE" && (
                  <div className="space-y-3">
                    {(currentQuestion.content?.options || []).map((opt: any) => {
                      const isSelected = currentAnswer?.selected === opt.key;
                      return (
                        <div
                          key={opt.key}
                          onClick={() => handleAnswerChange(currentQVId, { selected: opt.key })}
                          className={cn(
                            "flex items-center gap-4 p-4 rounded-xl border cursor-pointer transition-all",
                            isSelected
                              ? "bg-cyan-500/10 border-cyan-500 text-white shadow-md glow-cyan"
                              : "bg-slate-900/40 border-slate-800 text-slate-300 hover:border-slate-700 hover:bg-slate-800/40"
                          )}
                        >
                          <div
                            className={cn(
                              "h-5 w-5 rounded-full border flex items-center justify-center text-xs font-mono font-bold shrink-0",
                              isSelected ? "border-cyan-400 bg-cyan-500 text-slate-950" : "border-slate-600 text-slate-400"
                            )}
                          >
                            {opt.key}
                          </div>
                          <span className="text-sm font-medium flex-1">{opt.text}</span>
                        </div>
                      );
                    })}
                  </div>
                )}

                {currentQuestion.type === "MULTIPLE_CHOICE" && (
                  <div className="space-y-3">
                    {(currentQuestion.content?.options || []).map((opt: any) => {
                      const selectedKeys: string[] = currentAnswer?.selected || [];
                      const isChecked = selectedKeys.includes(opt.key);

                      const handleMultiToggle = () => {
                        let updated = [...selectedKeys];
                        if (isChecked) {
                          updated = updated.filter((k) => k !== opt.key);
                        } else {
                          updated.push(opt.key);
                        }
                        handleAnswerChange(currentQVId, { selected: updated });
                      };

                      return (
                        <div
                          key={opt.key}
                          onClick={handleMultiToggle}
                          className={cn(
                            "flex items-center gap-4 p-4 rounded-xl border cursor-pointer transition-all",
                            isChecked
                              ? "bg-purple-500/10 border-purple-500 text-white shadow-md"
                              : "bg-slate-900/40 border-slate-800 text-slate-300 hover:border-slate-700 hover:bg-slate-800/40"
                          )}
                        >
                          <div
                            className={cn(
                              "h-5 w-5 rounded border flex items-center justify-center text-xs font-mono font-bold shrink-0",
                              isChecked ? "border-purple-400 bg-purple-500 text-white" : "border-slate-600 text-slate-400"
                            )}
                          >
                            {isChecked ? <Check className="h-3.5 w-3.5" /> : opt.key}
                          </div>
                          <span className="text-sm font-medium flex-1">{opt.text}</span>
                        </div>
                      );
                    })}
                  </div>
                )}

                {currentQuestion.type === "TRUE_FALSE" && (
                  <div className="grid grid-cols-2 gap-4">
                    {[true, false].map((val) => {
                      const isSelected = currentAnswer?.value === val;
                      return (
                        <button
                          key={String(val)}
                          type="button"
                          onClick={() => handleAnswerChange(currentQVId, { value: val })}
                          className={cn(
                            "p-6 rounded-xl border text-center font-bold text-lg transition-all",
                            isSelected
                              ? "bg-cyan-500/20 border-cyan-400 text-cyan-300 glow-cyan"
                              : "bg-slate-900/40 border-slate-800 text-slate-300 hover:border-slate-700"
                          )}
                        >
                          {val ? "True" : "False"}
                        </button>
                      );
                    })}
                  </div>
                )}

                {currentQuestion.type === "TEXT" && (
                  <div className="space-y-2">
                    <label className="text-xs text-slate-400">Type your answer below:</label>
                    <input
                      type="text"
                      placeholder="Enter answer..."
                      value={currentAnswer?.text || ""}
                      onChange={(e) => handleAnswerChange(currentQVId, { text: e.target.value })}
                      className="w-full rounded-xl border border-slate-700 bg-slate-900 p-4 text-slate-100 focus:outline-none focus:border-cyan-500 font-mono text-sm"
                    />
                  </div>
                )}

                {currentQuestion.type === "NUMERIC" && (
                  <div className="space-y-2">
                    <label className="text-xs text-slate-400">Enter numerical answer:</label>
                    <input
                      type="number"
                      placeholder="e.g. 42"
                      value={currentAnswer?.value ?? ""}
                      onChange={(e) =>
                        handleAnswerChange(currentQVId, { value: parseFloat(e.target.value) || 0 })
                      }
                      className="w-full max-w-xs rounded-xl border border-slate-700 bg-slate-900 p-4 text-slate-100 focus:outline-none focus:border-cyan-500 font-mono text-base"
                    />
                  </div>
                )}
              </div>
            </div>

            <div className="flex items-center justify-between border-t border-slate-800 pt-6 mt-8">
              <Button
                variant="outline"
                onClick={() => setCurrentIdx((prev) => Math.max(0, prev - 1))}
                disabled={currentIdx === 0}
                className="border-slate-800 text-slate-300 hover:bg-slate-800"
              >
                <ChevronLeft className="h-4 w-4 mr-1" /> Previous Question
              </Button>

              <div className="text-xs text-slate-500 font-mono">
                Use controls to navigate
              </div>

              <Button
                onClick={() => setCurrentIdx((prev) => Math.min(totalQuestions - 1, prev + 1))}
                disabled={currentIdx === totalQuestions - 1}
                className="bg-cyan-600 hover:bg-cyan-500 text-white"
              >
                Next Question <ChevronRight className="h-4 w-4 ml-1" />
              </Button>
            </div>
          </div>
        </main>
      </div>

      <Dialog open={submitDialogOpen} onOpenChange={setSubmitDialogOpen}>
        <DialogContent onClose={() => setSubmitDialogOpen(false)}>
          <DialogHeader>
            <DialogTitle>Submit Assessment Exam?</DialogTitle>
            <DialogDescription>
              Are you sure you want to finalize your submission? Answers cannot be modified after submission.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-3 py-3 border-y border-slate-800 text-xs font-mono">
            <div className="flex justify-between text-slate-300">
              <span>Total Questions:</span>
              <span className="font-bold">{totalQuestions}</span>
            </div>
            <div className="flex justify-between text-emerald-400">
              <span>Answered:</span>
              <span className="font-bold">{answeredCount}</span>
            </div>
            <div className="flex justify-between text-amber-400">
              <span>Marked for Review:</span>
              <span className="font-bold">{reviewCount}</span>
            </div>
            <div className="flex justify-between text-slate-400">
              <span>Unanswered:</span>
              <span className="font-bold">{unansweredCount}</span>
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setSubmitDialogOpen(false)}>
              Continue Assessment
            </Button>
            <Button
              onClick={handleSubmitExam}
              disabled={isSubmitting}
              className="bg-emerald-600 hover:bg-emerald-500 text-white font-semibold glow-success"
            >
              {isSubmitting ? "Finalizing..." : "Submit Exam Now"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
