"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import {
  api,
  clearTokens,
  getAccessToken,
  type MeData,
  type SystemInfo,
} from "@/lib/api";

export default function DashboardPage() {
  const router = useRouter();
  const [me, setMe] = useState<MeData | null>(null);
  const [info, setInfo] = useState<SystemInfo | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!getAccessToken()) {
      router.replace("/login/");
      return;
    }
    (async () => {
      const [meRes, infoRes] = await Promise.all([
        api<MeData>("/api/v1/auth/me", {}, true),
        api<SystemInfo>("/api/v1/system/info", {}, true),
      ]);
      if (!meRes.success) {
        clearTokens();
        router.replace("/login/");
        return;
      }
      setMe(meRes.data || null);
      if (infoRes.success) setInfo(infoRes.data || null);
      else setError(infoRes.error?.message || "加载系统信息失败");
    })();
  }, [router]);

  function logout() {
    clearTokens();
    router.replace("/login/");
  }

  return (
    <div className="shell">
      <aside className="side">
        <h1>GrokForge</h1>
        <nav className="nav">
          <a className="active" href="/dashboard/">
            总览
          </a>
          <a href="#" onClick={(e) => e.preventDefault()} className="muted">
            任务（Phase 4）
          </a>
          <a href="#" onClick={(e) => e.preventDefault()} className="muted">
            账号（Phase 7）
          </a>
          <a href="#" onClick={(e) => e.preventDefault()} className="muted">
            设置（后续）
          </a>
        </nav>
      </aside>
      <section className="main">
        <div className="topbar">
          <div>
            <div style={{ fontSize: 22, fontWeight: 700 }}>总览</div>
            <div className="muted" style={{ fontSize: 13 }}>
              注册控制面 · 单机
            </div>
          </div>
          <div style={{ display: "flex", gap: 10, alignItems: "center" }}>
            <span className="badge">{me ? me.username : "…"}</span>
            <button className="btn btn-ghost" onClick={logout} type="button">
              退出
            </button>
          </div>
        </div>

        {error ? <div className="error">{error}</div> : null}

        <div className="grid">
          <div className="stat">
            <div className="k">版本</div>
            <div className="v" style={{ fontSize: 18 }}>
              {info?.version || "—"}
            </div>
          </div>
          <div className="stat">
            <div className="k">安装状态</div>
            <div className="v" style={{ fontSize: 18 }}>
              {info?.setup_completed ? "已完成" : "未完成"}
            </div>
          </div>
          <div className="stat">
            <div className="k">今日注册</div>
            <div className="v">0</div>
          </div>
          <div className="stat">
            <div className="k">队列</div>
            <div className="v">—</div>
          </div>
        </div>

        <div className="card" style={{ marginTop: 18 }}>
          <h2 style={{ marginTop: 0, fontSize: 16 }}>Phase 3 壳已就绪</h2>
          <p className="muted" style={{ marginBottom: 0 }}>
            后续 Phase 将接入任务引擎、代理、YYDS、Cloak 注册与测活。当前页面由 Go 同端口托管静态资源。
          </p>
        </div>
      </section>
    </div>
  );
}
