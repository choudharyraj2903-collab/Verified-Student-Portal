"use client";

import { createContext, useContext, useEffect, useState, useCallback } from "react";
import { api } from "./api";

export type UserRole = "STUDENT" | "COUNCIL_ADMIN" | "SUPER_ADMIN";

export interface UserProfile {
  id: string; user_id: string; full_name: string; roll_number: string;
  year: number; branch: string; phone?: string; bio?: string; avatar_url?: string;
}

export interface ProfileResult {
  profile: UserProfile;
  approved_by_council: Record<string, number>;
  total_approved: number;
  is_complete: boolean;
}

interface CouncilAdmin { id: string; email: string; council_code: string; }

export interface AuthState {
  userId: string | null;
  email: string | null;
  role: UserRole | null;
  councilCodes: string[];
  profile: ProfileResult | null;
  loading: boolean;
}

interface AuthContextValue extends AuthState {
  refresh: () => Promise<void>;
  isAuthenticated: boolean;
}

const AuthContext = createContext<AuthContextValue>({
  userId: null, email: null, role: null, councilCodes: [], profile: null,
  loading: true, isAuthenticated: false, refresh: async () => {},
});

async function detectRole(): Promise<{ role: UserRole | null; profile: ProfileResult | null; councilCodes: string[]; }> {
  // Try student profile
  const profileRes = await api.get<ProfileResult>("/profile");
  if (profileRes.success && profileRes.data?.profile) {
    return { role: "STUDENT", profile: profileRes.data, councilCodes: [] };
  }

  // Try admin endpoints
  const adminRes = await api.get("/admin/students?page=1");
  if (adminRes.success) {
    // Can access admin — check if SUPER_ADMIN (can see audit logs)
    const auditRes = await api.get("/admin/audit-logs?page=1");
    if (auditRes.success) {
      return { role: "SUPER_ADMIN", profile: null, councilCodes: [] };
    }
    // COUNCIL_ADMIN — fetch their council scopes from council-admins list
    // We don't have a /me endpoint, so probe via councils available to this user
    // The council requests endpoint returns data only for their council
    // Try each known council to find which one they have access to
    const councilsRes = await api.get<Array<{ id: string; code: string }>>("/councils");
    const codes: string[] = [];
    if (councilsRes.success && councilsRes.data) {
      for (const c of councilsRes.data) {
        const r = await api.get(`/verification/council/${c.code}`);
        if (r.success) { codes.push(c.code); break; } // only need first match
      }
    }
    return { role: "COUNCIL_ADMIN", profile: null, councilCodes: codes };
  }

  return { role: null, profile: null, councilCodes: [] };
}

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [state, setState] = useState<AuthState>({
    userId: null, email: null, role: null, councilCodes: [], profile: null, loading: true,
  });

  const refresh = useCallback(async () => {
    setState((s) => ({ ...s, loading: true }));
    try {
      const { role, profile, councilCodes } = await detectRole();
      setState({ userId: profile?.profile.user_id ?? null, email: null, role, councilCodes, profile, loading: false });
    } catch {
      setState({ userId: null, email: null, role: null, councilCodes: [], profile: null, loading: false });
    }
  }, []);

  useEffect(() => { refresh(); }, [refresh]);

  return (
    <AuthContext.Provider value={{ ...state, isAuthenticated: state.role !== null, refresh }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() { return useContext(AuthContext); }
