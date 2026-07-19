"use client";

import { FormEvent, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { api, setTokens, type LoginData, type SetupStatus } from "@/lib/api";

export default function LoginPage() {
  const router = useRouter();
  const [username, setUsername] = useState("admin");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    (async () => {
      const st = await api<SetupStatus>("/api/v1/setup/status");
      if (st.success && st.data && !st.data.completed) {
        router.replace("/setup/");
      }
    })();
  }, [router]);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      const res = await api<LoginData>("/api/v1/auth/login", {
        method: "POST",
        body: JSON.stringify({ username, password }),
      });
      if (!res.success || !res.data) {
        setError(res.error?.message || "登录失败");
        return;
      }
      setTokens(res.data.access_token, res.data.refresh_token);
      router.replace("/dashboard/");
    } catch {
      setError("网络错误");
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="center-page">
      <div className="stack card">
        <h1 style={{ margin: "0 0 6px" }}>登录 GrokForge</h1>
        <p className="muted" style={{ marginTop: 0 }}>单管理员控制面</p>
        <form onSubmit={onSubmit}>
          <div className="field">
            <label className="label">用户名</label>
            <input className="input" value={username} onChange={(e) => setUsername(e.target.value)} autoComplete="username" />
          </div>
          <div className="field">
            <label className="label">密码</label>
            <input className="input" type="password" value={password} onChange={(e) => setPassword(e.target.value)} autoComplete="current-password" />
          </div>
          {error ? <div className="error">{error}</div> : null}
          <button className="btn" type="submit" disabled={loading} style={{ width: "100%", marginTop: 8 }}>
            {loading ? "登录中…" : "登录"}
          </button>
        </form>
      </div>
    </main>
  );
}
