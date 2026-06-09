import { Loader2, Inbox } from "lucide-react";
import { Button } from "@/components/ui/button";

export function LoadingSpinner({ text = "Loading..." }: { text?: string }) {
  return (
    <div className="flex flex-col items-center justify-center min-h-[200px] gap-3 text-[#94a3b8]">
      <Loader2 className="h-8 w-8 animate-spin text-amber-500" />
      <p className="text-sm">{text}</p>
    </div>
  );
}

export function EmptyState({
  title,
  description,
  action,
  onAction,
}: {
  title: string;
  description?: string;
  action?: string;
  onAction?: () => void;
}) {
  return (
    <div className="flex flex-col items-center justify-center min-h-[200px] gap-4 text-center py-12">
      <Inbox className="h-12 w-12 text-[#475569]" />
      <div>
        <p className="font-semibold text-[#94a3b8]">{title}</p>
        {description && <p className="text-sm text-[#475569] mt-1">{description}</p>}
      </div>
      {action && onAction && (
        <Button onClick={onAction} className="bg-amber-500 hover:bg-amber-600 text-[#0a0f1e]">
          {action}
        </Button>
      )}
    </div>
  );
}

export function ErrorMessage({ message }: { message: string }) {
  return (
    <div className="rounded-lg border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-400">
      {message}
    </div>
  );
}

export function PageHeader({
  title,
  subtitle,
  action,
}: {
  title: string;
  subtitle?: string;
  action?: React.ReactNode;
}) {
  return (
    <div className="flex items-start justify-between mb-6">
      <div>
        <h1 className="text-2xl font-bold text-[#f1f5f9]">{title}</h1>
        {subtitle && <p className="text-sm text-[#94a3b8] mt-1">{subtitle}</p>}
      </div>
      {action && <div>{action}</div>}
    </div>
  );
}
