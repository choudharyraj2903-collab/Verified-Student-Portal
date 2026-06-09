"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { PageHeader, LoadingSpinner, EmptyState } from "@/components/shared/ui-helpers";
import { StatusBadge } from "@/components/shared/status-badges";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { Plus } from "lucide-react";
import { toast } from "sonner";
import { cn } from "@/lib/utils";

interface VReq { id: string; title: string; council_id: string; status: string; created_at: string; }
type Tab = "ALL" | "PENDING" | "APPROVED" | "REJECTED";

export default function MyRequestsPage() {
  const router = useRouter();
  const [requests, setRequests] = useState<VReq[]>([]);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState<Tab>("ALL");
  const [withdrawId, setWithdrawId] = useState<string | null>(null);

  async function load() {
    setLoading(true);
    const r = await api.get<VReq[]>("/verification");
    if (r.success) setRequests(r.data ?? []);
    setLoading(false);
  }

  useEffect(() => { load(); }, []);

  async function withdraw() {
    if (!withdrawId) return;
    const r = await api.delete(`/verification/${withdrawId}`);
    if (r.success) { toast.success("Request withdrawn."); load(); }
    else toast.error(r.message ?? "Failed to withdraw.");
    setWithdrawId(null);
  }

  const filtered = tab === "ALL" ? requests : requests.filter((r) => r.status === tab);

  return (
    <div className="p-6 space-y-4">
      <PageHeader title="My Verification Requests"
        action={
          <Link href="/verification/new" className="inline-flex items-center gap-1.5 rounded-lg bg-amber-500 hover:bg-amber-600 text-[#0a0f1e] text-sm font-medium px-3 h-8 transition-colors">
            <Plus className="h-4 w-4" />New Request
          </Link>
        }
      />

      <Tabs value={tab} onValueChange={(v) => setTab((v ?? "ALL") as Tab)}>
        <TabsList className="bg-[#111827] border border-[#1e2d45]">
          {(["ALL","PENDING","APPROVED","REJECTED"] as Tab[]).map((t) => (
            <TabsTrigger key={t} value={t} className="data-[state=active]:bg-amber-500 data-[state=active]:text-[#0a0f1e] text-[#94a3b8]">
              {t[0] + t.slice(1).toLowerCase()}
            </TabsTrigger>
          ))}
        </TabsList>
      </Tabs>

      {loading ? <LoadingSpinner /> : filtered.length === 0 ? (
        <EmptyState title="No requests yet" description="Submit your first verification request."
          action="Submit Request" onAction={() => router.push("/verification/new")} />
      ) : (
        <div className="rounded-lg border border-[#1e2d45] overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow className="border-[#1e2d45] hover:bg-transparent bg-[#111827]">
                {["Title","Council","Submitted","Status","Actions"].map(h=><TableHead key={h} className="text-[#94a3b8]">{h}</TableHead>)}
              </TableRow>
            </TableHeader>
            <TableBody>
              {filtered.map((r) => (
                <TableRow key={r.id} className="border-[#1e2d45] hover:bg-[#1e2d45]/50">
                  <TableCell className="text-[#f1f5f9] font-medium">{r.title}</TableCell>
                  <TableCell className="text-[#94a3b8] text-xs font-mono">{r.council_id.slice(0,8)}</TableCell>
                  <TableCell className="text-[#94a3b8]">{new Date(r.created_at).toLocaleDateString()}</TableCell>
                  <TableCell><StatusBadge status={r.status} /></TableCell>
                  <TableCell>
                    {r.status === "PENDING" && (
                      <Button size="sm" variant="ghost" className="text-red-400 hover:text-red-300 hover:bg-red-500/10 h-7 px-2 text-xs" onClick={() => setWithdrawId(r.id)}>Withdraw</Button>
                    )}
                    {r.status === "REJECTED" && (
                      <Link href="/verification/new" className="text-amber-400 hover:text-amber-300 text-xs px-2 h-7 inline-flex items-center">Resubmit</Link>
                    )}
                    {r.status === "APPROVED" && (
                      <Link href="/verification/card" className="text-emerald-400 hover:text-emerald-300 text-xs px-2 h-7 inline-flex items-center">View Card</Link>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      <ConfirmDialog open={!!withdrawId} onOpenChange={(o) => !o && setWithdrawId(null)}
        title="Withdraw Request" description="Are you sure? This cannot be undone."
        confirmLabel="Withdraw" onConfirm={withdraw} destructive />
    </div>
  );
}
