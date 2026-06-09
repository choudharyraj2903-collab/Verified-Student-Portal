"use client";

import { useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import { Separator } from "@/components/ui/separator";
import { Button } from "@/components/ui/button";
import { LoadingSpinner } from "@/components/shared/ui-helpers";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { Printer, BadgeCheck } from "lucide-react";

interface VerifiedRecord {
  id: string; title: string; description: string; status: string;
  por_date: string; reviewed_at: string; remarks?: string;
}
interface VerifiedCard {
  student: { full_name: string; roll_number: string; year: number; branch: string; };
  verified_records: Record<string, VerifiedRecord[]>;
  total_verified: number;
  generated_at: string;
}

export default function VerifiedCardPage() {
  const searchParams = useSearchParams();
  const { profile } = useAuth();
  const [card, setCard] = useState<VerifiedCard | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const userIdParam = searchParams.get("userId");
    const userId = userIdParam ?? profile?.profile.user_id;
    if (!userId) { setLoading(false); return; }

    api.get<VerifiedCard>(`/verification/card/${userId}`)
      .then((r) => { if (r.success) setCard(r.data); })
      .finally(() => setLoading(false));
  }, [searchParams, profile]);

  if (loading) return <div className="p-6"><LoadingSpinner /></div>;

  if (!card) return (
    <div className="flex items-center justify-center min-h-screen">
      <p className="text-[#94a3b8]">Card not available.</p>
    </div>
  );

  const { student, verified_records } = card;
  const initials = student.full_name.split(" ").map((w) => w[0]).join("").slice(0, 2).toUpperCase();
  const councils = Object.entries(verified_records ?? {}).filter(([, recs]) =>
    recs.some((r) => r.status === "APPROVED")
  );

  return (
    <div className="p-6">
      <div className="flex justify-end mb-4 print:hidden">
        <Button onClick={() => window.print()} variant="outline"
          className="border-[#1e2d45] text-[#94a3b8] hover:bg-[#1e2d45] hover:text-[#f1f5f9]">
          <Printer className="h-4 w-4 mr-2" />Print
        </Button>
      </div>

      <div className="max-w-2xl mx-auto rounded-2xl border border-[#1e2d45] bg-[#1a2235] overflow-hidden shadow-2xl">
        {/* Header */}
        <div className="bg-gradient-to-r from-amber-500/20 to-amber-600/10 border-b border-[#1e2d45] px-8 py-8 text-center">
          <Avatar className="h-20 w-20 mx-auto mb-4">
            <AvatarFallback className="bg-amber-500/20 text-amber-400 text-2xl font-bold">{initials}</AvatarFallback>
          </Avatar>
          <h1 className="text-2xl font-bold text-[#f1f5f9]">{student.full_name}</h1>
          <p className="text-[#94a3b8] mt-1">{student.roll_number} · Year {student.year} · {student.branch}</p>
          <div className="flex items-center justify-center gap-1.5 mt-3">
            <BadgeCheck className="h-4 w-4 text-amber-500" />
            <span className="text-sm text-amber-400 font-medium">{card.total_verified} Verified Position{card.total_verified !== 1 ? "s" : ""}</span>
          </div>
        </div>

        {/* Records */}
        <div className="px-8 py-6 space-y-6">
          {councils.length === 0 ? (
            <p className="text-center text-[#94a3b8] py-8">No verified records yet.</p>
          ) : (
            councils.map(([councilId, records]) => {
              const approved = records.filter((r) => r.status === "APPROVED");
              return (
                <div key={councilId}>
                  <p className="text-xs font-semibold uppercase tracking-widest text-amber-500 mb-3">
                    Council: {councilId.slice(0, 8)}…
                  </p>
                  <div className="space-y-4">
                    {approved.map((rec) => (
                      <div key={rec.id} className="rounded-lg border border-[#1e2d45] bg-[#111827] p-4">
                        <div className="flex items-start justify-between">
                          <p className="font-semibold text-[#f1f5f9]">{rec.title}</p>
                          <BadgeCheck className="h-4 w-4 text-emerald-400 shrink-0 mt-0.5" />
                        </div>
                        <p className="text-sm text-[#94a3b8] mt-1">{rec.description}</p>
                        <div className="flex gap-4 mt-2 text-xs text-[#475569]">
                          <span>PoR Date: {new Date(rec.por_date).toLocaleDateString()}</span>
                          {rec.reviewed_at && <span>Approved: {new Date(rec.reviewed_at).toLocaleDateString()}</span>}
                        </div>
                        {rec.remarks && (
                          <p className="mt-2 text-xs text-[#94a3b8] italic">"{rec.remarks}"</p>
                        )}
                      </div>
                    ))}
                  </div>
                  <Separator className="mt-6 bg-[#1e2d45]" />
                </div>
              );
            })
          )}
          <p className="text-xs text-[#475569] text-center">Generated {new Date(card.generated_at).toLocaleString()}</p>
        </div>
      </div>
    </div>
  );
}
