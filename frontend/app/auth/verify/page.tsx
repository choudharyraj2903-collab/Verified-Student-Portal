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
    if (!token) {
      setStatus("error");
      return;
    }

    // Remove token from URL immediately
    window.history.replaceState({}, "", "/auth/verify");

    // The backend verify endpoint redirects to /dashboard on success
    // We call it directly via a redirect which sets cookies
    const verifyUrl = `${API_URL}/auth/verify?token=${encodeURIComponent(token)}`;
    
    // Use fetch to trigger the redirect; the browser follows it and sets cookies
    fetch(verifyUrl, { credentials: "include", redirect: "follow" })
      .then((res) => {
        if (res.ok || res.url.includes("/dashboard") || res.redirected) {
          setStatus("success");
          setTimeout(() => router.replace("/dashboard"), 800);
        } else {
          setStatus("error");
        }
      })
      .catch(() => setStatus("error"));
  }, [router, searchParams]);

  return (
    <main className="flex items-center justify-center min-h-screen bg-[#0a0f1e] px-4">
      <Card className="w-full max-w-md bg-[#1a2235] border-[#1e2d45]">
        <CardHeader className="text-center">
          <CardTitle className="text-[#f1f5f9]">
            {status === "loading" && "Verifying your login link..."}
            {status === "success" && "Login successful. Redirecting..."}
            {status === "error" && "Verification Failed"}
          </CardTitle>
        </CardHeader>
        <CardContent className="flex flex-col items-center gap-4 pb-6">
          {status === "loading" && (
            <Loader2 className="h-10 w-10 animate-spin text-amber-500" />
          )}
          {status === "success" && (
            <Loader2 className="h-10 w-10 animate-spin text-emerald-400" />
          )}
          {status === "error" && (
            <>
              <XCircle className="h-10 w-10 text-red-400" />
              <p className="text-sm text-[#94a3b8] text-center">
                This link is invalid or has expired.
              </p>
              <Button
                onClick={() => router.push("/auth/login")}
                className="bg-amber-500 hover:bg-amber-600 text-[#0a0f1e]"
              >
                Back to Login
              </Button>
            </>
          )}
        </CardContent>
      </Card>
    </main>
  );
}
