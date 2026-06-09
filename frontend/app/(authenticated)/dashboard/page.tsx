"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useAuth } from "@/lib/auth-context";
import { api } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { PageHeader, LoadingSpinner } from "@/components/shared/ui-helpers";
import { StatusBadge, SeverityBadge } from "@/components/shared/status-badges";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Plus, BadgeCheck, AlertCircle } from "lucide-react";
import { cn } from "@/lib/utils";

interface VReq { id: string; title: string; council_id: string; status: string; created_at: string; }
interface CouncilReq { request: VReq; student: { full_name: string; roll_number: string }; }
interface AuditEvent { id: string; event_type: string; severity: string; created_at: string; }
interface StudentResult { students: { id: string }[]; }

function Btn({ href, children, variant = "amber", className }: { href: string; children: React.ReactNode; variant?: "amber" | "outline"; className?: string }) {
  return (
    <Link href={href} className={cn(
      "inline-flex items-center justify-center rounded-lg text-sm font-medium px-3 h-8 transition-colors gap-1.5",
      variant === "amber" ? "bg-amber-500 hover:bg-amber-600 text-[#0a0f1e]" : "border border-[#1e2d45] text-[#94a3b8] hover:bg-[#1e2d45] hover:text-[#f1f5f9]",
      className
    )}>{children}</Link>
  );
}

