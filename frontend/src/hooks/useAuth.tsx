import React, { createContext, useContext, useState, useEffect } from "react";
import { setAuthToken, getStoredAuthToken, clearAuthToken } from "../lib/api";
import { queryClient } from "../app/providers";

export type UserRole = "exam_admin" | "candidate" | "system_auditor";

export interface AuthUser {
  id: string; // sub or candidate id
  role: UserRole;
  name: string;
  email: string;
}

interface AuthContextType {
  token: string | null;
  role: UserRole | null;
  user: AuthUser | null;
  loginAsDevAdmin: () => void;
  loginAsDevCandidate: (candidateSubId?: string) => void;
  setCustomToken: (rawToken: string, role: UserRole) => void;
  logout: () => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

// Known dev IDs from backend
const DEV_ADMIN_SUB = "00000000-0000-0000-0000-000000000001";
const DEV_CANDIDATE_SUB = "00000000-0000-0000-0000-000000000002";

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [token, setTokenState] = useState<string | null>(getStoredAuthToken());
  const [role, setRoleState] = useState<UserRole | null>(null);
  const [user, setUserState] = useState<AuthUser | null>(null);

  useEffect(() => {
    const existing = getStoredAuthToken();
    const isLoggedOut = typeof localStorage !== 'undefined' && localStorage.getItem('sentinelexam_logged_out') === 'true';
    if (existing) {
      parseAndSetToken(existing);
    } else if (!isLoggedOut) {
      loginAsDevAdmin();
    }
  }, []);

  const parseAndSetToken = (rawToken: string) => {
    if (typeof localStorage !== 'undefined') {
      localStorage.removeItem('sentinelexam_logged_out');
    }
    setAuthToken(rawToken);
    setTokenState(rawToken);

    if (rawToken.startsWith("dev:")) {
      const parts = rawToken.split(":");
      const roleStr = parts[1] || "exam_admin";
      const subId = parts[2] || DEV_ADMIN_SUB;
      const parsedRole: UserRole = roleStr.includes("candidate") ? "candidate" : "exam_admin";

      setRoleState(parsedRole);
      setUserState({
        id: subId,
        role: parsedRole,
        name: parsedRole === "exam_admin" ? "Dev Administrator" : "Dev Candidate",
        email: parsedRole === "exam_admin" ? "admin@sentinelexam.internal" : "candidate@sentinelexam.internal",
      });
    } else {
      // Standard JWT decoding fallback
      try {
        const payload = JSON.parse(atob(rawToken.split(".")[1]));
        const roles: string[] = payload.realm_access?.roles || payload.roles || [];
        const isCandidate = roles.includes("candidate");
        const parsedRole: UserRole = isCandidate ? "candidate" : "exam_admin";

        setRoleState(parsedRole);
        setUserState({
          id: payload.sub || "user-id",
          role: parsedRole,
          name: payload.name || payload.preferred_username || "Authenticated User",
          email: payload.email || "user@sentinelexam.com",
        });
      } catch {
        // Simple fallback if parsing fails
        setRoleState("exam_admin");
        setUserState({
          id: "unknown",
          role: "exam_admin",
          name: "Authenticated User",
          email: "user@sentinelexam.com",
        });
      }
    }
  };

  const loginAsDevAdmin = () => {
    const devToken = `dev:exam_admin:${DEV_ADMIN_SUB}`;
    parseAndSetToken(devToken);
  };

  const loginAsDevCandidate = (candidateSubId = DEV_CANDIDATE_SUB) => {
    const devToken = `dev:candidate:${candidateSubId}`;
    parseAndSetToken(devToken);
  };

  const setCustomToken = (rawToken: string, userRole: UserRole) => {
    if (typeof localStorage !== 'undefined') {
      localStorage.removeItem('sentinelexam_logged_out');
    }
    setAuthToken(rawToken);
    setTokenState(rawToken);
    setRoleState(userRole);
    setUserState({
      id: "custom-user",
      role: userRole,
      name: userRole === "exam_admin" ? "Admin User" : "Candidate User",
      email: `${userRole}@sentinelexam.com`,
    });
  };

  const logout = () => {
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem('sentinelexam_logged_out', 'true');
    }
    clearAuthToken();
    try {
      queryClient.clear();
    } catch {
      // ignore
    }
    setTokenState(null);
    setRoleState(null);
    setUserState(null);
  };

  return (
    <AuthContext.Provider
      value={{
        token,
        role,
        user,
        loginAsDevAdmin,
        loginAsDevCandidate,
        setCustomToken,
        logout,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
}
