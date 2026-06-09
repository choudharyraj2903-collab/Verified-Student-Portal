"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { AuthProvider, useAuth } from "@/lib/auth-context";
import { Sidebar } from "@/components/shared/sidebar";
import { LoadingSpinner } from "@/components/shared/ui-helpers";

function AuthenticatedShell({ children }: { children: React.ReactNode }) {
  const { role, profile, loading, isAuthenticated } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (!loading && !isAuthenticated) {
      router.replace("/auth/login");
    }
  }, [loading, isAuthenticated, router]);

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <LoadingSpinner text="Loading portal..." />
      </div>
    );
  }

  if (!isAuthenticated || !role) return null;

  const fullName = profile?.profile.full_name;
  const councilCode = profile ? undefined : undefined; // admin role badge
  const roleBadge = role === "SUPER_ADMIN" ? "Super Admin" : role === "COUNCIL_ADMIN" ? "Council Admin" : undefined;

  return (
    <div className="flex min-h-screen bg-[#0a0f1e]">
      <Sidebar role={role} fullName={fullName} roleBadge={roleBadge} />
      <main className="flex-1 overflow-auto">
        {children}
      </main>
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
