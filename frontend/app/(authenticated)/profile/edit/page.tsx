"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { ErrorMessage, LoadingSpinner } from "@/components/shared/ui-helpers";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";

export default function ProfileEditPage() {
  const { profile, refresh } = useAuth();
  const [form, setForm] = useState({ full_name: "", year: "", branch: "", phone: "", bio: "" });
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [fetching, setFetching] = useState(true);

  useEffect(() => {
    api.get<{ profile: { full_name: string; year: number; branch: string; phone?: string; bio?: string } }>("/profile")
      .then((r) => {
        if (r.success && r.data?.profile) {
          const p = r.data.profile;
          setForm({
            full_name: p.full_name ?? "",
            year: String(p.year ?? ""),
            branch: p.branch ?? "",
            phone: p.phone ?? "",
            bio: p.bio ?? "",
          });
        }
      })
      .finally(() => setFetching(false));
  }, []);

  function set(k: string) {
    return (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
      setForm((f) => ({ ...f, [k]: e.target.value }));
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      const res = await api.put("/profile", {
        full_name: form.full_name,
        year: Number(form.year),
        branch: form.branch,
        phone: form.phone || null,
        bio: form.bio || null,
      });
      if (res.success) {
        await refresh();
        toast.success("Profile updated!");
      } else {
        setError(res.message ?? "Failed to update profile.");
      }
    } catch {
      setError("Network error. Please try again.");
    } finally {
      setLoading(false);
    }
  }

  if (fetching) return <div className="p-6"><LoadingSpinner /></div>;

  const rollNumber = profile?.profile.roll_number ?? "";

  return (
    <div className="flex items-start justify-center min-h-screen p-6 bg-[#0a0f1e]">
      <Card className="w-full max-w-lg bg-[#1a2235] border-[#1e2d45]">
        <CardHeader>
          <CardTitle className="text-[#f1f5f9]">Edit Profile</CardTitle>
          <CardDescription className="text-[#94a3b8]">
            Update your profile information.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-1.5">
              <label className="text-sm text-[#94a3b8]">Full Name <span className="text-red-400">*</span></label>
              <Input value={form.full_name} onChange={set("full_name")} required
                className="bg-[#111827] border-[#1e2d45] text-[#f1f5f9]" />
            </div>

            <div className="space-y-1.5">
              <label className="text-sm text-[#94a3b8]">Roll Number <span className="text-xs text-[#475569]">(cannot be changed)</span></label>
              <Input value={rollNumber} disabled
                className="bg-[#111827] border-[#1e2d45] text-[#475569] cursor-not-allowed" />
            </div>

            <div className="space-y-1.5">
              <label className="text-sm text-[#94a3b8]">Year <span className="text-red-400">*</span></label>
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
            </div>

            <div className="space-y-1.5">
              <label className="text-sm text-[#94a3b8]">Branch <span className="text-red-400">*</span></label>
              <Input value={form.branch} onChange={set("branch")} required
                className="bg-[#111827] border-[#1e2d45] text-[#f1f5f9]" />
            </div>

            <div className="space-y-1.5">
              <label className="text-sm text-[#94a3b8]">Phone <span className="text-xs text-[#475569]">Optional</span></label>
              <Input value={form.phone} onChange={set("phone")} type="tel"
                className="bg-[#111827] border-[#1e2d45] text-[#f1f5f9]" />
            </div>

            <div className="space-y-1.5">
              <label className="text-sm text-[#94a3b8]">Bio <span className="text-xs text-[#475569]">{form.bio.length}/300 — Optional</span></label>
              <Textarea value={form.bio} onChange={set("bio")} maxLength={300} rows={3}
                className="bg-[#111827] border-[#1e2d45] text-[#f1f5f9] resize-none" />
            </div>

            {error && <ErrorMessage message={error} />}

            <Button type="submit" disabled={loading || !form.year}
              className="w-full bg-amber-500 hover:bg-amber-600 text-[#0a0f1e] font-semibold">
              {loading && <Loader2 className="h-4 w-4 animate-spin mr-2" />}
              Save Changes
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
