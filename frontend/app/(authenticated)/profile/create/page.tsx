"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { ErrorMessage } from "@/components/shared/ui-helpers";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";

export default function ProfileCreatePage() {
  const router = useRouter();
  const { refresh } = useAuth();
  const [form, setForm] = useState({ full_name: "", roll_number: "", year: "", branch: "", phone: "", bio: "" });
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  function set(k: string) {
    return (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
      setForm((f) => ({ ...f, [k]: e.target.value }));
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      const res = await api.post("/profile", {
        ...form,
        year: Number(form.year),
      });
      if (res.success) {
        await refresh();
        toast.success("Profile created!");
        router.push("/dashboard");
      } else if (res.error === "PROFILE_EXISTS") {
        router.replace("/profile/edit");
      } else {
        setError(res.message ?? "Failed to create profile.");
      }
    } catch {
      setError("Network error. Please try again.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex items-start justify-center min-h-screen p-6 bg-[#0a0f1e]">
      <Card className="w-full max-w-lg bg-[#1a2235] border-[#1e2d45]">
        <CardHeader>
          <CardTitle className="text-[#f1f5f9]">Complete Your Profile</CardTitle>
          <CardDescription className="text-[#94a3b8]">
            This information will appear on your verified card.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <Field label="Full Name" required>
              <Input value={form.full_name} onChange={set("full_name")} required
                className="bg-[#111827] border-[#1e2d45] text-[#f1f5f9] placeholder:text-[#475569]" />
            </Field>

            <Field label="Roll Number" required hint="e.g. 210123">
              <Input value={form.roll_number} onChange={set("roll_number")} required
                className="bg-[#111827] border-[#1e2d45] text-[#f1f5f9] placeholder:text-[#475569]" />
            </Field>

            <Field label="Year" required>
              <Select value={form.year} onValueChange={(v) => setForm((f) => ({ ...f, year: v ?? "" }))}>
                <SelectTrigger className="bg-[#111827] border-[#1e2d45] text-[#f1f5f9]">
                  <SelectValue placeholder="Select year" />
                </SelectTrigger>
                <SelectContent className="bg-[#1a2235] border-[#1e2d45]">
                  {[1, 2, 3, 4, 5].map((y) => (
                    <SelectItem key={y} value={String(y)} className="text-[#f1f5f9] focus:bg-[#1e2d45]">Year {y}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>

            <Field label="Branch" required>
              <Input value={form.branch} onChange={set("branch")} required
                className="bg-[#111827] border-[#1e2d45] text-[#f1f5f9] placeholder:text-[#475569]" />
            </Field>

            <Field label="Phone" hint="Optional">
              <Input value={form.phone} onChange={set("phone")} type="tel"
                className="bg-[#111827] border-[#1e2d45] text-[#f1f5f9] placeholder:text-[#475569]" />
            </Field>

            <Field label="Bio" hint={`${form.bio.length}/300 — Optional`}>
              <Textarea value={form.bio} onChange={set("bio")} maxLength={300} rows={3}
                className="bg-[#111827] border-[#1e2d45] text-[#f1f5f9] placeholder:text-[#475569] resize-none" />
            </Field>

            {error && <ErrorMessage message={error} />}

            <Button type="submit" disabled={loading || !form.year}
              className="w-full bg-amber-500 hover:bg-amber-600 text-[#0a0f1e] font-semibold">
              {loading && <Loader2 className="h-4 w-4 animate-spin mr-2" />}
              Save Profile
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}

function Field({ label, required, hint, children }: {
  label: string; required?: boolean; hint?: string; children: React.ReactNode;
}) {
  return (
    <div className="space-y-1.5">
      <label className="text-sm text-[#94a3b8]">
        {label}{required && <span className="text-red-400 ml-0.5">*</span>}
        {hint && <span className="text-[#475569] ml-2 text-xs">{hint}</span>}
      </label>
      {children}
    </div>
  );
}
