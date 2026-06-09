"use client";

import { useEffect, useState, useCallback } from "react";
import { api } from "@/lib/api";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { PageHeader, LoadingSpinner } from "@/components/shared/ui-helpers";
import { SeverityBadge } from "@/components/shared/status-badges";
import { ChevronLeft, ChevronRight } from "lucide-react";

interface AuditEvent {
  id: string; event_type: string; severity: string;
  created_at: string; user_id?: string; metadata?: Record<string, unknown>;
}
interface Result { logs: AuditEvent[]; page: number; total?: number; }

const EVENTS = ["", "LOGIN_SUCCESS", "LOGOUT", "ROLE_CHANGED", "VERIFICATION_REQUEST_SUBMITTED",
  "VERIFICATION_APPROVED", "VERIFICATION_REJECTED", "UNAUTHORIZED_SCOPE_ACCESS"];

export default function AuditLogsPage() {
  const [logs, setLogs] = useState<AuditEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [filters, setFilters] = useState({ event: "", severity: "", from: "", to: "" });

  const load = useCallback(async () => {
    setLoading(true);
    const params = new URLSearchParams({ page: String(page) });
    if (filters.event) params.set("event", filters.event);
    if (filters.severity) params.set("severity", filters.severity);
    if (filters.from) params.set("from", filters.from);
    if (filters.to) params.set("to", filters.to);

    const r = await api.get<Result>(`/admin/audit-logs?${params}`);
    if (r.success) setLogs((r.data as any)?.logs ?? r.data ?? []);
    setLoading(false);
  }, [page, filters]);

  useEffect(() => { load(); }, [load]);

  // Auto-refresh every 30s
  useEffect(() => {
    const t = setInterval(() => load(), 30000);
    return () => clearInterval(t);
  }, [load]);

  function setFilter(k: string, v: string) {
    setFilters((f) => ({ ...f, [k]: v }));
    setPage(1);
  }

  return (
    <div className="p-6 space-y-4">
      <PageHeader title="Audit Logs" />

      <div className="flex gap-3 flex-wrap">
        <Select value={filters.event || "ALL"} onValueChange={(v) => setFilter("event", (v ?? "ALL") === "ALL" ? "" : (v ?? ""))}>
          <SelectTrigger className="w-56 bg-[#111827] border-[#1e2d45] text-[#f1f5f9]">
            <SelectValue placeholder="All Events" />
          </SelectTrigger>
          <SelectContent className="bg-[#1a2235] border-[#1e2d45]">
            <SelectItem value="ALL" className="text-[#f1f5f9] focus:bg-[#1e2d45]">All Events</SelectItem>
            {EVENTS.filter(Boolean).map((e) => (
              <SelectItem key={e} value={e} className="text-[#f1f5f9] focus:bg-[#1e2d45] text-xs">{e}</SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Select value={filters.severity || "ALL"} onValueChange={(v) => setFilter("severity", (v ?? "ALL") === "ALL" ? "" : (v ?? ""))}>
          <SelectTrigger className="w-32 bg-[#111827] border-[#1e2d45] text-[#f1f5f9]">
            <SelectValue placeholder="Severity" />
          </SelectTrigger>
          <SelectContent className="bg-[#1a2235] border-[#1e2d45]">
            {["ALL", "INFO", "WARN", "CRITICAL"].map((s) => (
              <SelectItem key={s} value={s} className="text-[#f1f5f9] focus:bg-[#1e2d45]">{s}</SelectItem>
            ))}
          </SelectContent>
        </Select>

        <Input type="date" value={filters.from} onChange={(e) => setFilter("from", e.target.value)}
          placeholder="From" className="w-36 bg-[#111827] border-[#1e2d45] text-[#f1f5f9]" />
        <Input type="date" value={filters.to} onChange={(e) => setFilter("to", e.target.value)}
          placeholder="To" className="w-36 bg-[#111827] border-[#1e2d45] text-[#f1f5f9]" />
      </div>

      {loading ? <LoadingSpinner /> : (
        <div className="rounded-lg border border-[#1e2d45] overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow className="border-[#1e2d45] hover:bg-transparent bg-[#111827]">
                {["Timestamp", "Event", "Severity", "User", "Details"].map((h) => (
                  <TableHead key={h} className="text-[#94a3b8]">{h}</TableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {logs.length === 0 ? (
                <TableRow><TableCell colSpan={5} className="text-center text-[#475569] py-8">No logs found.</TableCell></TableRow>
              ) : logs.map((e) => (
                <TableRow key={e.id} className="border-[#1e2d45] hover:bg-[#1e2d45]/50">
                  <TableCell className="text-[#94a3b8] text-xs whitespace-nowrap">
                    {new Date(e.created_at).toLocaleString()}
                  </TableCell>
                  <TableCell className="text-[#f1f5f9] font-mono text-xs">{e.event_type}</TableCell>
                  <TableCell><SeverityBadge severity={e.severity} /></TableCell>
                  <TableCell className="text-[#94a3b8] font-mono text-xs">
                    {e.user_id ? e.user_id.slice(0, 8) + "…" : "—"}
                  </TableCell>
                  <TableCell>
                    {e.metadata ? (
                      <Tooltip>
                        <TooltipTrigger>
                          <span className="text-xs text-[#3b82f6] cursor-pointer hover:underline">View</span>
                        </TooltipTrigger>
                        <TooltipContent className="bg-[#1a2235] border-[#1e2d45] max-w-xs">
                          <pre className="text-xs text-[#f1f5f9] whitespace-pre-wrap">
                            {JSON.stringify(e.metadata, null, 2)}
                          </pre>
                        </TooltipContent>
                      </Tooltip>
                    ) : <span className="text-[#475569]">—</span>}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      <div className="flex items-center gap-2 justify-end">
        <Button variant="ghost" size="sm" onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page === 1}
          className="text-[#94a3b8] hover:text-[#f1f5f9]">
          <ChevronLeft className="h-4 w-4" />
        </Button>
        <span className="text-sm text-[#94a3b8]">Page {page}</span>
        <Button variant="ghost" size="sm" onClick={() => setPage((p) => p + 1)} disabled={logs.length === 0}
          className="text-[#94a3b8] hover:text-[#f1f5f9]">
          <ChevronRight className="h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}
