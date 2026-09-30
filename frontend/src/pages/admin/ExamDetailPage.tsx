import React, { useState } from "react";
import { useParams, Link } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, CheckCircle, Plus, AlertCircle } from "lucide-react";
import { api, AttachQuestionInput, AssignCandidateInput, ExamQuestion, ExamAssignment, ExamAttempt } from "../../lib/api";
import { Card, CardHeader, CardTitle, CardContent } from "../../components/ui/card";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Badge } from "../../components/ui/badge";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "../../components/ui/tabs";
import { Table, TableHeader, TableRow, TableHead, TableBody, TableCell } from "../../components/ui/table";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "../../components/ui/dialog";
import { Skeleton } from "../../components/ui/skeleton";
import { useAuth } from "../../hooks/useAuth";
import { formatDate } from "../../lib/utils";

export function ExamDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { role } = useAuth();
  const queryClient = useQueryClient();
  const [activeTab, setActiveTab] = useState("overview");

  // Modal dialog states
  const [addQuestionModalOpen, setAddQuestionModalOpen] = useState(false);
  const [assignCandidateModalOpen, setAssignCandidateModalOpen] = useState(false);
  const [attachError, setAttachError] = useState<string | null>(null);
  const [assignError, setAssignError] = useState<string | null>(null);

  // Form states
  const [selectedQuestionVersionId, setSelectedQuestionVersionId] = useState("");
  const [questionPosition, setQuestionPosition] = useState(1);
  const [selectedCandidateId, setSelectedCandidateId] = useState("");

  const { data: exam, isLoading: isLoadingExam } = useQuery({
    queryKey: ["exam", id],
    queryFn: () => api.getExam(id!),
    enabled: role === "exam_admin" && !!id && id !== "undefined",
  });

  const { data: candidates } = useQuery({
    queryKey: ["candidates"],
    queryFn: () => api.listCandidates(),
    enabled: role === "exam_admin",
  });

  const publishMutation = useMutation({
    mutationFn: () => api.publishExam(id!),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["exam", id] }),
    onError: (err: any) => {
      alert(err.message || "Failed to publish assessment. Ensure at least one question is attached.");
    },
  });

  const startExamMutation = useMutation({
    mutationFn: () => api.startExam(id!),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["exam", id] }),
    onError: (err: any) => {
      alert(err.message || "Failed to start assessment.");
    },
  });

  const pauseExamMutation = useMutation({
    mutationFn: () => api.pauseExam(id!),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["exam", id] }),
    onError: (err: any) => {
      alert(err.message || "Failed to pause assessment.");
    },
  });

  const resumeExamMutation = useMutation({
    mutationFn: () => api.resumeExam(id!),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["exam", id] }),
    onError: (err: any) => {
      alert(err.message || "Failed to resume assessment.");
    },
  });

  const attachQuestionMutation = useMutation({
    mutationFn: (input: AttachQuestionInput) => api.attachQuestion(id!, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["exam", id] });
      setAddQuestionModalOpen(false);
      setSelectedQuestionVersionId("");
      setAttachError(null);
    },
    onError: (err: any) => {
      console.error("[ExamDetailPage] Attach question error:", err);
      setAttachError(err.message || "Failed to attach question version");
    },
  });

  const assignCandidateMutation = useMutation({
    mutationFn: (input: AssignCandidateInput) => api.assignCandidate(id!, input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["exam", id] });
      setAssignCandidateModalOpen(false);
      setSelectedCandidateId("");
      setAssignError(null);
    },
    onError: (err: any) => {
      console.error("[ExamDetailPage] Assign candidate error:", err);
      setAssignError(err.message || "Failed to assign candidate");
    },
  });

  const revokeAssignmentMutation = useMutation({
    mutationFn: (candidateId: string) => api.revokeAssignment(id!, candidateId),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["exam", id] }),
    onError: (err: any) => {
      alert(err.message || "Failed to revoke assignment");
    },
  });

  if (isLoadingExam) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-32 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (!exam) {
    return (
      <div className="text-center py-12 text-slate-400">
        <p>Exam not found.</p>
        <Link to="/admin/exams">
          <Button variant="outline" className="mt-4">Back to Exams</Button>
        </Link>
      </div>
    );
  }

  const handleAttachQuestion = (e: React.FormEvent) => {
    e.preventDefault();
    setAttachError(null);
    if (!selectedQuestionVersionId.trim()) {
      setAttachError("Please enter a valid Question Version UUID.");
      return;
    }
    attachQuestionMutation.mutate({
      question_version_id: selectedQuestionVersionId.trim(),
      position: Number(questionPosition) || (exam.questions?.length || 0) + 1,
    });
  };

  const handleAssignCandidate = (e: React.FormEvent) => {
    e.preventDefault();
    setAssignError(null);
    if (!selectedCandidateId) {
      setAssignError("Please select a candidate to assign.");
      return;
    }
    assignCandidateMutation.mutate({ candidate_id: selectedCandidateId });
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-slate-800 pb-5">
        <div className="space-y-1">
          <Link to="/admin/exams" className="text-xs text-slate-400 hover:text-cyan-400 flex items-center gap-1 mb-2">
            <ArrowLeft className="h-3.5 w-3.5" /> Back to Assessments
          </Link>
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold text-white">{exam.title}</h1>
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
          </div>
          <div className="text-xs text-slate-400 font-mono">
            Exam ID: {exam.id}
          </div>
        </div>

        <div className="flex items-center gap-3">
          {exam.status === "DRAFT" && (
            <Button
              onClick={() => publishMutation.mutate()}
              disabled={publishMutation.isPending}
              className="bg-emerald-600 hover:bg-emerald-500 text-white shadow glow-success"
            >
              <CheckCircle className="h-4 w-4 mr-2" />
              Publish Assessment
            </Button>
          )}

          {exam.status === "SCHEDULED" && (
            <Button
              onClick={() => startExamMutation.mutate()}
              disabled={startExamMutation.isPending}
              className="bg-cyan-600 hover:bg-cyan-500 text-white shadow glow-cyan"
            >
              Start Exam (Make LIVE)
            </Button>
          )}

          {exam.status === "LIVE" && (
            <Button
              onClick={() => pauseExamMutation.mutate()}
              disabled={pauseExamMutation.isPending}
              variant="outline"
              className="border-amber-500/40 text-amber-300 hover:bg-amber-500/10"
            >
              Pause Exam
            </Button>
          )}

          {exam.status === "PAUSED" && (
            <Button
              onClick={() => resumeExamMutation.mutate()}
              disabled={resumeExamMutation.isPending}
              className="bg-cyan-600 hover:bg-cyan-500 text-white shadow glow-cyan"
            >
              Resume Exam (LIVE)
            </Button>
          )}
        </div>
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList className="bg-slate-900 border-slate-800">
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="questions">Questions ({exam.questions?.length || 0})</TabsTrigger>
          <TabsTrigger value="assignments">Assignments ({exam.assignments?.length || 0})</TabsTrigger>
          <TabsTrigger value="attempts">Attempts ({exam.attempts?.length || 0})</TabsTrigger>
        </TabsList>

        <TabsContent value="overview">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <Card className="md:col-span-2 border-slate-800 bg-slate-900/60">
              <CardHeader>
                <CardTitle className="text-lg text-white">Assessment Rules & Details</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div>
                  <h4 className="text-xs uppercase text-slate-400 font-semibold tracking-wider">Description</h4>
                  <p className="text-sm text-slate-200 mt-1">{exam.description || "No description provided."}</p>
                </div>

                <div>
                  <h4 className="text-xs uppercase text-slate-400 font-semibold tracking-wider">Candidate Instructions</h4>
                  <div className="mt-1 p-3 rounded-md bg-slate-950/80 border border-slate-800 text-xs text-slate-300 font-mono whitespace-pre-wrap">
                    {exam.instructions || "Standard test protocol applies."}
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-4 pt-4 border-t border-slate-800">
                  <div>
                    <span className="text-xs text-slate-400 block">Duration</span>
                    <span className="text-lg font-bold text-cyan-400 font-mono">{exam.duration_minutes || 60} mins</span>
                  </div>
                  <div>
                    <span className="text-xs text-slate-400 block">Status</span>
                    <span className="text-lg font-bold text-white font-mono">{exam.status}</span>
                  </div>
                </div>
              </CardContent>
            </Card>

            <Card className="border-slate-800 bg-slate-900/60">
              <CardHeader>
                <CardTitle className="text-sm text-slate-300">Exam Lifecycle Status</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3 text-xs">
                <div className="flex justify-between py-1.5 border-b border-slate-800">
                  <span className="text-slate-400">Created:</span>
                  <span className="text-slate-200">{formatDate(exam.created_at)}</span>
                </div>
                <div className="flex justify-between py-1.5 border-b border-slate-800">
                  <span className="text-slate-400">Status:</span>
                  <span className="font-semibold text-cyan-400">{exam.status}</span>
                </div>
                <div className="flex justify-between py-1.5 border-b border-slate-800">
                  <span className="text-slate-400">Questions Attached:</span>
                  <span className="text-slate-200">{exam.questions?.length || 0}</span>
                </div>
                <div className="flex justify-between py-1.5">
                  <span className="text-slate-400">Candidates Assigned:</span>
                  <span className="text-slate-200">{exam.assignments?.length || 0}</span>
                </div>
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="questions">
          <div className="space-y-4">
            <div className="flex justify-between items-center">
              <h3 className="text-base font-semibold text-white">Attached Question Versions</h3>
              {exam.status === "DRAFT" && (
                <Button
                  onClick={() => { setAttachError(null); setAddQuestionModalOpen(true); }}
                  className="bg-cyan-600 hover:bg-cyan-500 text-xs"
                >
                  <Plus className="h-3.5 w-3.5 mr-1" /> Attach Question
                </Button>
              )}
            </div>

            <Card className="border-slate-800 bg-slate-900/60">
              <CardContent className="p-0">
                {!exam.questions || exam.questions.length === 0 ? (
                  <div className="p-8 text-center text-slate-400 text-sm">
                    No questions attached to this assessment yet.
                  </div>
                ) : (
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Pos</TableHead>
                        <TableHead>Version ID</TableHead>
                        <TableHead>Question Prompt</TableHead>
                        <TableHead>Marks</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {exam.questions.map((q: ExamQuestion, idx: number) => (
                        <TableRow key={q.id || `eq-${q.question_version_id}-${idx}`}>
                          <TableCell className="font-mono text-xs">{q.position || idx + 1}</TableCell>
                          <TableCell className="font-mono text-xs text-cyan-400">
                            {q.question_version_id}
                          </TableCell>
                          <TableCell className="text-xs text-slate-200 max-w-xs truncate">
                            {q.question_version?.content?.stem || "Question stem prompt"}
                          </TableCell>
                          <TableCell className="font-mono text-xs text-white">
                            {q.marks || q.question_version?.marks || 10}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                )}
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="assignments">
          <div className="space-y-4">
            <div className="flex justify-between items-center">
              <h3 className="text-base font-semibold text-white">Assigned Candidates</h3>
              {(exam.status === "SCHEDULED" || exam.status === "LIVE") && (
                <Button
                  onClick={() => { setAssignError(null); setAssignCandidateModalOpen(true); }}
                  className="bg-indigo-600 hover:bg-indigo-500 text-xs"
                >
                  <Plus className="h-3.5 w-3.5 mr-1" /> Assign Candidate
                </Button>
              )}
            </div>

            <Card className="border-slate-800 bg-slate-900/60">
              <CardContent className="p-0">
                {!exam.assignments || exam.assignments.length === 0 ? (
                  <div className="p-8 text-center text-slate-400 text-sm">
                    No candidates assigned yet.
                  </div>
                ) : (
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Candidate UUID</TableHead>
                        <TableHead>Status</TableHead>
                        <TableHead>Assigned Date</TableHead>
                        <TableHead className="text-right">Action</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {exam.assignments.map((a: ExamAssignment, idx: number) => (
                        <TableRow key={a.id || a.candidate_id || `assign-${idx}`}>
                          <TableCell className="font-mono text-xs text-indigo-300 font-semibold">
                            {a.candidate_id}
                          </TableCell>
                          <TableCell>
                            <Badge variant={a.status === "REVOKED" ? "destructive" : "success"}>
                              {a.status || "ASSIGNED"}
                            </Badge>
                          </TableCell>
                          <TableCell className="text-slate-400 text-xs font-mono">
                            {formatDate(a.assigned_at)}
                          </TableCell>
                          <TableCell className="text-right">
                            {a.status !== "REVOKED" && (
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => revokeAssignmentMutation.mutate(a.candidate_id)}
                                className="text-red-400 hover:text-red-300 text-xs"
                              >
                                Revoke
                              </Button>
                            )}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                )}
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="attempts">
          <Card className="border-slate-800 bg-slate-900/60">
            <CardContent className="p-0">
              {!exam.attempts || exam.attempts.length === 0 ? (
                <div className="p-8 text-center text-slate-400 text-sm">
                  No candidate attempts recorded for this assessment.
                </div>
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Attempt ID</TableHead>
                      <TableHead>Candidate ID</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Started At</TableHead>
                      <TableHead>Expires At</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {exam.attempts.map((att: ExamAttempt, idx: number) => (
                      <TableRow key={att.id || `att-${idx}`}>
                        <TableCell className="font-mono text-xs text-cyan-400">{att.id}</TableCell>
                        <TableCell className="font-mono text-xs text-slate-300">{att.candidate_id}</TableCell>
                        <TableCell>
                          <Badge
                            variant={
                              att.status === "SUBMITTED" || att.status === "FINALIZED"
                                ? "success"
                                : att.status === "ACTIVE" || att.status === "IN_PROGRESS"
                                ? "warning"
                                : "outline"
                            }
                          >
                            {att.status}
                          </Badge>
                        </TableCell>
                        <TableCell className="text-slate-400 text-xs">{formatDate(att.started_at)}</TableCell>
                        <TableCell className="text-slate-400 text-xs font-mono">{formatDate(att.expires_at)}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      <Dialog open={addQuestionModalOpen} onOpenChange={setAddQuestionModalOpen}>
        <DialogContent onClose={() => setAddQuestionModalOpen(false)}>
          <DialogHeader>
            <DialogTitle>Attach Question Version</DialogTitle>
            <DialogDescription>
              Provide the Question Version UUID to attach an active question version to this assessment.
            </DialogDescription>
          </DialogHeader>

          {attachError && (
            <div className="p-3 rounded-lg bg-red-500/10 border border-red-500/30 text-red-400 text-xs flex items-center gap-2">
              <AlertCircle className="h-4 w-4 shrink-0" />
              <span>{attachError}</span>
            </div>
          )}

          <form onSubmit={handleAttachQuestion} className="space-y-4">
            <div>
              <label className="text-xs text-slate-300 font-medium">Question Version ID</label>
              <Input
                required
                placeholder="e.g. 8b3115d5-f479-4c6d-b90f-054e91f0b6bc"
                value={selectedQuestionVersionId}
                onChange={(e) => setSelectedQuestionVersionId(e.target.value)}
                className="mt-1 font-mono text-xs"
              />
              <span className="text-[10px] text-slate-500 mt-1 block">
                Copy Version UUID from Question Bank detail page.
              </span>
            </div>

            <div>
              <label className="text-xs text-slate-300 font-medium">Position / Sequence Order</label>
              <Input
                type="number"
                required
                min={1}
                value={questionPosition}
                onChange={(e) => setQuestionPosition(parseInt(e.target.value) || 1)}
                className="mt-1 font-mono text-xs"
              />
            </div>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setAddQuestionModalOpen(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={attachQuestionMutation.isPending} className="bg-cyan-600 hover:bg-cyan-500">
                {attachQuestionMutation.isPending ? "Attaching..." : "Attach Question"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={assignCandidateModalOpen} onOpenChange={setAssignCandidateModalOpen}>
        <DialogContent onClose={() => setAssignCandidateModalOpen(false)}>
          <DialogHeader>
            <DialogTitle>Assign Candidate</DialogTitle>
            <DialogDescription>
              Assign a registered candidate to take this assessment.
            </DialogDescription>
          </DialogHeader>

          {assignError && (
            <div className="p-3 rounded-lg bg-red-500/10 border border-red-500/30 text-red-400 text-xs flex items-center gap-2">
              <AlertCircle className="h-4 w-4 shrink-0" />
              <span>{assignError}</span>
            </div>
          )}

          <form onSubmit={handleAssignCandidate} className="space-y-4">
            <div>
              <label className="text-xs text-slate-300 font-medium">Select Candidate</label>
              <select
                required
                value={selectedCandidateId}
                onChange={(e) => setSelectedCandidateId(e.target.value)}
                className="w-full mt-1 rounded-md border border-slate-700 bg-slate-950 p-2 text-xs text-slate-100"
              >
                <option value="">-- Choose Candidate --</option>
                {candidates?.items?.map((c: any, idx: number) => (
                  <option key={c.id || `cand-opt-${idx}`} value={c.id}>
                    {c.full_name || `${c.first_name || ''} ${c.last_name || ''}`.trim()} ({c.email || 'No email'}) - {c.id}
                  </option>
                ))}
              </select>
            </div>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setAssignCandidateModalOpen(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={assignCandidateMutation.isPending} className="bg-indigo-600 hover:bg-indigo-500">
                {assignCandidateMutation.isPending ? "Assigning..." : "Assign Candidate"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}
