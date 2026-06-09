"use client";

import { useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Loader2, XCircle } from "lucide-react";

const API_URL = process.env.NEXT_PUBLIC_API_URL!;

export default function VerifyPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [status, setStatus] = useState<"loading" | "success" | "error">("loading");

  useEffect(() => {
    const token = searchParams.get("token");
    if (!token) { setStatus("error"); return; }

    // Remove token from address bar immediately
    router.replace("/auth/verify");

    // Hit backend verify — it sets HttpOnly cookies then redirects to /dashboard
    // We can't use fetch (cookies won't be set on cross-origin redirect in Next.js)
    // Instead redirect the browser directly so it follows the redirect and receives cookies
    window.location.href = `${API_URL}/auth/verify?token=${encodeURIComponent(token)}`;
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <main className="flex items-center justify-center min-h-screen bg-[#0a0f1e] px-4">
      <Card className="w-full max-w-md bg-[#1a2235] border-[#1e2d45]">
        <CardHeader className="text-center">
          <CardTitle className="text-[#f1f5f9]">
            {status === "loading" ? "Verifying your login link..." : "Verification Failed"}
          </CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col items-center gap-4 pb-6">
          {status === "loading" && <Loader2 className="h-10 w-10 animate-spin text-amber-500" />}
          {status === "error" && (
            <>
              <XCircle className="h-10 w-10 text-red-400" />
              <p className="text-sm text-[#94a3b8] text-center">This link is invalid or has expired.</p>
              <Button onClick={() => router.push("/auth/login")}
                className="bg-amber-500 hover:bg-amber-600 text-[#0a0f1e]">
                Back to Login
              </Button>
            </>
          )}
        </CardContent>
      </Card>
    </main>
  );
}
