import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

export function StatusBadge({ status }: { status: string }) {
  const map: Record<string, string> = {
    PENDING: "bg-amber-500/20 text-amber-400 border-amber-500/30",
    APPROVED: "bg-emerald-500/20 text-emerald-400 border-emerald-500/30",
    REJECTED: "bg-red-500/20 text-red-400 border-red-500/30",
  };
  return (
    <Badge variant="outline" className={cn("font-medium", map[status] ?? "bg-slate-500/20 text-slate-400")}>
      {status}
    </Badge>
  );
}

export function SeverityBadge({ severity }: { severity: string }) {
  const map: Record<string, string> = {
    INFO: "bg-slate-500/20 text-slate-300 border-slate-500/30",
    WARN: "bg-amber-500/20 text-amber-400 border-amber-500/30",
    CRITICAL: "bg-red-500/20 text-red-400 border-red-500/30",
  };
  return (
    <Badge variant="outline" className={cn("font-medium", map[severity] ?? "bg-slate-500/20 text-slate-300")}>
      {severity}
    </Badge>
  );
}
