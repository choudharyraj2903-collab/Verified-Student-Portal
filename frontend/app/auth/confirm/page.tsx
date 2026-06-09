"use client";

import { useEffect } from "react";
import { useSearchParams } from "next/navigation";
import Link from "next/link";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { CheckCircle } from "lucide-react";
import { api } from "@/lib/api";

export default function ConfirmPage() {
  const searchParams = useSearchParams();

  useEffect(() => {
    const token = searchParams.get("token");
    if (token) {
      api.get(`/auth/confirm?token=${encodeURIComponent(token)}`).catch(() => {});
    }
  }, [searchParams]);

  return (
    <main className="flex items-center justify-center min-h-screen bg-[#0a0f1e] px-4">
      <Card className="w-full max-w-md bg-[#1a2235] border-[#1e2d45]">
        <CardHeader className="text-center space-y-3">
          <div className="flex justify-center">
            <CheckCircle className="h-14 w-14 text-emerald-400" />
          </div>
          <CardTitle className="text-[#f1f5f9]">Login Confirmed</CardTitle>
          <CardDescription className="text-[#94a3b8]">
            This session has been verified as yours. You are safe.
          </CardDescription>
        </CardHeader>
        <CardContent className="text-center">
        <Link href="/dashboard" className="inline-flex items-center justify-center rounded-lg bg-amber-500 hover:bg-amber-600 text-[#0a0f1e] font-medium px-4 h-8 text-sm transition-colors">
            Go to Dashboard
          </Link>
        </CardContent>
      </Card>
    </main>
  );
}
