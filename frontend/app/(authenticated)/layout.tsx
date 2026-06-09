"use client";

import { useEffect } from "react";
import { useRouter, usePathname } from "next/navigation";
import { AuthProvider, useAuth } from "@/lib/auth-context";
import { Sidebar } from "@/components/shared/sidebar";
import { LoadingSpinner } from "@/components/shared/ui-helpers";
import { Badge } from "@/components/ui/badge";

const PAGE_TITLES: Record<string, string> = {
  "/dashboard": "Dashboard",
  "/verification": "My Requests",
  "/verification/new": "Submit Request",
  "/verification/card": "Verified Card",
  "/profile/create": "Create Profile",
  "/profile/edit": "Edit Profile",
  "/council": "Council Requests",
  "/admin/students": "Students",
  "/admin/council-admins": "Council Admins",
  "/admin/requests": "All Requests",
  "/admin/audit-logs": "Audit Logs",
};

function TopBar() {
  const { role, profile } = useAuth();
  const pathname = usePathname();

  const title = Object.entries(PAGE_TITLES).find(([path]) =>
    path === "/" ? pathname === "/" : pathname.startsWith(path)
  )?.[1] ?? "Portal";

  const displayName = profile?.profile.full_name ?? (role ? role.replace("_", " ") : "");
  const roleLabel = role === "SUPER_ADMIN" ? "Super Admin" : role === "COUNCIL_ADMIN" ? "Council Admin" : "Student";
  const roleCls = role === "SUPER_ADMIN" ? "bg-red-500/20 text-red-400 border-red-500/30" :
    role === "COUNCIL_ADMIN" ? "bg-blue-500/20 text-blue-400 border-blue-500/30" :
    "bg-amber-500/20 text-amber-400 border-amber-500/30";

  return (
    <header className="h-12 border-b border-[#1e2d45] bg-[#111827]/50 px-6 flex items-center justify-between shrink-0">
      <h2 className="text-sm font-semibold text-[#f1f5f9]">{title}</h2>
      <div className="flex items-center gap-2">
        {displayName && <span className="text-sm text-[#94a3b8]">{displayName}</span>}
        <Badge variant="outline" className={`text-xs ${roleCls}`}>{roleLabel}</Badge>
      </div>
    </header>
  );
}

function AuthenticatedShell({ children }: { children: React.ReactNode }) {
  const { role, profile, loading, isAuthenticated } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (!loading && !isAuthenticated) router.replace("/auth/login");
  }, [loading, isAuthenticated, router]);

  if (loading) return (
    <div className="flex items-center justify-center min-h-screen">
      <LoadingSpinner text="Loading portal..." />
    </div>
  );

  if (!isAuthenticated || !role) return null;

  const roleBadge = role === "SUPER_ADMIN" ? "Super Admin" : role === "COUNCIL_ADMIN" ? "Council Admin" : undefined;

  return (
    <div className="flex min-h-screen bg-[#0a0f1e]">
      <Sidebar role={role} fullName={profile?.profile.full_name} roleBadge={roleBadge} />
      <div className="flex-1 flex flex-col overflow-hidden">
        <TopBar />
        <main className="flex-1 overflow-auto">{children}</main>
      </div>
    </div>
  );
}

export default function AuthenticatedLayout({ children }: { children: React.ReactNode }) {
  return (
    <AuthProvider>
      <AuthenticatedShell>{children}</AuthenticatedShell>
    </AuthProvider>
  );
}