function StudentDashboard() {
  const { profile } = useAuth();
  const [requests, setRequests] = useState<VReq[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    api.get<VReq[]>("/verification").then((r) => { if (r.success) setRequests(r.data ?? []); }).finally(() => setLoading(false));
  }, []);

  const pending = requests.filter((r) => r.status === "PENDING").length;
  const approved = requests.filter((r) => r.status === "APPROVED").length;
  const rejected = requests.filter((r) => r.status === "REJECTED").length;

  return (
    <div className="p-6 space-y-6">
      <PageHeader title="Dashboard" />
      <p className="text-xl font-semibold text-[#f1f5f9]">Welcome back, {profile?.profile.full_name ?? "Student"}</p>

      {profile && !profile.is_complete && (
        <div className="flex items-center justify-between rounded-lg border border-amber-500/40 bg-amber-500/10 px-4 py-3">
          <div className="flex items-center gap-2 text-amber-400">
            <AlertCircle className="h-4 w-4 shrink-0" />
            <span className="text-sm">Complete your profile to start submitting verification requests</span>
          </div>
          <Btn href="/profile/create" className="ml-4 shrink-0 h-7 text-xs">Complete Profile</Btn>
        </div>
      )}

      <div className="grid grid-cols-3 gap-4">
        {[
          { label: "Pending", count: pending, cls: "text-amber-400", bg: "bg-amber-500/10 border-amber-500/30" },
          { label: "Approved", count: approved, cls: "text-emerald-400", bg: "bg-emerald-500/10 border-emerald-500/30" },
          { label: "Rejected", count: rejected, cls: "text-red-400", bg: "bg-red-500/10 border-red-500/30" },
        ].map(({ label, count, cls, bg }) => (
          <Card key={label} className={`border ${bg} bg-[#1a2235]`}>
            <CardContent className="px-4 py-4">
              <p className={`text-sm font-medium ${cls}`}>{label}</p>
              <p className={`text-3xl font-bold mt-1 ${cls}`}>{count}</p>
            </CardContent>
          </Card>
        ))}
      </div>

      <Card className="bg-[#1a2235] border-[#1e2d45]">
        <CardHeader className="px-4 py-3 border-b border-[#1e2d45]">
          <CardTitle className="text-sm font-medium text-[#94a3b8]">Recent Requests</CardTitle>
        </CardHeader>
        <CardContent className="p-0">
          {loading ? <LoadingSpinner /> : requests.length === 0 ? (
            <p className="text-sm text-[#475569] p-4">No requests yet.</p>
          ) : (
            <Table>
              <TableHeader><TableRow className="border-[#1e2d45] hover:bg-transparent">
                {["Title","Date","Status"].map(h=><TableHead key={h} className="text-[#94a3b8]">{h}</TableHead>)}
              </TableRow></TableHeader>
              <TableBody>
                {requests.slice(0, 5).map((r) => (
                  <TableRow key={r.id} className="border-[#1e2d45] hover:bg-[#1e2d45]/50">
                    <TableCell className="text-[#f1f5f9]">{r.title}</TableCell>
                    <TableCell className="text-[#94a3b8]">{new Date(r.created_at).toLocaleDateString()}</TableCell>
                    <TableCell><StatusBadge status={r.status} /></TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <div className="flex gap-3">
        <Btn href="/verification/new"><Plus className="h-4 w-4" />Submit New Request</Btn>
        <Btn href="/verification/card" variant="outline"><BadgeCheck className="h-4 w-4" />View Verified Card</Btn>
      </div>
    </div>
  );
}

function CouncilAdminDashboard() {
  const { councilCodes } = useAuth();
  const councilCode = councilCodes[0] ?? "";
  const [requests, setRequests] = useState<CouncilReq[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!councilCode) { setLoading(false); return; }
    api.get<CouncilReq[]>(`/verification/council/${councilCode}`).then((r) => { if (r.success) setRequests(r.data ?? []); }).finally(() => setLoading(false));
  }, [councilCode]);

  const pending = requests.filter((r) => r.request.status === "PENDING");
  const reviewed = requests.filter((r) => r.request.status !== "PENDING");

  return (
    <div className="p-6 space-y-6">
      <PageHeader title="Dashboard" subtitle={councilCode ? `Council: ${councilCode}` : undefined} />
      <div className="grid grid-cols-2 gap-4">
        <Card className="bg-[#1a2235] border-amber-500/30"><CardContent className="px-4 py-4"><p className="text-sm font-medium text-amber-400">Pending</p><p className="text-3xl font-bold text-amber-400 mt-1">{pending.length}</p></CardContent></Card>
        <Card className="bg-[#1a2235] border-[#1e2d45]"><CardContent className="px-4 py-4"><p className="text-sm font-medium text-[#94a3b8]">Reviewed</p><p className="text-3xl font-bold text-[#f1f5f9] mt-1">{reviewed.length}</p></CardContent></Card>
      </div>
      <Card className="bg-[#1a2235] border-[#1e2d45]">
        <CardHeader className="px-4 py-3 border-b border-[#1e2d45]"><CardTitle className="text-sm font-medium text-[#94a3b8]">Recent Pending</CardTitle></CardHeader>
        <CardContent className="p-0">
          {loading ? <LoadingSpinner /> : pending.length === 0 ? <p className="text-sm text-[#475569] p-4">No pending requests.</p> : (
            <Table>
              <TableHeader><TableRow className="border-[#1e2d45] hover:bg-transparent">{["Student","Title","Date"].map(h=><TableHead key={h} className="text-[#94a3b8]">{h}</TableHead>)}</TableRow></TableHeader>
              <TableBody>{pending.slice(0,5).map((r)=>(
                <TableRow key={r.request.id} className="border-[#1e2d45] hover:bg-[#1e2d45]/50">
                  <TableCell className="text-[#f1f5f9]">{r.student.full_name}</TableCell>
                  <TableCell className="text-[#94a3b8]">{r.request.title}</TableCell>
                  <TableCell className="text-[#94a3b8]">{new Date(r.request.created_at).toLocaleDateString()}</TableCell>
                </TableRow>
              ))}</TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function SuperAdminDashboard() {
  const [students, setStudents] = useState(0);
  const [logs, setLogs] = useState<AuditEvent[]>([]);
  const [requests, setRequests] = useState<VReq[]>([]);
  const [admins, setAdmins] = useState(0);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.all([
      api.get<StudentResult>("/admin/students"),
      api.get<{ logs: AuditEvent[] }>("/admin/audit-logs?page=1"),
      api.get<VReq[]>("/admin/verification"),
      api.get<unknown[]>("/admin/council-admins"),
    ]).then(([s, l, r, a]) => {
      if (s.success) setStudents((s.data as StudentResult)?.students?.length ?? 0);
      if (l.success) setLogs((l.data as { logs: AuditEvent[] })?.logs ?? []);
      if (r.success) setRequests((r.data as VReq[]) ?? []);
      if (a.success) setAdmins(((a.data as unknown[]) ?? []).length);
    }).finally(() => setLoading(false));
  }, []);

  if (loading) return <div className="p-6"><LoadingSpinner /></div>;
  const pending = requests.filter((r) => r.status === "PENDING").length;

  return (
    <div className="p-6 space-y-6">
      <PageHeader title="Dashboard" subtitle="System Overview" />
      <div className="grid grid-cols-4 gap-4">
        {[["Total Students", students, "text-[#f1f5f9]"],["Total Requests", requests.length,"text-[#f1f5f9]"],["Pending", pending,"text-amber-400"],["Council Admins", admins,"text-[#f1f5f9]"]].map(([label, value, cls]) => (
          <Card key={String(label)} className="bg-[#1a2235] border-[#1e2d45]"><CardContent className="px-4 py-4"><p className="text-sm font-medium text-[#94a3b8]">{label}</p><p className={`text-3xl font-bold mt-1 ${cls}`}>{value}</p></CardContent></Card>
        ))}
      </div>
      <Card className="bg-[#1a2235] border-[#1e2d45]">
        <CardHeader className="px-4 py-3 border-b border-[#1e2d45]"><CardTitle className="text-sm font-medium text-[#94a3b8]">Recent Activity</CardTitle></CardHeader>
        <CardContent className="p-0">
          {logs.length === 0 ? <p className="text-sm text-[#475569] p-4">No recent activity.</p> : (
            <Table>
              <TableHeader><TableRow className="border-[#1e2d45] hover:bg-transparent">{["Event","Severity","Time"].map(h=><TableHead key={h} className="text-[#94a3b8]">{h}</TableHead>)}</TableRow></TableHeader>
              <TableBody>{logs.slice(0,10).map((e)=>(
                <TableRow key={e.id} className="border-[#1e2d45] hover:bg-[#1e2d45]/50">
                  <TableCell className="text-[#f1f5f9] font-mono text-xs">{e.event_type}</TableCell>
                  <TableCell><SeverityBadge severity={e.severity} /></TableCell>
                  <TableCell className="text-[#94a3b8] text-xs">{new Date(e.created_at).toLocaleString()}</TableCell>
                </TableRow>
              ))}</TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

export default function DashboardPage() {
  const { role } = useAuth();
  if (role === "STUDENT") return <StudentDashboard />;
  if (role === "COUNCIL_ADMIN") return <CouncilAdminDashboard />;
  if (role === "SUPER_ADMIN") return <SuperAdminDashboard />;
  return null;
}
