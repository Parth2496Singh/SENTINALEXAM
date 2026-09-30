import React, { useState } from "react";
import { useParams, Link } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { ArrowLeft, Plus, AlertCircle } from "lucide-react";
import { api, CreateQuestionWithVersionInput, QuestionType, Question } from "../../lib/api";
import { Card, CardContent } from "../../components/ui/card";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Textarea } from "../../components/ui/textarea";
import { Badge } from "../../components/ui/badge";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "../../components/ui/dialog";
import { Skeleton } from "../../components/ui/skeleton";

import { useAuth } from "../../hooks/useAuth";
import { formatDate } from "../../lib/utils";

export function QuestionBankDetailPage() {
  const { bankId } = useParams<{ bankId: string }>();
  const { role } = useAuth();
  const queryClient = useQueryClient();
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  // Question Form State
  const [type, setType] = useState<QuestionType>("SINGLE_CHOICE");
  const [stem, setStem] = useState("");
  const [marks, setMarks] = useState(10);
  const [negativeMarks, setNegativeMarks] = useState(0);

  // Single / Multiple Choice Options
  const [options, setOptions] = useState<Array<{ key: string; text: string }>>([
    { key: "A", text: "" },
    { key: "B", text: "" },
    { key: "C", text: "" },
    { key: "D", text: "" },
  ]);
  const [selectedSingleChoice, setSelectedSingleChoice] = useState("A");
  const [selectedMultiChoice, setSelectedMultiChoice] = useState<string[]>(["A"]);

  // True/False
  const [trueFalseAnswer, setTrueFalseAnswer] = useState(true);

  // Text
  const [textAnswer, setTextAnswer] = useState("");

  // Numeric
  const [numericValue, setNumericValue] = useState(0);
  const [numericTolerance, setNumericTolerance] = useState(0);

  const { data: bank, isLoading: isLoadingBank } = useQuery({
    queryKey: ["question-bank", bankId],
    queryFn: () => api.getQuestionBank(bankId!),
    enabled: role === "exam_admin" && !!bankId && bankId !== "undefined",
  });

  const createQuestionMutation = useMutation({
    mutationFn: async (input: CreateQuestionWithVersionInput) => {
      return await api.createQuestionWithVersion(bankId!, input);
    },
    onSuccess: async (data: any) => {
      console.log("[DIAGNOSTIC] onSuccess data:", data);
      console.log("[DIAGNOSTIC] data?.question:", data?.question);
      console.log("[DIAGNOSTIC] data?.question?.id:", data?.question?.id);
      console.log("[DIAGNOSTIC] data?.id:", data?.id);
      console.log("[DIAGNOSTIC] Object.keys(data):", Object.keys(data || {}));

      const questionId =
        data?.question?.id ||
        data?.question?.ID ||
        data?.question?.Id ||
        data?.id ||
        data?.ID ||
        data?.question_id ||
        (data?.data?.question?.id) ||
        (data?.data?.question?.ID) ||
        (data?.data?.id);

      console.log("[DIAGNOSTIC] Resolved questionId:", questionId);

      if (!questionId) {
        console.error("[DIAGNOSTIC] Resolution failed! Raw data:", JSON.stringify(data));
        setFormError("Question created, but failed to resolve Question ID for publishing.");
        return;
      }

      try {
        console.log("[DIAGNOSTIC] Calling publishQuestion with ID:", questionId);
        const pubRes = await api.publishQuestion(questionId);
        console.log("[DIAGNOSTIC] publishQuestion response:", pubRes);
      } catch (pubErr: any) {
        console.error("Auto-publish question failed:", pubErr);
        setFormError(`Question created, but publishing failed: ${pubErr?.message || 'Unknown error'}`);
        return;
      }

      queryClient.invalidateQueries({ queryKey: ["question-bank", bankId] });
      setCreateModalOpen(false);
      resetForm();
    },
    onError: (err: any) => {
      console.error("[QuestionBankDetailPage] Create Question Error:", err);
      setFormError(err.message || "Failed to create question");
    },
  });

  const resetForm = () => {
    setStem("");
    setType("SINGLE_CHOICE");
    setMarks(10);
    setNegativeMarks(0);
    setOptions([
      { key: "A", text: "" },
      { key: "B", text: "" },
      { key: "C", text: "" },
      { key: "D", text: "" },
    ]);
    setSelectedSingleChoice("A");
    setSelectedMultiChoice(["A"]);
    setTrueFalseAnswer(true);
    setTextAnswer("");
    setNumericValue(0);
    setNumericTolerance(0);
    setFormError(null);
  };

  const handleCreateSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setFormError(null);

    if (!stem.trim()) {
      setFormError("Question stem / prompt cannot be empty.");
      return;
    }

    let content: any = { stem: stem.trim() };
    let correctAnswer: any = {};

    if (type === "SINGLE_CHOICE") {
      const validOptions = options.filter((o) => o.text.trim() !== "");
      if (validOptions.length < 2) {
        setFormError("At least 2 choices must have text.");
        return;
      }
      content.options = validOptions;
      correctAnswer = { selected: selectedSingleChoice };
    } else if (type === "MULTIPLE_CHOICE") {
      const validOptions = options.filter((o) => o.text.trim() !== "");
      if (validOptions.length < 2) {
        setFormError("At least 2 choices must have text.");
        return;
      }
      if (selectedMultiChoice.length === 0) {
        setFormError("Please select at least one correct choice.");
        return;
      }
      content.options = validOptions;
      correctAnswer = { selected: selectedMultiChoice };
    } else if (type === "TRUE_FALSE") {
      correctAnswer = { value: trueFalseAnswer };
    } else if (type === "TEXT") {
      if (!textAnswer.trim()) {
        setFormError("Please provide the accepted text answer.");
        return;
      }
      correctAnswer = { text: textAnswer.trim(), case_sensitive: false };
    } else if (type === "NUMERIC") {
      correctAnswer = { value: numericValue, tolerance: numericTolerance };
    }

    createQuestionMutation.mutate({
      type,
      content,
      correct_answer: correctAnswer,
      marks: Number(marks) || 10,
      negative_marks: Number(negativeMarks) || 0,
    });
  };

  if (isLoadingBank) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-32 w-full" />
      </div>
    );
  }

  if (!bank) {
    return (
      <div className="text-center py-12 text-slate-400">
        <p>Question Bank not found.</p>
        <Link to="/admin/question-banks">
          <Button variant="outline" className="mt-4">Back to Banks</Button>
        </Link>
      </div>
    );
  }

  const questions = bank.questions || [];

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-slate-800 pb-5">
        <div>
          <Link to="/admin/question-banks" className="text-xs text-slate-400 hover:text-purple-400 flex items-center gap-1 mb-2">
            <ArrowLeft className="h-3.5 w-3.5" /> Back to Question Banks
          </Link>
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold text-white">{bank.name}</h1>
            <Badge variant="outline" className="font-mono text-purple-400 border-purple-500/30">
              {bank.status || "ACTIVE"}
            </Badge>
          </div>
          <div className="text-xs text-slate-400 font-mono mt-1 flex items-center gap-3">
            <span>Bank UUID: {bank.id}</span>
            {bank.created_at && (
              <span>Created: {formatDate(bank.created_at)}</span>
            )}
          </div>
        </div>

        <Button onClick={() => { setFormError(null); setCreateModalOpen(true); }} className="bg-purple-600 hover:bg-purple-500 text-white">
          <Plus className="h-4 w-4 mr-2" />
          Create New Question
        </Button>
      </div>

      <Card className="border-slate-800 bg-slate-900/60">
        <CardContent className="p-0">
          {questions.length === 0 ? (
            <div className="p-12 text-center text-slate-400">
              <p className="text-base font-medium">No questions in this bank yet.</p>
              <Button onClick={() => { setFormError(null); setCreateModalOpen(true); }} variant="outline" className="mt-3">
                Author First Question
              </Button>
            </div>
          ) : (
            <div className="divide-y divide-slate-800">
              {questions.map((q: Question, idx: number) => {
                const latestVersion = q.versions?.[q.versions.length - 1];
                const stemText = latestVersion?.content?.stem || "Question Version Prompt";
                const vType = latestVersion?.type || q.versions?.[0]?.type || "SINGLE_CHOICE";

                return (
                  <div key={q.id || `q-${idx}`} className="p-5 hover:bg-slate-800/30 transition-colors space-y-2">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <Badge
                          variant={
                            vType === "SINGLE_CHOICE"
                              ? "default"
                              : vType === "MULTIPLE_CHOICE"
                              ? "purple"
                              : vType === "TRUE_FALSE"
                              ? "warning"
                              : "secondary"
                          }
                        >
                          {vType}
                        </Badge>
                        <span className="text-xs font-mono text-slate-500">Question ID: {q.id}</span>
                      </div>
                      <div className="flex items-center gap-2">
                        {latestVersion && (
                          <Badge variant={latestVersion.status === "ACTIVE" ? "success" : "outline"}>
                            Version {latestVersion.version} ({latestVersion.status})
                          </Badge>
                        )}
                      </div>
                    </div>

                    <div className="text-sm font-medium text-slate-100 font-sans">{stemText}</div>

                    {latestVersion && (
                      <div className="pt-2 text-xs text-slate-400 flex items-center justify-between font-mono bg-slate-950/60 p-2.5 rounded border border-slate-800/80">
                        <div>
                          Version UUID: <span className="text-cyan-400 font-semibold">{latestVersion.id}</span>
                        </div>
                        <div className="text-slate-400">
                          Marks: <span className="text-white font-semibold">{latestVersion.marks}</span>
                        </div>
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Author Question Modal */}
      <Dialog open={createModalOpen} onOpenChange={setCreateModalOpen}>
        <DialogContent className="max-w-2xl" onClose={() => setCreateModalOpen(false)}>
          <DialogHeader>
            <DialogTitle>Author Assessment Question</DialogTitle>
            <DialogDescription>
              Create a versioned question with exact scoring criteria in a single unified API call.
            </DialogDescription>
          </DialogHeader>

          {formError && (
            <div className="p-3 rounded-lg bg-red-500/10 border border-red-500/30 text-red-400 text-xs flex items-center gap-2">
              <AlertCircle className="h-4 w-4 shrink-0" />
              <span>{formError}</span>
            </div>
          )}

          <form onSubmit={handleCreateSubmit} className="space-y-4">
            <div className="grid grid-cols-3 gap-3">
              <div>
                <label className="text-xs text-slate-300 font-medium">Question Type</label>
                <select
                  value={type}
                  onChange={(e) => setType(e.target.value as QuestionType)}
                  className="w-full mt-1 rounded-md border border-slate-700 bg-slate-950 p-2 text-xs text-slate-100"
                >
                  <option value="SINGLE_CHOICE">Single Choice (Radio)</option>
                  <option value="MULTIPLE_CHOICE">Multiple Choice (Checkbox)</option>
                  <option value="TRUE_FALSE">True / False</option>
                  <option value="TEXT">Short Text Answer</option>
                  <option value="NUMERIC">Numeric Answer</option>
                </select>
              </div>

              <div>
                <label className="text-xs text-slate-300 font-medium">Marks</label>
                <Input
                  type="number"
                  required
                  min={1}
                  value={marks}
                  onChange={(e) => setMarks(parseFloat(e.target.value) || 10)}
                  className="mt-1 text-xs font-mono"
                />
              </div>

              <div>
                <label className="text-xs text-slate-300 font-medium">Negative Marks</label>
                <Input
                  type="number"
                  min={0}
                  value={negativeMarks}
                  onChange={(e) => setNegativeMarks(parseFloat(e.target.value) || 0)}
                  className="mt-1 text-xs font-mono"
                />
              </div>
            </div>

            <div>
              <label className="text-xs text-slate-300 font-medium">Question Stem / Prompt</label>
              <Textarea
                required
                rows={3}
                placeholder="Enter the question text or prompt here..."
                value={stem}
                onChange={(e) => setStem(e.target.value)}
                className="mt-1 text-xs"
              />
            </div>

            {/* Dynamic Options for Choice Questions */}
            {(type === "SINGLE_CHOICE" || type === "MULTIPLE_CHOICE") && (
              <div className="space-y-2 pt-2 border-t border-slate-800">
                <label className="text-xs text-slate-300 font-semibold block">Choice Options</label>
                {options.map((opt, i) => (
                  <div key={opt.key} className="flex items-center gap-2">
                    <span className="w-6 text-center text-xs font-mono font-bold text-cyan-400">{opt.key}.</span>
                    <Input
                      placeholder={`Option ${opt.key} text`}
                      value={opt.text}
                      onChange={(e) => {
                        const newOpts = [...options];
                        newOpts[i].text = e.target.value;
                        setOptions(newOpts);
                      }}
                      className="text-xs flex-1"
                    />
                    {type === "SINGLE_CHOICE" && (
                      <input
                        type="radio"
                        name="singleCorrect"
                        checked={selectedSingleChoice === opt.key}
                        onChange={() => setSelectedSingleChoice(opt.key)}
                        className="accent-cyan-500 h-4 w-4"
                      />
                    )}
                    {type === "MULTIPLE_CHOICE" && (
                      <input
                        type="checkbox"
                        checked={selectedMultiChoice.includes(opt.key)}
                        onChange={(e) => {
                          if (e.target.checked) {
                            setSelectedMultiChoice([...selectedMultiChoice, opt.key]);
                          } else {
                            setSelectedMultiChoice(selectedMultiChoice.filter((k) => k !== opt.key));
                          }
                        }}
                        className="accent-purple-500 h-4 w-4"
                      />
                    )}
                  </div>
                ))}
              </div>
            )}

            {/* Dynamic Input for True/False */}
            {type === "TRUE_FALSE" && (
              <div className="pt-2 border-t border-slate-800 space-y-2">
                <label className="text-xs text-slate-300 font-semibold block">Correct Answer</label>
                <div className="flex gap-4">
                  <label className="flex items-center gap-2 text-xs text-slate-200 cursor-pointer">
                    <input
                      type="radio"
                      name="tf"
                      checked={trueFalseAnswer === true}
                      onChange={() => setTrueFalseAnswer(true)}
                      className="accent-cyan-500"
                    />
                    True
                  </label>
                  <label className="flex items-center gap-2 text-xs text-slate-200 cursor-pointer">
                    <input
                      type="radio"
                      name="tf"
                      checked={trueFalseAnswer === false}
                      onChange={() => setTrueFalseAnswer(false)}
                      className="accent-cyan-500"
                    />
                    False
                  </label>
                </div>
              </div>
            )}

            {/* Dynamic Input for Text */}
            {type === "TEXT" && (
              <div className="pt-2 border-t border-slate-800">
                <label className="text-xs text-slate-300 font-semibold block">Accepted Text Answer</label>
                <Input
                  required
                  placeholder="e.g. TCP/IP"
                  value={textAnswer}
                  onChange={(e) => setTextAnswer(e.target.value)}
                  className="mt-1 text-xs font-mono"
                />
              </div>
            )}

            {/* Dynamic Input for Numeric */}
            {type === "NUMERIC" && (
              <div className="pt-2 border-t border-slate-800 grid grid-cols-2 gap-3">
                <div>
                  <label className="text-xs text-slate-300 font-semibold block">Target Numeric Value</label>
                  <Input
                    type="number"
                    required
                    value={numericValue}
                    onChange={(e) => setNumericValue(parseFloat(e.target.value) || 0)}
                    className="mt-1 text-xs font-mono"
                  />
                </div>
                <div>
                  <label className="text-xs text-slate-300 font-semibold block">Allowed Tolerance (±)</label>
                  <Input
                    type="number"
                    required
                    value={numericTolerance}
                    onChange={(e) => setNumericTolerance(parseFloat(e.target.value) || 0)}
                    className="mt-1 text-xs font-mono"
                  />
                </div>
              </div>
            )}

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setCreateModalOpen(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={createQuestionMutation.isPending} className="bg-purple-600 hover:bg-purple-500">
                {createQuestionMutation.isPending ? "Publishing..." : "Create & Publish Question"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}
