"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { PageHeader, LoadingSpinner } from "@/components/shared/ui-helpers";
import { StatusBadge } from "@/components/shared/status-badges";
import { ArrowLeft, Download, Loader2 } from "lucide-react";
import { toast } from "sonner";

interface StudentDetail {
  profile: { id: string; user_id: string; full_name: string; roll_number: string; year: number; branch: string; email?: string; };
  records: Array<{ id: string; title: string; council_id: string; status: string; created_at: string; }>;
}

type Tab = "ALL" | "PENDING" | "APPROVED" | "REJECTED";

export default function StudentDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const { role } = useAuth();
  const [data, setData] = useState<StudentDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState<Tab>("ALL");
  const [deactivateOpen, setDeactivateOpen] = useState(false);
  const [reason, setReason] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [downloading, setDownloading] = useState(false);

  useEffect(() => {
    api.get<StudentDetail>(`/admin/students/${id}`)
      .then((r) => { if (r.success) setData(r.data); })
      .finally(() => setLoading(false));
  }, [id]);

  async function downloadReport() {
    setDownloading(true);
    const r = await api.get<Blob>(`/admin/students/${id}/report`);
    if (r.success && r.data) {
      const url = URL.createObjectURL(r.data as Blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `report_${data?.profile.roll_number ?? id}.pdf`;
      a.click();
      URL.revokeObjectURL(url);
    } else {
      toast.error("Failed to download report.");
    }
    setDownloading(false);
  }

  async function deactivate() {
    if (!reason.trim()) return;
    setSubmitting(true);
    const r = await api.post(`/admin/students/${id}/deactivate`, { reason });
    if (r.success) {
      toast.success("Student deactivated.");
      router.push("/admin/students");
    } else {
      toast.error(r.message ?? "Failed.");
    }
    setSubmitting(false);
    setDeactivateOpen(false);
  }

  if (loading) return <div className="p-6"><LoadingSpinner /></div>;
  if (!data) return <div className="p-6 text-[#94a3b8]">Student not found.</div>;

  const { profile, records } = data;
  const filtered = tab === "ALL" ? records : records.filter((r) => r.status === tab);

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <Button variant="ghost" onClick={() => router.back()}
          className="text-[#94a3b8] hover:text-[#f1f5f9] -ml-2">
          <ArrowLeft className="h-4 w-4 mr-2" />Back
        </Button>
        <div className="flex gap-2">
          <Button onClick={downloadReport} disabled={downloading}
            className="bg-amber-500 hover:bg-amber-600 text-[#0a0f1e]">
            {downloading ? <Loader2 className="h-4 w-4 animate-spin mr-2" /> : <Download className="h-4 w-4 mr-2" />}
            Download Report
          </Button>
          {role === "SUPER_ADMIN" && (
            <Button variant="outline" onClick={() => setDeactivateOpen(true)}
              className="border-red-500/50 text-red-400 hover:bg-red-500/10 hover:border-red-500">
              Deactivate Account
            </Button>
          )}
        </div>
      </div>

      <PageHeader title={profile.full_name} subtitle={`${profile.roll_number} · Year ${profile.year} · ${profile.branch}`} />

      <Card className="bg-[#1a2235] border-[#1e2d45]">
        <CardContent className="grid grid-cols-3 gap-4 p-4 text-sm">
          {[
            ["Roll Number", profile.roll_number],
            ["Year", `Year ${profile.year}`],
            ["Branch", profile.branch],
          ].map(([k, v]) => (
            <div key={k}>
              <p className="text-[#475569]">{k}</p>
              <p className="text-[#f1f5f9] font-medium mt-0.5">{v}</p>
            </div>
          ))}
        </CardContent>
      </Card>

      <Tabs value={tab} onValueChange={(v) => setTab(v as Tab)}>
        <TabsList className="bg-[#111827] border border-[#1e2d45]">
          {(["ALL", "PENDING", "APPROVED", "REJECTED"] as Tab[]).map((t) => (
            <TabsTrigger key={t} value={t}
              className="data-[state=active]:bg-amber-500 data-[state=active]:text-[#0a0f1e] text-[#94a3b8]">
              {t.charAt(0) + t.slice(1).toLowerCase()}
            </TabsTrigger>
          ))}
        </TabsList>
      </Tabs>

      <div className="rounded-lg border border-[#1e2d45] overflow-hidden">
        <Table>
          <TableHeader>
            <TableRow className="border-[#1e2d45] hover:bg-transparent bg-[#111827]">
              {["Title", "Council", "Date", "Status"].map((h) => (
                <TableHead key={h} className="text-[#94a3b8]">{h}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {filtered.length === 0 ? (
              <TableRow><TableCell colSpan={4} className="text-center text-[#475569] py-8">No records.</TableCell></TableRow>
            ) : filtered.map((r) => (
              <TableRow key={r.id} className="border-[#1e2d45] hover:bg-[#1e2d45]/50">
                <TableCell className="text-[#f1f5f9]">{r.title}</TableCell>
                <TableCell className="text-[#94a3b8] font-mono text-xs">{r.council_id.slice(0, 8)}</TableCell>
                <TableCell className="text-[#94a3b8]">{new Date(r.created_at).toLocaleDateString()}</TableCell>
                <TableCell><StatusBadge status={r.status} /></TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>

      <Dialog open={deactivateOpen} onOpenChange={setDeactivateOpen}>
        <DialogContent className="bg-[#1a2235] border-[#1e2d45]">
          <DialogHeader>
            <DialogTitle className="text-[#f1f5f9]">Deactivate Account</DialogTitle>
          </DialogHeader>
          <div className="space-y-2">
            <label className="text-sm text-[#94a3b8]">Reason <span className="text-red-400">*</span></label>
            <Input value={reason} onChange={(e) => setReason(e.target.value)}
              placeholder="Provide reason for deactivation"
              className="bg-[#111827] border-[#1e2d45] text-[#f1f5f9] placeholder:text-[#475569]" />
          </div>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setDeactivateOpen(false)} className="text-[#94a3b8]">Cancel</Button>
            <Button onClick={deactivate} disabled={submitting || !reason.trim()}
              className="bg-red-600 hover:bg-red-700 text-white">
              {submitting && <Loader2 className="h-4 w-4 animate-spin mr-2" />}
              Deactivate
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
