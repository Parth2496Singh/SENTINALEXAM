import React, { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Users, Plus, Search, AlertCircle } from "lucide-react";
import { api, CreateCandidateInput, Candidate } from "../../lib/api";
import { Card, CardContent } from "../../components/ui/card";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Table, TableHeader, TableRow, TableHead, TableBody, TableCell } from "../../components/ui/table";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from "../../components/ui/dialog";
import { Skeleton } from "../../components/ui/skeleton";
import { useAuth } from "../../hooks/useAuth";
import { formatDate } from "../../lib/utils";

export function CandidatesPage() {
  const { role } = useAuth();
  const queryClient = useQueryClient();
  const [search, setSearch] = useState("");
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);

  const [keycloakSub, setKeycloakSub] = useState("");
  const [email, setEmail] = useState("");
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");

  const { data: candidatesData, isLoading } = useQuery({
    queryKey: ["candidates"],
    queryFn: () => api.listCandidates(),
    enabled: role === "exam_admin",
  });

  const createMutation = useMutation({
    mutationFn: (input: CreateCandidateInput) => api.createCandidate(input),
    onSuccess: (data: any) => {
      console.log("[CandidatesPage] Candidate registered successfully:", data);
      queryClient.invalidateQueries({ queryKey: ["candidates"] });
      setCreateDialogOpen(false);
      setKeycloakSub("");
      setEmail("");
      setFirstName("");
      setLastName("");
      setFormError(null);
    },
    onError: (err: any) => {
      console.error("[CandidatesPage] Create candidate error:", err);
      setFormError(err.message || "Failed to register candidate. Ensure valid email and UUID inputs.");
    },
  });

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault();
    setFormError(null);

    const fullName = `${firstName.trim()} ${lastName.trim()}`.trim();
    if (!fullName) {
      setFormError("Candidate first name and last name are required.");
      return;
    }

    if (keycloakSub.trim() && keycloakSub.trim().length !== 36) {
      setFormError("Keycloak Sub must be a valid 36-character UUID string or left blank.");
      return;
    }

    createMutation.mutate({
      keycloak_sub: keycloakSub.trim() || undefined,
      email: email.trim() || undefined,
      full_name: fullName,
      first_name: firstName.trim(),
      last_name: lastName.trim(),
    });
  };

  const candidates = (candidatesData?.items || []).filter(Boolean);
  const q = (search || "").toLowerCase();
  const filteredCandidates = candidates.filter(
    (c: Candidate) =>
      !q ||
      String(c.email ?? "").toLowerCase().includes(q) ||
      String(c.full_name ?? "").toLowerCase().includes(q) ||
      String(c.first_name ?? "").toLowerCase().includes(q) ||
      String(c.last_name ?? "").toLowerCase().includes(q)
  );

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-slate-800 pb-5">
        <div>
          <h1 className="text-2xl font-bold text-white flex items-center gap-2">
            <Users className="h-6 w-6 text-indigo-400" />
            Candidate Directory
          </h1>
          <p className="text-slate-400 text-sm mt-1">
            Registered test-takers and Keycloak identity bindings.
          </p>
        </div>
        <Button onClick={() => { setFormError(null); setCreateDialogOpen(true); }} className="bg-indigo-600 hover:bg-indigo-500 text-white">
          <Plus className="h-4 w-4 mr-2" />
          Register Candidate
        </Button>
      </div>

      <div className="flex items-center gap-4">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-3 top-2.5 h-4 w-4 text-slate-500" />
          <Input
            placeholder="Search by candidate name or email..."
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
          ) : filteredCandidates.length === 0 ? (
            <div className="p-12 text-center text-slate-400">
              <p className="text-base font-medium">No candidates registered.</p>
              <Button onClick={() => { setFormError(null); setCreateDialogOpen(true); }} variant="outline" className="mt-3">
                Register First Candidate
              </Button>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Candidate Name</TableHead>
                  <TableHead>Email</TableHead>
                  <TableHead>Candidate UUID</TableHead>
                  <TableHead>Registered</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredCandidates.map((c: Candidate | any, idx: number) => {
                  const candidateId = c.id || c.ID || c.candidate_id || `cand-${idx}`;
                  const fullName = c.full_name || c.FullName || `${c.first_name || c.FirstName || ''} ${c.last_name || c.LastName || ''}`.trim() || 'Candidate';
                  const candidateEmail = c.email || c.Email || "—";
                  const createdAt = c.created_at || c.CreatedAt;

                  return (
                    <TableRow key={candidateId}>
                      <TableCell className="font-semibold text-white">
                        {fullName}
                      </TableCell>
                      <TableCell className="text-slate-300 text-xs font-mono">
                        {candidateEmail}
                      </TableCell>
                      <TableCell className="font-mono text-xs text-cyan-400">
                        {candidateId}
                      </TableCell>
                      <TableCell className="text-slate-400 text-xs">
                        {formatDate(createdAt)}
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
            <DialogTitle>Register Candidate</DialogTitle>
            <DialogDescription>
              Add candidate user profile to allow assessment assignments.
            </DialogDescription>
          </DialogHeader>

          {formError && (
            <div className="p-3 rounded-lg bg-red-500/10 border border-red-500/30 text-red-400 text-xs flex items-center gap-2">
              <AlertCircle className="h-4 w-4 shrink-0" />
              <span>{formError}</span>
            </div>
          )}

          <form onSubmit={handleCreate} className="space-y-4">
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="text-xs text-slate-300 font-medium">First Name</label>
                <Input
                  required
                  placeholder="e.g. Alice"
                  value={firstName}
                  onChange={(e) => setFirstName(e.target.value)}
                  className="mt-1"
                />
              </div>
              <div>
                <label className="text-xs text-slate-300 font-medium">Last Name</label>
                <Input
                  required
                  placeholder="e.g. Smith"
                  value={lastName}
                  onChange={(e) => setLastName(e.target.value)}
                  className="mt-1"
                />
              </div>
            </div>

            <div>
              <label className="text-xs text-slate-300 font-medium">Email Address (Optional)</label>
              <Input
                type="email"
                placeholder="candidate@example.com"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="mt-1"
              />
            </div>

            <div>
              <label className="text-xs text-slate-300 font-medium">Keycloak Sub / Dev UUID (Optional)</label>
              <Input
                placeholder="e.g. 00000000-0000-0000-0000-000000000002"
                value={keycloakSub}
                onChange={(e) => setKeycloakSub(e.target.value)}
                className="mt-1 font-mono text-xs"
              />
              <span className="text-[10px] text-slate-500 mt-1 block">
                Leave blank to generate an automatic dev sub identifier.
              </span>
            </div>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setCreateDialogOpen(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={createMutation.isPending} className="bg-indigo-600 hover:bg-indigo-500">
                {createMutation.isPending ? "Registering..." : "Register Candidate"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}
