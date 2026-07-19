"use client";

import { FormEvent, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { api, type SetupStatus } from "@/lib/api";

export default function SetupPage() {
  const router = useRouter();
  const [username, setUsername] = useState("admin");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [checks, setChecks] = useState<Record<string, string>>({});
  const [completed, setCompleted] = useState(false);

  useEffect(() => {
    (async () => {
      const st = await api<SetupStatus>("/api/v1/setup/status");
      if (st.success && st.data) {
        setChecks(st.data.checks || {});
        setCompleted(st.data.completed);
        if (st.data.completed) router.replace("/login/");
      }
    })();
  }, [router]);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    if (password !== confirm) {
      setError("两次密码不一致");
      return;
    }
    if (password.length < 8) {
      setError("密码至少 8 位");
      return;
    }
    setLoading(true);
    try {
      const res = await api<{ completed: boolean }>("/api/v1/setup/init", {
        method: "POST",
        body: JSON.stringify({ username, password }),
      });
      if (!res.success) {
        setError(res.error?.message || "初始化失败");
        return;
      }
      router.replace("/login/");
    } catch {
      setError("网络错误");
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="center-page">
      <div className="stack card">
        <h1 style={{ margin: "0 0 6px" }}>GrokForge 安装向导</h1>
        <p className="muted" style={{ marginTop: 0 }}>
          首次启动必须设置管理员密码，无默认口令。
        </p>

        <div style={{ display: "flex", gap: 8, marginBottom: 16, flexWrap: "wrap" }}>
          {Object.entries(checks).map(([k, v]) => (
            <span key={k} className="badge">
              {k}: <span className={v === "up" ? "ok" : "error"}>{v}</span>
            </span>
          ))}
        </div>

        {completed ? (
          <p className="ok">已完成安装，正在跳转登录…</p>
        ) : (
          <form onSubmit={onSubmit}>
            <div className="field">
              <label className="label">用户名</label>
              <input className="input" value={username} onChange={(e) => setUsername(e.target.value)} autoComplete="username" />
            </div>
            <div className="field">
              <label className="label">密码（≥8）</label>
              <input className="input" type="password" value={password} onChange={(e) => setPassword(e.target.value)} autoComplete="new-password" />
            </div>
            <div className="field">
              <label className="label">确认密码</label>
              <input className="input" type="password" value={confirm} onChange={(e) => setConfirm(e.target.value)} autoComplete="new-password" />
            </div>
            {error ? <div className="error">{error}</div> : null}
            <button className="btn" type="submit" disabled={loading} style={{ width: "100%", marginTop: 8 }}>
              {loading ? "提交中…" : "完成安装"}
            </button>
          </form>
        )}
      </div>
    </main>
  );
}
