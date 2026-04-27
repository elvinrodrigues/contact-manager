"use client";

import { useEffect, useState } from "react";
import { useTheme } from "next-themes";
import { Users, Moon, Sun, Sparkles } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

export function Navbar() {
  const { theme, setTheme } = useTheme();
  const [mounted, setMounted] = useState(false);
  const [scrolled, setScrolled] = useState(false);

  // Avoid hydration mismatch for theme icon
  useEffect(() => {
    setMounted(true);
  }, []);

  // Add shadow on scroll
  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 4);
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  const isDark = mounted && theme === "dark";

  return (
    <header
      className={cn(
        "sticky top-0 z-50 w-full border-b border-border/60",
        "bg-background/80 backdrop-blur-lg backdrop-saturate-150",
        "transition-shadow duration-200",
        scrolled && "shadow-sm",
      )}
    >
      <div className="max-w-6xl mx-auto px-4 sm:px-6 h-16 flex items-center justify-between gap-4">
        {/* ── Brand ─────────────────────────────────────────── */}
        <div className="flex items-center gap-3 select-none">
          {/* Logo mark */}
          <div className="relative w-9 h-9 flex-shrink-0">
            {/* Glow layer */}
            <div className="absolute inset-0 rounded-xl bg-primary/30 blur-md" />
            {/* Icon container */}
            <div className="relative w-9 h-9 rounded-xl bg-primary flex items-center justify-center shadow-sm">
              <Users
                size={17}
                className="text-primary-foreground"
                strokeWidth={2.2}
              />
            </div>
          </div>

          {/* Name + tag */}
          <div className="flex flex-col leading-none">
            <span className="font-bold text-[15px] tracking-tight text-foreground">
              Contact Manager
            </span>
            <span className="hidden sm:flex items-center gap-0.5 text-[10px] font-medium text-muted-foreground leading-none mt-0.5">
              <Sparkles size={8} className="text-primary/70" />
              Contact Manager
            </span>
          </div>
        </div>

        {/* ── Right side ────────────────────────────────────── */}
        <div className="flex items-center gap-2">
          {/* Dark / Light toggle */}
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setTheme(isDark ? "light" : "dark")}
            aria-label={
              mounted
                ? isDark
                  ? "Switch to light mode"
                  : "Switch to dark mode"
                : "Toggle theme"
            }
            className={cn(
              "relative h-9 w-9 rounded-xl overflow-hidden",
              "text-muted-foreground hover:text-foreground",
              "hover:bg-accent transition-colors duration-150",
            )}
          >
            {/* Sun icon — visible in dark mode */}
            <span
              className={cn(
                "absolute inset-0 flex items-center justify-center transition-all duration-300",
                mounted && isDark
                  ? "opacity-100 rotate-0 scale-100"
                  : "opacity-0 rotate-90 scale-50",
              )}
            >
              <Sun size={17} strokeWidth={2} />
            </span>

            {/* Moon icon — visible in light mode */}
            <span
              className={cn(
                "absolute inset-0 flex items-center justify-center transition-all duration-300",
                mounted && !isDark
                  ? "opacity-100 rotate-0 scale-100"
                  : "opacity-0 -rotate-90 scale-50",
              )}
            >
              <Moon size={17} strokeWidth={2} />
            </span>
          </Button>
        </div>
      </div>
    </header>
  );
}
