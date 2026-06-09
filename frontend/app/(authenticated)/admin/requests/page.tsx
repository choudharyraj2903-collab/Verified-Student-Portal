"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { PageHeader, LoadingSpinner, EmptyState } from "@/components/shared/ui-helpers";
import { StatusBadge } from "@/components/shared/status-badges";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";

interface Council { id: string; code: string; name: string; }
interface Request {
  id: string; title: string; status: string; council_code: string;
  created_at: string; user_id: string;
}
type Action = { id: string; type: "approve" | "reject" } | null;

export default function AdminRequestsPage() {
  const [requests, setRequests] = useState<Request[]>([]);
  const [councils, setCouncils] = useState<Council[]>([]);
  const [statusFilter, setStatusFilter] = useState("ALL");
  const [councilFilter, setCouncilFilter] = useState("ALL");
  const [loading, setLoading] = useState(true);
  const [action, setAction] = useState<Action>(null);
  const [remarks, setRemarks] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function load() {
    setLoading(true);
    const query = statusFilter !== "ALL" ? `?status=${statusFilter}` : "";
    const [r, c] = await Promise.all([
      api.get<Request[]>(`/admin/verification${query}`),
      api.get<Council[]>("/councils"),
    ]);
    if (r.success) setRequests(r.data ?? []);
    if (c.success) setCouncils(c.data ?? []);
    setLoading(false);
  }

  useEffect(() => { load(); }, [statusFilter]);

  async function submit() {
    if (!action) return;
    if (action.type === "reject" && !remarks.trim()) return;
    setSubmitting(true);
    const endpoint = action.type === "approve"
      ? `/admin/verification/${action.id}/approve`
      : `/admin/verification/${action.id}/reject`;
    const r = await api.put(endpoint, { remarks });
    if (r.success) {
      toast.success(`Request ${action.type}d.`);
      setAction(null); setRemarks(""); load();
    } else toast.error(r.message ?? "Action failed.");
    setSubmitting(false);
  }

  const filtered = councilFilter === "ALL" ? requests
    : requests.filter((r) => r.council_code === councilFilter);

  return (
    <div className="p-6 space-y-4">
      <PageHeader title="All Verification Requests" />

      <div className="flex gap-3 flex-wrap">
        <Select value={statusFilter} onValueChange={(v) => setStatusFilter(v ?? "ALL")}>
          <SelectTrigger className="w-36 bg-[#111827] border-[#1e2d45] text-[#f1f5f9]">
            <SelectValue />
          </SelectTrigger>
          <SelectContent className="bg-[#1a2235] border-[#1e2d45]">
            {["ALL", "PENDING", "APPROVED", "REJECTED"].map((s) => (
              <SelectItem key={s} value={s} className="text-[#f1f5f9] focus:bg-[#1e2d45]">
                {s.charAt(0) + s.slice(1).toLowerCase()}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Select value={councilFilter} onValueChange={(v) => setCouncilFilter(v ?? "ALL")}>
          <SelectTrigger className="w-44 bg-[#111827] border-[#1e2d45] text-[#f1f5f9]">
            <SelectValue placeholder="All Councils" />
          </SelectTrigger>
          <SelectContent className="bg-[#1a2235] border-[#1e2d45]">
            <SelectItem value="ALL" className="text-[#f1f5f9] focus:bg-[#1e2d45]">All Councils</SelectItem>
            {councils.map((c) => (
              <SelectItem key={c.id} value={c.code} className="text-[#f1f5f9] focus:bg-[#1e2d45]">
                {c.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {loading ? <LoadingSpinner /> : filtered.length === 0 ? (
        <EmptyState title="No requests found" />
      ) : (
        <div className="rounded-lg border border-[#1e2d45] overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow className="border-[#1e2d45] hover:bg-transparent bg-[#111827]">
                {["Council", "Title", "Date", "Status", "Actions"].map((h) => (
                  <TableHead key={h} className="text-[#94a3b8]">{h}</TableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {filtered.map((r) => (
                <TableRow key={r.id} className="border-[#1e2d45] hover:bg-[#1e2d45]/50">
                  <TableCell className="text-[#94a3b8] font-mono text-xs">{r.council_code}</TableCell>
                  <TableCell className="text-[#f1f5f9]">{r.title}</TableCell>
                  <TableCell className="text-[#94a3b8]">{new Date(r.created_at).toLocaleDateString()}</TableCell>
                  <TableCell><StatusBadge status={r.status} /></TableCell>
                  <TableCell>
                    {r.status === "PENDING" && (
                      <div className="flex gap-2">
                        <Button size="sm" className="h-7 px-2 text-xs bg-emerald-600 hover:bg-emerald-700 text-white"
                          onClick={() => { setAction({ id: r.id, type: "approve" }); setRemarks(""); }}>
                          Approve
                        </Button>
                        <Button size="sm" className="h-7 px-2 text-xs bg-red-600 hover:bg-red-700 text-white"
                          onClick={() => { setAction({ id: r.id, type: "reject" }); setRemarks(""); }}>
                          Reject
                        </Button>
                      </div>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      <Dialog open={!!action} onOpenChange={(o) => !o && setAction(null)}>
        <DialogContent className="bg-[#1a2235] border-[#1e2d45]">
          <DialogHeader>
            <DialogTitle className="text-[#f1f5f9]">
              {action?.type === "approve" ? "Approve Request" : "Reject Request"}
            </DialogTitle>
          </DialogHeader>
          <div className="space-y-2">
            <label className="text-sm text-[#94a3b8]">
              Remarks {action?.type === "reject" && <span className="text-red-400">*</span>}
            </label>
            <Textarea value={remarks} onChange={(e) => setRemarks(e.target.value)} rows={3}
              className="bg-[#111827] border-[#1e2d45] text-[#f1f5f9] resize-none" />
          </div>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setAction(null)} className="text-[#94a3b8]">Cancel</Button>
            <Button onClick={submit} disabled={submitting || (action?.type === "reject" && !remarks.trim())}
              className={action?.type === "approve" ? "bg-emerald-600 hover:bg-emerald-700 text-white" : "bg-red-600 hover:bg-red-700 text-white"}>
              {submitting && <Loader2 className="h-4 w-4 animate-spin mr-2" />}
              Confirm
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
