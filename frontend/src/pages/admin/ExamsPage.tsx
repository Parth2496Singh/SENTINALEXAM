import React, { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { FileSpreadsheet, Plus, Search, Eye, AlertCircle } from "lucide-react";
import { api, CreateExamInput, Exam } from "../../lib/api";
import { Card, CardContent } from "../../components/ui/card";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Textarea } from "../../components/ui/textarea";
import { Badge } from "../../components/ui/badge";
import { Table, TableHeader, TableRow, TableHead, TableBody, TableCell } from "../../components/ui/table";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "../../components/ui/dialog";
import { Skeleton } from "../../components/ui/skeleton";
import { useAuth } from "../../hooks/useAuth";
import { formatDate } from "../../lib/utils";

export function ExamsPage() {
  const { role } = useAuth();
  const queryClient = useQueryClient();
  const [search, setSearch] = useState("");
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  // Form state
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [durationMinutes, setDurationMinutes] = useState(60);
  const [passMarks, setPassMarks] = useState(50);
  const [instructions, setInstructions] = useState("Read all questions carefully before answering.");

  const { data: examsData, isLoading } = useQuery({
    queryKey: ["exams"],
    queryFn: () => api.listExams(),
    enabled: role === "exam_admin",
  });

  const createMutation = useMutation({
    mutationFn: (input: CreateExamInput) => api.createExam(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["exams"] });
      setCreateDialogOpen(false);
      resetForm();
    },
    onError: (err: any) => {
      console.error("[ExamsPage] Create error:", err);
      setFormError(err.message || "Failed to create exam");
    },
  });

  const resetForm = () => {
    setTitle("");
    setDescription("");
    setDurationMinutes(60);
    setPassMarks(50);
    setInstructions("Read all questions carefully before answering.");
    setFormError(null);
  };

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault();
    setFormError(null);

    if (!title.trim()) {
      setFormError("Exam title is required.");
      return;
    }

    if (durationMinutes <= 0) {
      setFormError("Duration must be greater than 0 minutes.");
      return;
    }

    createMutation.mutate({
      title: title.trim(),
      description: description.trim(),
      duration_minutes: durationMinutes,
      pass_marks: passMarks,
      total_marks: 100,
      instructions: instructions.trim(),
    });
  };

  const exams = (examsData?.items || []).filter(Boolean);
  const searchLower = (search || '').toLowerCase();
  const filteredExams = exams.filter(
    (e: Exam) =>
      String(e.title ?? '').toLowerCase().includes(searchLower) ||
      String(e.description ?? '').toLowerCase().includes(searchLower) ||
      String(e.code ?? '').toLowerCase().includes(searchLower)
  );

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-slate-800 pb-5">
        <div>
          <h1 className="text-2xl font-bold text-white flex items-center gap-2">
            <FileSpreadsheet className="h-6 w-6 text-cyan-400" />
            Exam Management
          </h1>
          <p className="text-slate-400 text-sm mt-1">
            Configure assessments, attach published questions, and assign candidates.
          </p>
        </div>
        <Button onClick={() => { setFormError(null); setCreateDialogOpen(true); }} className="bg-cyan-600 hover:bg-cyan-500 text-white glow-cyan">
          <Plus className="h-4 w-4 mr-2" />
          Create Exam
        </Button>
      </div>

      {/* Filter / Search Bar */}
      <div className="flex items-center gap-4">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-3 top-2.5 h-4 w-4 text-slate-500" />
          <Input
            placeholder="Search by title or description..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9 bg-slate-900 border-slate-800"
          />
        </div>
      </div>

      {/* Exam Table */}
      <Card className="border-slate-800 bg-slate-900/60">
        <CardContent className="p-0">
          {isLoading ? (
            <div className="p-6 space-y-3">
              <Skeleton className="h-12 w-full" />
              <Skeleton className="h-12 w-full" />
              <Skeleton className="h-12 w-full" />
            </div>
          ) : filteredExams.length === 0 ? (
            <div className="p-12 text-center text-slate-400">
              <p className="text-base font-medium">No assessments match your criteria.</p>
              <p className="text-xs text-slate-500 mt-1">Click "Create Exam" above to add your first assessment.</p>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Exam Title & ID</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Duration</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredExams.map((exam: Exam, idx: number) => {
                  const uniqueKey = exam.id ? `exam-${exam.id}-${idx}` : `exam-fallback-${idx}`;
                  return (
                    <TableRow key={uniqueKey}>
                    <TableCell>
                      {exam.id && exam.id !== 'undefined' ? (
                        <Link to={`/admin/exams/${exam.id}`} className="font-semibold text-white hover:text-cyan-400 transition-colors">
                          {exam.title}
                        </Link>
                      ) : (
                        <span className="font-semibold text-white">{exam.title}</span>
                      )}
                      <div className="text-xs text-slate-400 font-mono font-normal">
                        ID: {exam.id || 'N/A'}
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
                    <TableCell className="text-slate-300 text-xs font-mono">
                      {exam.duration_minutes || 60} mins
                    </TableCell>
                    <TableCell className="text-slate-400 text-xs">
                      {formatDate(exam.created_at)}
                    </TableCell>
                    <TableCell className="text-right space-x-2">
                      {exam.id && exam.id !== 'undefined' ? (
                        <Link to={`/admin/exams/${exam.id}`}>
                          <Button variant="ghost" size="sm" className="text-xs">
                            <Eye className="h-3.5 w-3.5 mr-1" /> View & Configure
                          </Button>
                        </Link>
                      ) : (
                        <Button disabled variant="ghost" size="sm" className="text-xs opacity-50">
                          Unavailable
                        </Button>
                      )}
                    </TableCell>
                  </TableRow>
                );
              })}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Create Exam Dialog */}
      <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
        <DialogContent onClose={() => setCreateDialogOpen(false)}>
          <DialogHeader>
            <DialogTitle>Create Assessment</DialogTitle>
            <DialogDescription>
              Define assessment metadata and scoring criteria.
            </DialogDescription>
          </DialogHeader>

          {formError && (
            <div className="p-3 rounded-lg bg-red-500/10 border border-red-500/30 text-red-400 text-xs flex items-center gap-2">
              <AlertCircle className="h-4 w-4 shrink-0" />
              <span>{formError}</span>
            </div>
          )}

          <form onSubmit={handleCreate} className="space-y-4">
            <div>
              <label className="text-xs text-slate-300 font-medium">Exam Title</label>
              <Input
                required
                placeholder="e.g. Computer Science Fundamentals"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                className="mt-1"
              />
            </div>

            <div>
              <label className="text-xs text-slate-300 font-medium">Description</label>
              <Textarea
                placeholder="Overview of assessment scope..."
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                className="mt-1 text-xs"
              />
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="text-xs text-slate-300 font-medium">Duration (minutes)</label>
                <Input
                  type="number"
                  required
                  min={1}
                  value={durationMinutes}
                  onChange={(e) => setDurationMinutes(parseInt(e.target.value) || 60)}
                  className="mt-1 font-mono"
                />
              </div>
              <div>
                <label className="text-xs text-slate-300 font-medium">Passing Marks</label>
                <Input
                  type="number"
                  required
                  min={0}
                  value={passMarks}
                  onChange={(e) => setPassMarks(parseFloat(e.target.value) || 0)}
                  className="mt-1 font-mono"
                />
              </div>
            </div>

            <div>
              <label className="text-xs text-slate-300 font-medium">Instructions for Candidates</label>
              <Textarea
                rows={3}
                value={instructions}
                onChange={(e) => setInstructions(e.target.value)}
                className="mt-1 text-xs"
              />
            </div>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setCreateDialogOpen(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={createMutation.isPending} className="bg-cyan-600 hover:bg-cyan-500">
                {createMutation.isPending ? "Creating..." : "Save Exam"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}
