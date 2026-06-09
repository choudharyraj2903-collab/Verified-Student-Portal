"use client";

import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { api } from "@/lib/api";
import { CheckCircle, Shield, Loader2 } from "lucide-react";
import { ErrorMessage } from "@/components/shared/ui-helpers";

const IITK_DOMAIN = "@iitk.ac.in";

export default function LoginPage() {
  const [email, setEmail] = useState("");
  const [loading, setLoading] = useState(false);
  const [sent, setSent] = useState(false);
  const [error, setError] = useState("");

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");

    const normalised = email.trim().toLowerCase();
    if (!normalised.endsWith(IITK_DOMAIN)) {
      setError(`Email must end with ${IITK_DOMAIN}`);
      return;
    }

    setLoading(true);
    try {
      const res = await api.post("/auth/magic-link", { email: normalised });
      if (res.success) {
        setSent(true);
      } else {
        setError(res.message ?? "Something went wrong. Please try again.");
      }
    } catch {
      setError("Network error. Please try again.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="flex items-center justify-center min-h-screen bg-[#0a0f1e] px-4">
      <Card className="w-full max-w-md bg-[#1a2235] border-[#1e2d45]">
        <CardHeader className="text-center space-y-3">
          <div className="flex justify-center">
            <div className="rounded-full bg-amber-500/10 border border-amber-500/30 p-3">
              <Shield className="h-7 w-7 text-amber-500" />
            </div>
          </div>
          <CardTitle className="text-[#f1f5f9]">Campus Council Portal</CardTitle>
          <CardDescription className="text-[#94a3b8]">
            {sent ? "Check your inbox" : "Sign in to your account"}
          </CardDescription>
        </CardHeader>

        <CardContent>
          {sent ? (
            <div className="flex flex-col items-center gap-4 py-4 text-center">
              <CheckCircle className="h-12 w-12 text-emerald-400" />
              <div>
                <p className="font-semibold text-[#f1f5f9]">Login link sent!</p>
                <p className="text-sm text-[#94a3b8] mt-1">
                  Check your IIT Kanpur email inbox. The link expires in 15 minutes.
                </p>
              </div>
            </div>
          ) : (
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="space-y-1.5">
                <label className="text-sm text-[#94a3b8]">Institute Email</label>
                <Input
                  type="email"
                  placeholder="yourname@iitk.ac.in"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                  className="bg-[#111827] border-[#1e2d45] text-[#f1f5f9] placeholder:text-[#475569] focus-visible:ring-amber-500"
                />
              </div>
              {error && <ErrorMessage message={error} />}
              <Button
                type="submit"
                disabled={loading}
                className="w-full bg-amber-500 hover:bg-amber-600 text-[#0a0f1e] font-semibold"
              >
                {loading ? <Loader2 className="h-4 w-4 animate-spin mr-2" /> : null}
                Send Login Link
              </Button>
            </form>
          )}
        </CardContent>
      </Card>
    </main>
  );
}
