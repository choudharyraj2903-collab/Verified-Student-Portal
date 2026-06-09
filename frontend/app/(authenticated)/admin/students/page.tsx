"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { PageHeader, LoadingSpinner, EmptyState } from "@/components/shared/ui-helpers";
import { Search } from "lucide-react";

interface Student {
  id: string; user_id: string; email: string; full_name: string;
  roll_number: string; year: number; branch: string;
  pending?: number; approved?: number; rejected?: number;
}
interface Result { students: Student[]; }

export default function AdminStudentsPage() {
  const router = useRouter();
  const [data, setData] = useState<Student[]>([]);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);

  useEffect(() => {
    setLoading(true);
    api.get<Result>(`/admin/students?page=${page}`).then((r) => {
      if (r.success) setData(r.data?.students ?? []);
    }).finally(() => setLoading(false));
  }, [page]);

  const filtered = search
    ? data.filter((s) =>
        s.full_name?.toLowerCase().includes(search.toLowerCase()) ||
        s.roll_number?.toLowerCase().includes(search.toLowerCase()))
    : data;

  return (
    <div className="p-6 space-y-4">
      <PageHeader title="Students"
        action={
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-[#475569]" />
            <Input value={search} onChange={(e) => setSearch(e.target.value)}
              placeholder="Search name or roll no..."
              className="pl-9 bg-[#111827] border-[#1e2d45] text-[#f1f5f9] placeholder:text-[#475569] w-64" />
          </div>
        }
      />

      {loading ? <LoadingSpinner /> : filtered.length === 0 ? (
        <EmptyState title="No students found" />
      ) : (
        <>
          <div className="rounded-lg border border-[#1e2d45] overflow-hidden">
            <Table>
              <TableHeader>
                <TableRow className="border-[#1e2d45] hover:bg-transparent bg-[#111827]">
                  {["Name", "Roll No", "Email", "Year", "Branch", "Actions"].map((h) => (
                    <TableHead key={h} className="text-[#94a3b8]">{h}</TableHead>
                  ))}
                </TableRow>
              </TableHeader>
              <TableBody>
                {filtered.map((s) => (
                  <TableRow key={s.user_id} className="border-[#1e2d45] hover:bg-[#1e2d45]/50 cursor-pointer"
                    onClick={() => router.push(`/admin/students/${s.user_id}`)}>
                    <TableCell className="text-[#f1f5f9] font-medium">{s.full_name}</TableCell>
                    <TableCell className="text-[#94a3b8] font-mono text-xs">{s.roll_number}</TableCell>
                    <TableCell className="text-[#94a3b8] text-sm">{s.email}</TableCell>
                    <TableCell className="text-[#94a3b8]">{s.year}</TableCell>
                    <TableCell className="text-[#94a3b8]">{s.branch}</TableCell>
                    <TableCell>
                      <Button size="sm" variant="ghost"
                        className="h-7 px-2 text-xs text-amber-400 hover:text-amber-300 hover:bg-amber-500/10"
                        onClick={(e) => { e.stopPropagation(); router.push(`/admin/students/${s.user_id}`); }}>
                        View Details
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
          <div className="flex items-center justify-end gap-2">
            <Button size="sm" variant="ghost" disabled={page === 1}
              onClick={() => setPage((p) => p - 1)}
              className="text-[#94a3b8] hover:text-[#f1f5f9]">Previous</Button>
            <span className="text-sm text-[#94a3b8]">Page {page}</span>
            <Button size="sm" variant="ghost" disabled={filtered.length < 50}
              onClick={() => setPage((p) => p + 1)}
              className="text-[#94a3b8] hover:text-[#f1f5f9]">Next</Button>
          </div>
        </>
      )}
    </div>
  );
}
