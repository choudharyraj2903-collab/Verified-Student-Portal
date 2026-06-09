import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import { Toaster } from "sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import "./globals.css";

const geistSans = Geist({ variable: "--font-geist-sans", subsets: ["latin"] });
const geistMono = Geist_Mono({ variable: "--font-geist-mono", subsets: ["latin"] });

export const metadata: Metadata = {
  title: "Campus Council Portal — IIT Kanpur",
  description: "Position of Responsibility Verification System",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={`${geistSans.variable} ${geistMono.variable} h-full`}>
      <body className="min-h-full flex flex-col bg-[#0a0f1e] text-[#f1f5f9] antialiased">
        <TooltipProvider>
          {children}
          <Toaster theme="dark" richColors position="top-right" />
        </TooltipProvider>
      </body>
    </html>
  );
}
