"use client";

import { ReactNode } from "react";
import { usePathname, useRouter } from "next/navigation";
import { clearTokens } from "@/lib/api";

const NAV = [
  { href: "/dashboard/", label: "总览" },
  { href: "/jobs/", label: "任务" },
  { href: "/accounts/", label: "账号" },
  { href: "/proxies/", label: "代理" },
  { href: "/email/", label: "邮箱" },
];

export function Shell({
  title,
  subtitle,
  username,
  children,
}: {
  title: string;
  subtitle?: string;
  username?: string;
  children: ReactNode;
}) {
  const pathname = usePathname() || "";
  const router = useRouter();

  function logout() {
    clearTokens();
    router.replace("/login/");
  }

  return (
    <div className="shell">
      <aside className="side">
        <h1>GrokForge</h1>
        <nav className="nav">
          {NAV.map((n) => {
            const active = pathname === n.href || pathname.startsWith(n.href);
            return (
              <a key={n.href} href={n.href} className={active ? "active" : undefined}>
                {n.label}
              </a>
            );
          })}
        </nav>
      </aside>
      <section className="main">
        <div className="topbar">
          <div>
            <div style={{ fontSize: 22, fontWeight: 700 }}>{title}</div>
            {subtitle ? (
              <div className="muted" style={{ fontSize: 13 }}>
                {subtitle}
              </div>
            ) : null}
          </div>
          <div style={{ display: "flex", gap: 10, alignItems: "center" }}>
            <span className="badge">{username || "…"}</span>
            <button className="btn btn-ghost" onClick={logout} type="button">
              退出
            </button>
          </div>
        </div>
        {children}
      </section>
    </div>
  );
}
