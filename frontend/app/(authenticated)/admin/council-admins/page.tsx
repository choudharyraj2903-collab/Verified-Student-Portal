"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { PageHeader, LoadingSpinner, ErrorMessage } from "@/components/shared/ui-helpers";
import { ConfirmDialog } from "@/components/shared/confirm-dialog";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";

interface Council { id: string; code: string; name: string; }
interface CouncilAdmin {
  id: string; email: string; council_code: string; council_name: string;
  assigned_at: string; is_active: boolean;
}

export default function CouncilAdminsPage() {
  const [councils, setCouncils] = useState<Council[]>([]);
  const [admins, setAdmins] = useState<CouncilAdmin[]>([]);
  const [form, setForm] = useState({ email: "", council_code: "" });
  const [formError, setFormError] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [loading, setLoading] = useState(true);
  const [removeId, setRemoveId] = useState<string | null>(null);

  async function load() {
    const [c, a] = await Promise.all([api.get<Council[]>("/councils"), api.get<CouncilAdmin[]>("/admin/council-admins")]);
    if (c.success) setCouncils(c.data ?? []);
    if (a.success) setAdmins(a.data ?? []);
    setLoading(false);
  }

  useEffect(() => { load(); }, []);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setFormError("");
    setSubmitting(true);
    const r = await api.post("/admin/council-admins", { email: form.email, council_code: form.council_code });
    if (r.success) {
      toast.success("Council admin assigned.");
      setForm({ email: "", council_code: "" });
      load();
    } else {
      setFormError(r.message ?? "Failed to assign admin.");
    }
    setSubmitting(false);
  }

  async function handleRemove() {
    if (!removeId) return;
    const r = await api.delete(`/admin/council-admins/${removeId}`);
    if (r.success) { toast.success("Council admin removed."); load(); }
    else toast.error(r.message ?? "Failed.");
    setRemoveId(null);
  }

  return (
    <div className="p-6 space-y-6">
      <PageHeader title="Council Admins" />

      <Card className="bg-[#1a2235] border-[#1e2d45]">
        <CardHeader className="pb-2">
          <CardTitle className="text-sm text-[#94a3b8]">Assign New Council Admin</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleCreate} className="flex gap-3 items-end flex-wrap">
            <div className="space-y-1.5 flex-1 min-w-48">
              <label className="text-xs text-[#475569]">Email</label>
              <Input value={form.email} onChange={(e) => setForm((f) => ({ ...f, email: e.target.value }))}
                placeholder="user@iitk.ac.in" required type="email"
                className="bg-[#111827] border-[#1e2d45] text-[#f1f5f9] placeholder:text-[#475569]" />
            </div>
            <div className="space-y-1.5 w-48">
              <label className="text-xs text-[#475569]">Council</label>
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
            <Button type="submit" disabled={submitting || !form.council_code}
              className="bg-amber-500 hover:bg-amber-600 text-[#0a0f1e]">
              {submitting && <Loader2 className="h-4 w-4 animate-spin mr-2" />}
              Assign Admin
            </Button>
          </form>
          {formError && <div className="mt-3"><ErrorMessage message={formError} /></div>}
        </CardContent>
      </Card>

      {loading ? <LoadingSpinner /> : (
        <div className="rounded-lg border border-[#1e2d45] overflow-hidden">
          <Table>
            <TableHeader>
              <TableRow className="border-[#1e2d45] hover:bg-transparent bg-[#111827]">
                {["Email", "Council", "Assigned", "Active", "Actions"].map((h) => (
                  <TableHead key={h} className="text-[#94a3b8]">{h}</TableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {admins.length === 0 ? (
                <TableRow><TableCell colSpan={5} className="text-center text-[#475569] py-8">No council admins yet.</TableCell></TableRow>
              ) : admins.map((a) => (
                <TableRow key={a.id} className="border-[#1e2d45] hover:bg-[#1e2d45]/50">
                  <TableCell className="text-[#f1f5f9]">{a.email}</TableCell>
                  <TableCell className="text-[#94a3b8]">{a.council_name} ({a.council_code})</TableCell>
                  <TableCell className="text-[#94a3b8]">{new Date(a.assigned_at).toLocaleDateString()}</TableCell>
                  <TableCell>
                    <span className={a.is_active ? "text-emerald-400" : "text-[#475569]"}>
                      {a.is_active ? "Active" : "Inactive"}
                    </span>
                  </TableCell>
                  <TableCell>
                    <Button size="sm" variant="ghost"
                      className="h-7 px-2 text-xs text-red-400 hover:text-red-300 hover:bg-red-500/10"
                      onClick={() => setRemoveId(a.id)}>
                      Remove
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      <ConfirmDialog open={!!removeId} onOpenChange={(o) => !o && setRemoveId(null)}
        title="Remove Council Admin" description="This will revoke their council admin role immediately."
        confirmLabel="Remove" onConfirm={handleRemove} destructive />
    </div>
  );
}
