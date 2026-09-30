import React, { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Database, Plus, Search, Eye, AlertCircle } from "lucide-react";
import { api, CreateQuestionBankInput, QuestionBank } from "../../lib/api";
import { Card, CardContent } from "../../components/ui/card";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Textarea } from "../../components/ui/textarea";
import { Table, TableHeader, TableRow, TableHead, TableBody, TableCell } from "../../components/ui/table";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "../../components/ui/dialog";
import { Skeleton } from "../../components/ui/skeleton";
import { useAuth } from "../../hooks/useAuth";
import { formatDate } from "../../lib/utils";

export function QuestionBanksPage() {
  const { role } = useAuth();
  const queryClient = useQueryClient();
  const [search, setSearch] = useState("");
  const [createDialogOpen, setCreateDialogOpen] = useState(false);

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [formError, setFormError] = useState<string | null>(null);

  const { data: banksData, isLoading } = useQuery({
    queryKey: ["question-banks"],
    queryFn: () => api.listQuestionBanks(),
    enabled: role === "exam_admin",
  });

  const createMutation = useMutation({
    mutationFn: (input: CreateQuestionBankInput) => api.createQuestionBank(input),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["question-banks"] });
      setCreateDialogOpen(false);
      setName("");
      setDescription("");
      setFormError(null);
    },
    onError: (err: any) => {
      console.error("[QuestionBanksPage] Create error:", err);
      setFormError(err.message || "Failed to create question bank");
    },
  });

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault();
    setFormError(null);
    if (!name.trim()) {
      setFormError("Question bank name is required.");
      return;
    }
    createMutation.mutate({ name: name.trim(), description: description.trim() });
  };

  const banks = (banksData?.items || []).filter(Boolean);
  const searchLower = (search || '').toLowerCase();
  const filteredBanks = banks.filter(
    (b: QuestionBank) =>
      String(b.name ?? '').toLowerCase().includes(searchLower) ||
      String(b.description ?? '').toLowerCase().includes(searchLower)
  );

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-slate-800 pb-5">
        <div>
          <h1 className="text-2xl font-bold text-white flex items-center gap-2">
            <Database className="h-6 w-6 text-purple-400" />
            Question Banks
          </h1>
          <p className="text-slate-400 text-sm mt-1">
            Author and publish versioned questions across multiple domains.
          </p>
        </div>
        <Button onClick={() => { setFormError(null); setCreateDialogOpen(true); }} className="bg-purple-600 hover:bg-purple-500 text-white glow-cyan">
          <Plus className="h-4 w-4 mr-2" />
          Create Question Bank
        </Button>
      </div>

      <div className="flex items-center gap-4">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-3 top-2.5 h-4 w-4 text-slate-500" />
          <Input
            placeholder="Search by bank name or description..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9 bg-slate-900 border-slate-800"
          />
        </div>
      </div>

      <Card className="border-slate-800 bg-slate-900/60">
        <CardContent className="p-0">
          {isLoading ? (
            <div className="p-6 space-y-3">
              <Skeleton className="h-12 w-full" />
              <Skeleton className="h-12 w-full" />
            </div>
          ) : filteredBanks.length === 0 ? (
            <div className="p-12 text-center text-slate-400">
              <p className="text-base font-medium">No question banks found.</p>
              <p className="text-xs text-slate-500 mt-1">Create a question bank to start authoring questions.</p>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Bank Name & ID</TableHead>
                  <TableHead>Description</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Created At</TableHead>
                  <TableHead className="text-right">Action</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredBanks.map((bank: QuestionBank, idx: number) => {
                  const uniqueKey = bank.id ? `qb-${bank.id}-${idx}` : `qb-fallback-${idx}`;
                  return (
                    <TableRow key={uniqueKey}>
                      <TableCell>
                        {bank.id && bank.id !== 'undefined' ? (
                          <Link to={`/admin/question-banks/${bank.id}`} className="font-semibold text-white hover:text-purple-400 transition-colors">
                            {bank.name}
                          </Link>
                        ) : (
                          <span className="font-semibold text-white">{bank.name}</span>
                        )}
                        <div className="text-xs text-slate-400 font-mono font-normal">
                          ID: {bank.id || 'N/A'}
                        </div>
                      </TableCell>
                      <TableCell className="text-slate-300 text-xs max-w-md truncate">
                        {bank.description || "—"}
                      </TableCell>
                      <TableCell className="text-slate-300 text-xs font-mono">
                        {bank.status || "ACTIVE"}
                      </TableCell>
                      <TableCell className="text-slate-400 text-xs">
                        {formatDate(bank.created_at)}
                      </TableCell>
                      <TableCell className="text-right">
                        {bank.id && bank.id !== 'undefined' ? (
                          <Link to={`/admin/question-banks/${bank.id}`}>
                            <Button variant="ghost" size="sm" className="text-xs">
                              <Eye className="h-3.5 w-3.5 mr-1" /> View Questions
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

      <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
        <DialogContent onClose={() => setCreateDialogOpen(false)}>
          <DialogHeader>
            <DialogTitle>Create Question Bank</DialogTitle>
            <DialogDescription>
              Organize related assessment questions into a centralized repository.
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
              <label className="text-xs text-slate-300 font-medium">Bank Name</label>
              <Input
                required
                placeholder="e.g. Cybersecurity & Network Protocols"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="mt-1"
              />
            </div>

            <div>
              <label className="text-xs text-slate-300 font-medium">Description</label>
              <Textarea
                placeholder="Overview of topics covered in this bank..."
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                className="mt-1 text-xs"
              />
            </div>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setCreateDialogOpen(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={createMutation.isPending} className="bg-purple-600 hover:bg-purple-500">
                {createMutation.isPending ? "Creating..." : "Save Question Bank"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}
