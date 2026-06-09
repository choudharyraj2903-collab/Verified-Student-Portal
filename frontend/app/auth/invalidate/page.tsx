"use client";

import { useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import Link from "next/link";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { ShieldOff, Loader2 } from "lucide-react";
import { api } from "@/lib/api";

export default function InvalidatePage() {
  const searchParams = useSearchParams();
  const [done, setDone] = useState(false);

  useEffect(() => {
    const token = searchParams.get("token");
    const call = token
      ? api.get(`/auth/invalidate?token=${encodeURIComponent(token)}`)
      : Promise.resolve();
    call.catch(() => {}).finally(() => setDone(true));
  }, [searchParams]);

  return (
    <main className="flex items-center justify-center min-h-screen bg-[#0a0f1e] px-4">
      <Card className="w-full max-w-md bg-[#1a2235] border-[#1e2d45]">
        <CardHeader className="text-center space-y-3">
          <div className="flex justify-center">
            {done ? (
              <ShieldOff className="h-14 w-14 text-red-400" />
            ) : (
              <Loader2 className="h-14 w-14 animate-spin text-amber-500" />
            )}
          </div>
          <CardTitle className="text-[#f1f5f9]">Session Terminated</CardTitle>
          <CardDescription className="text-[#94a3b8]">
            The suspicious session has been ended. If this keeps happening, your email
            account may be compromised. Contact the portal administrator.
          </CardDescription>
        </CardHeader>
        <CardContent className="text-center">
          <Link href="/auth/login" className="inline-flex items-center justify-center rounded-lg bg-amber-500 hover:bg-amber-600 text-[#0a0f1e] font-medium px-4 h-8 text-sm transition-colors">
            Sign in again
          </Link>
        </CardContent>
      </Card>
    </main>
  );
}
