"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { api } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { ErrorMessage, PageHeader } from "@/components/shared/ui-helpers";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";

interface Council { id: string; code: string; name: string; }

export default function NewVerificationPage() {
  const router = useRouter();
  const [councils, setCouncils] = useState<Council[]>([]);
  const [form, setForm] = useState({ title: "", council_code: "", description: "", proof_link: "", por_date: "" });
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    api.get<Council[]>("/councils").then((r) => { if (r.success) setCouncils(r.data ?? []); });
  }, []);

  function set(k: string) {
    return (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) =>
      setForm((f) => ({ ...f, [k]: e.target.value }));
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");

    try { new URL(form.proof_link); } catch { setError("Proof link must be a valid URL."); return; }
    if (new Date(form.por_date) > new Date()) { setError("Date cannot be in the future."); return; }

    setLoading(true);
    try {
      const res = await api.post("/verification", {
        title: form.title,
        council_code: form.council_code,
        description: form.description,
        proof_link: form.proof_link,
        por_date: new Date(form.por_date).toISOString(),
      });
      if (res.success) {
        toast.success("Request submitted!");
        router.push("/verification");
      } else {
        setError(res.message ?? "Failed to submit request.");
      }
    } catch {
      setError("Network error. Please try again.");
    } finally {
      setLoading(false);
    }
  }

  const today = new Date().toISOString().split("T")[0];

  return (
    <div className="p-6">
      <PageHeader title="Submit Verification Request" />
      <div className="max-w-lg">
        <Card className="bg-[#1a2235] border-[#1e2d45]">
          <CardHeader>
            <CardTitle className="text-[#f1f5f9] text-base">New Request</CardTitle>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="space-y-1.5">
                <label className="text-sm text-[#94a3b8]">Title <span className="text-red-400">*</span></label>
                <Input value={form.title} onChange={set("title")} required
                  placeholder="e.g. Design Head, Event Coordinator"
                  className="bg-[#111827] border-[#1e2d45] text-[#f1f5f9] placeholder:text-[#475569]" />
              </div>

              <div className="space-y-1.5">
                <label className="text-sm text-[#94a3b8]">Council <span className="text-red-400">*</span></label>
                <Select value={form.council_code} onValueChange={(v) => setForm((f) => ({ ...f, council_code: v ?? "" }))}>
                  <SelectTrigger className="bg-[#111827] border-[#1e2d45] text-[#f1f5f9]">
                    <SelectValue placeholder="Select council" />
                  </SelectTrigger>
                  <SelectContent className="bg-[#1a2235] border-[#1e2d45]">
                    {councils.map((c) => (
                      <SelectItem key={c.id} value={c.code} className="text-[#f1f5f9] focus:bg-[#1e2d45]">
                        {c.name} ({c.code})
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-1.5">
                <label className="text-sm text-[#94a3b8]">Description <span className="text-red-400">*</span></label>
                <Textarea value={form.description} onChange={set("description")} required rows={4}
                  placeholder="Describe the role and responsibilities"
                  className="bg-[#111827] border-[#1e2d45] text-[#f1f5f9] placeholder:text-[#475569] resize-none" />
              </div>

              <div className="space-y-1.5">
                <label className="text-sm text-[#94a3b8]">Proof Link <span className="text-red-400">*</span></label>
                <Input value={form.proof_link} onChange={set("proof_link")} required
                  placeholder="https://drive.google.com/..."
                  className="bg-[#111827] border-[#1e2d45] text-[#f1f5f9] placeholder:text-[#475569]" />
              </div>

              <div className="space-y-1.5">
                <label className="text-sm text-[#94a3b8]">Date of PoR <span className="text-red-400">*</span></label>
                <Input type="date" value={form.por_date} onChange={set("por_date")} required max={today}
                  className="bg-[#111827] border-[#1e2d45] text-[#f1f5f9]" />
              </div>

              {error && <ErrorMessage message={error} />}

              <div className="flex gap-3 pt-2">
                <Button type="submit" disabled={loading || !form.council_code}
                  className="bg-amber-500 hover:bg-amber-600 text-[#0a0f1e] font-semibold">
                  {loading && <Loader2 className="h-4 w-4 animate-spin mr-2" />}
                  Submit for Verification
                </Button>
                <Link href="/verification" className="inline-flex items-center justify-center rounded-lg text-sm font-medium px-3 h-8 text-[#94a3b8] hover:text-[#f1f5f9] hover:bg-[#1e2d45] transition-colors">Cancel</Link>
              </div>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
