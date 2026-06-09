"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import { Button } from "@/components/ui/button";
import { Textarea } from "@/components/ui/textarea";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { PageHeader, LoadingSpinner, EmptyState } from "@/components/shared/ui-helpers";
import { StatusBadge } from "@/components/shared/status-badges";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";

interface CouncilRequest {
  request: { id: string; title: string; status: string; created_at: string; remarks?: string; };
  student: { full_name: string; roll_number: string; };
}
type Tab = "ALL" | "PENDING" | "APPROVED" | "REJECTED";
type Action = { id: string; type: "approve" | "reject" } | null;

export default function CouncilPage() {
  const { councilCodes } = useAuth();
  const councilCode = councilCodes[0] ?? "";
  const [requests, setRequests] = useState<CouncilRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState<Tab>("ALL");
  const [action, setAction] = useState<Action>(null);
  const [remarks, setRemarks] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function load() {
    if (!councilCode) { setLoading(false); return; }
    setLoading(true);
    const r = await api.get<CouncilRequest[]>(`/verification/council/${councilCode}`);
    if (r.success) setRequests(r.data ?? []);
    setLoading(false);
  }

  useEffect(() => { load(); }, [councilCode]);

  async function submit() {
    if (!action) return;
    if (action.type === "reject" && !remarks.trim()) return;
    setSubmitting(true);
    const r = await api.put(`/verification/${action.id}/${action.type}`, { remarks });
    if (r.success) {
      toast.success(`Request ${action.type === "approve" ? "approved" : "rejected"}.`);
      setAction(null);
      setRemarks("");
      load();
    } else {
      toast.error(r.message ?? "Action failed.");
    }
    setSubmitting(false);
  }

  const filtered = tab === "ALL" ? requests : requests.filter((r) => r.request.status === tab);

  return (
    <div className="p-6 space-y-4">
      <PageHeader title={`${councilCode || "Council"} — Requests`} />

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

      {loading ? <LoadingSpinner /> : filtered.length === 0 ? (
        <EmptyState title="No requests" description="No requests match this filter." />
      ) : (
        <div className="rounded-lg border border-[#1e2d45] overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow className="border-[#1e2d45] hover:bg-transparent bg-[#111827]">
                {["Student", "Roll No", "Title", "Submitted", "Status", "Actions"].map((h) => (
                  <TableHead key={h} className="text-[#94a3b8]">{h}</TableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {filtered.map(({ request: req, student }) => (
                <TableRow key={req.id} className="border-[#1e2d45] hover:bg-[#1e2d45]/50">
                  <TableCell className="text-[#f1f5f9]">{student.full_name}</TableCell>
                  <TableCell className="text-[#94a3b8] font-mono text-xs">{student.roll_number}</TableCell>
                  <TableCell className="text-[#94a3b8]">{req.title}</TableCell>
                  <TableCell className="text-[#94a3b8]">{new Date(req.created_at).toLocaleDateString()}</TableCell>
                  <TableCell>
                    {req.remarks ? (
                      <Tooltip>
                        <TooltipTrigger><StatusBadge status={req.status} /></TooltipTrigger>
                        <TooltipContent className="bg-[#1a2235] border-[#1e2d45] text-[#f1f5f9]">{req.remarks}</TooltipContent>
                      </Tooltip>
                    ) : <StatusBadge status={req.status} />}
                  </TableCell>
                  <TableCell>
                    {req.status === "PENDING" && (
                      <div className="flex gap-2">
                        <Button size="sm" className="h-7 px-2 text-xs bg-emerald-600 hover:bg-emerald-700 text-white"
                          onClick={() => { setAction({ id: req.id, type: "approve" }); setRemarks(""); }}>
                          Approve
                        </Button>
                        <Button size="sm" className="h-7 px-2 text-xs bg-red-600 hover:bg-red-700 text-white"
                          onClick={() => { setAction({ id: req.id, type: "reject" }); setRemarks(""); }}>
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
              Remarks {action?.type === "reject" ? <span className="text-red-400">*</span> : "(optional)"}
            </label>
            <Textarea value={remarks} onChange={(e) => setRemarks(e.target.value)} rows={3}
              placeholder={action?.type === "reject" ? "Required — provide reason for rejection" : "Optional remarks"}
              className="bg-[#111827] border-[#1e2d45] text-[#f1f5f9] placeholder:text-[#475569] resize-none" />
          </div>
          <DialogFooter>
            <Button variant="ghost" onClick={() => setAction(null)} className="text-[#94a3b8]">Cancel</Button>
            <Button onClick={submit} disabled={submitting || (action?.type === "reject" && !remarks.trim())}
              className={action?.type === "approve"
                ? "bg-emerald-600 hover:bg-emerald-700 text-white"
                : "bg-red-600 hover:bg-red-700 text-white"}>
              {submitting && <Loader2 className="h-4 w-4 animate-spin mr-2" />}
              Confirm {action?.type === "approve" ? "Approval" : "Rejection"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
