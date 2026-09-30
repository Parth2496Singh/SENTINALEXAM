import React, { useState } from "react";
import { useNavigate } from "react-router-dom";
import { ShieldAlert, UserCheck, ShieldCheck, Key, ArrowRight } from "lucide-react";
import { useAuth } from "../hooks/useAuth";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "../components/ui/card";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";

export function LoginPage() {
  const { loginAsDevAdmin, loginAsDevCandidate, setCustomToken } = useAuth();
  const navigate = useNavigate();
  const [customToken, setCustomTokenInput] = useState("");

  const handleAdminLogin = () => {
    loginAsDevAdmin();
    navigate("/admin");
  };

  const handleCandidateLogin = () => {
    loginAsDevCandidate();
    navigate("/candidate");
  };

  const handleCustomSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!customToken.trim()) return;
    setCustomToken(customToken.trim(), customToken.includes("candidate") ? "candidate" : "exam_admin");
    navigate(customToken.includes("candidate") ? "/candidate" : "/admin");
  };

  return (
    <div className="min-h-screen bg-slate-950 flex flex-col items-center justify-center p-4 relative overflow-hidden">
      {/* Background decoration */}
      <div className="absolute -top-40 -left-40 w-96 h-96 bg-cyan-500/10 rounded-full blur-3xl pointer-events-none" />
      <div className="absolute -bottom-40 -right-40 w-96 h-96 bg-indigo-500/10 rounded-full blur-3xl pointer-events-none" />

      <div className="w-full max-w-md space-y-6 z-10">
        <div className="text-center space-y-2">
          <div className="inline-flex h-14 w-14 items-center justify-center rounded-2xl bg-cyan-500/10 border border-cyan-500/30 text-cyan-400 glow-cyan mb-2">
            <ShieldAlert className="h-8 w-8" />
          </div>
          <h1 className="text-3xl font-extrabold tracking-tight text-white">
            SENTINEL<span className="text-cyan-400 font-light">EXAM</span>
          </h1>
          <p className="text-slate-400 text-sm">
            Enterprise Digital Assessment & Infrastructure Platform
          </p>
        </div>

        <Card className="border-slate-800 bg-slate-900/90 shadow-2xl backdrop-blur-xl">
          <CardHeader className="text-center pb-4">
            <CardTitle className="text-lg text-slate-200">Select Access Persona</CardTitle>
            <CardDescription className="text-xs text-slate-400">
              Docker Dev Environment Bypass & Token Authentication
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <Button
              onClick={handleAdminLogin}
              className="w-full h-12 bg-cyan-600/90 hover:bg-cyan-500 text-white font-medium justify-between px-4 glow-cyan"
            >
              <div className="flex items-center gap-3">
                <ShieldCheck className="h-5 w-5" />
                <div className="text-left">
                  <div className="text-sm font-semibold">Login as Exam Admin</div>
                  <div className="text-[10px] text-cyan-100/80">Full management & configuration</div>
                </div>
              </div>
              <ArrowRight className="h-4 w-4" />
            </Button>

            <Button
              onClick={handleCandidateLogin}
              variant="outline"
              className="w-full h-12 border-emerald-500/30 bg-emerald-500/10 hover:bg-emerald-500/20 text-emerald-300 justify-between px-4"
            >
              <div className="flex items-center gap-3">
                <UserCheck className="h-5 w-5 text-emerald-400" />
                <div className="text-left">
                  <div className="text-sm font-semibold">Login as Candidate</div>
                  <div className="text-[10px] text-emerald-400/80">Take assigned assessments</div>
                </div>
              </div>
              <ArrowRight className="h-4 w-4 text-emerald-400" />
            </Button>

            <div className="relative my-4">
              <div className="absolute inset-0 flex items-center">
                <span className="w-full border-t border-slate-800" />
              </div>
              <div className="relative flex justify-center text-xs uppercase">
                <span className="bg-slate-900 px-2 text-slate-500 font-mono">Custom JWT / Token</span>
              </div>
            </div>

            <form onSubmit={handleCustomSubmit} className="space-y-3">
              <div className="relative">
                <Key className="absolute left-3 top-2.5 h-4 w-4 text-slate-500" />
                <Input
                  type="text"
                  placeholder="Bearer dev:exam_admin:... or JWT"
                  value={customToken}
                  onChange={(e) => setCustomTokenInput(e.target.value)}
                  className="pl-9 text-xs font-mono bg-slate-950 border-slate-800"
                />
              </div>
              <Button type="submit" variant="secondary" className="w-full text-xs" disabled={!customToken.trim()}>
                Authenticate Token
              </Button>
            </form>
          </CardContent>
        </Card>

        <div className="text-center text-xs text-slate-500 font-mono">
          Phase 3 Core Backend Connected & Verified
        </div>
      </div>
    </div>
  );
}
