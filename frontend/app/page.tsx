"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { api } from "@/lib/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Shield } from "lucide-react";

export default function LandingPage() {
  const router = useRouter();

  useEffect(() => {
    api.get("/profile").then((res) => {
      if (res.success) router.replace("/dashboard");
    }).catch(() => {});
  }, [router]);

  return (
    <main className="flex flex-col items-center justify-center min-h-screen bg-[#0a0f1e] px-4 text-center">
      <div className="max-w-2xl space-y-6">
        <div className="flex justify-center mb-2">
          <div className="rounded-full bg-amber-500/10 border border-amber-500/30 p-4">
            <Shield className="h-10 w-10 text-amber-500" />
          </div>
        </div>

        <h1 className="text-5xl font-bold text-[#f1f5f9] tracking-tight">
          Campus Council Portal
        </h1>
        <p className="text-lg text-[#94a3b8] font-medium">
          IIT Kanpur — Position of Responsibility Verification System
        </p>
        <p className="text-[#94a3b8] max-w-lg mx-auto">
          Submit, track, and verify council positions of responsibility. Students can
          get official verification badges; council admins review and approve requests.
        </p>

        <div className="pt-2">
          <Link href="/auth/login" className="inline-flex items-center justify-center rounded-lg bg-amber-500 hover:bg-amber-600 text-[#0a0f1e] font-semibold px-8 h-11 text-base transition-colors">
            Sign In with Institute Email
          </Link>
        </div>

        <div className="flex flex-wrap justify-center gap-2 pt-4">
          {["GNS", "ANC", "SNT", "MNC"].map((council) => (
            <Badge
              key={council}
              variant="outline"
              className="border-[#1e2d45] text-[#94a3b8] bg-[#111827]"
            >
              {council}
            </Badge>
          ))}
        </div>
      </div>
    </main>
  );
}
