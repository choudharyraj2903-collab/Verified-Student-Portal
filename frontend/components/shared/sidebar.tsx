"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import {
  Home,
  List,
  Plus,
  BadgeCheck,
  User,
  Inbox,
  Users,
  Shield,
  Activity,
  LogOut,
} from "lucide-react";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import { cn } from "@/lib/utils";
import { api } from "@/lib/api";
import { toast } from "sonner";
import type { UserRole } from "@/lib/auth-context";

interface NavItem {
  label: string;
  href: string;
  icon: React.ElementType;
}

const navByRole: Record<UserRole, NavItem[]> = {
  STUDENT: [
    { label: "Dashboard", href: "/dashboard", icon: Home },
    { label: "My Requests", href: "/verification", icon: List },
    { label: "Submit Request", href: "/verification/new", icon: Plus },
    { label: "Verified Card", href: "/verification/card", icon: BadgeCheck },
    { label: "Profile", href: "/profile/edit", icon: User },
  ],
  COUNCIL_ADMIN: [
    { label: "Dashboard", href: "/dashboard", icon: Home },
    { label: "Council Requests", href: "/council", icon: Inbox },
    { label: "Students", href: "/admin/students", icon: Users },
    { label: "Profile", href: "/profile/edit", icon: User },
  ],
  SUPER_ADMIN: [
    { label: "Dashboard", href: "/dashboard", icon: Home },
    { label: "Students", href: "/admin/students", icon: Users },
    { label: "Council Admins", href: "/admin/council-admins", icon: Shield },
    { label: "All Requests", href: "/admin/requests", icon: List },
    { label: "Audit Logs", href: "/admin/audit-logs", icon: Activity },
    { label: "Profile", href: "/profile/edit", icon: User },
  ],
};

interface SidebarProps {
  role: UserRole;
  fullName?: string;
  email?: string;
  roleBadge?: string;
}

export function Sidebar({ role, fullName, email, roleBadge }: SidebarProps) {
  const pathname = usePathname();
  const router = useRouter();
  const navItems = navByRole[role];

  const initials = fullName
    ? fullName.split(" ").map((w) => w[0]).join("").slice(0, 2).toUpperCase()
    : email?.slice(0, 2).toUpperCase() ?? "??";

  async function handleLogout() {
    await api.post("/auth/logout");
    router.push("/auth/login");
    toast.success("Logged out");
  }

  return (
    <aside className="flex flex-col w-60 min-h-screen bg-[#111827] border-r border-[#1e2d45] shrink-0">
      {/* Header */}
      <div className="p-4 border-b border-[#1e2d45]">
        <p className="text-xs font-semibold uppercase tracking-widest text-amber-500 mb-3">
          Campus Council Portal
        </p>
        <div className="flex items-center gap-3">
          <Avatar className="h-9 w-9 bg-amber-500/20">
            <AvatarFallback className="bg-amber-500/20 text-amber-400 text-sm font-semibold">
              {initials}
            </AvatarFallback>
          </Avatar>
          <div className="overflow-hidden">
            <p className="text-sm font-medium text-[#f1f5f9] truncate">
              {fullName ?? email ?? "User"}
            </p>
            <p className="text-xs text-[#94a3b8] truncate">
              {roleBadge ?? role.replace("_", " ")}
            </p>
          </div>
        </div>
      </div>

      {/* Nav */}
      <nav className="flex-1 p-3 space-y-1">
        {navItems.map(({ label, href, icon: Icon }) => {
          const active =
            href === "/dashboard" ? pathname === href : pathname.startsWith(href);
          return (
            <Link
              key={href}
              href={href}
              className={cn(
                "flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-colors",
                active
                  ? "bg-amber-500/20 text-amber-400"
                  : "text-[#94a3b8] hover:bg-[#1e2d45] hover:text-[#f1f5f9]"
              )}
            >
              <Icon className="h-4 w-4 shrink-0" />
              {label}
            </Link>
          );
        })}
      </nav>

      {/* Logout */}
      <div className="p-3 border-t border-[#1e2d45]">
        <button
          onClick={handleLogout}
          className="flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium text-[#94a3b8] hover:bg-red-500/10 hover:text-red-400 transition-colors w-full"
        >
          <LogOut className="h-4 w-4" />
          Logout
        </button>
      </div>
    </aside>
  );
}
