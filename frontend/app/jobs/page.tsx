"use client";

import { FormEvent, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Shell } from "@/components/Shell";
import { api, clearTokens, getAccessToken, type MeData } from "@/lib/api";

type Job = {
  id: number;
  kind: string;
  status: string;
  progress: number;
  step: string;
  driver: string;
  error_code: string;
  created_at: string;
};

type ListData = { items: Job[]; total: number };

export default function JobsPage() {
  const router = useRouter();
  const [me, setMe] = useState<MeData | null>(null);
  const [items, setItems] = useState<Job[]>([]);
  const [total, setTotal] = useState(0);
  const [kind, setKind] = useState("noop");
  const [count, setCount] = useState(1);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function load() {
    const res = await api<ListData>("/api/v1/jobs?page_size=50", {}, true);
    if (!res.success) {
      setError(res.error?.message || "加载失败");
      return;
    }
    setItems(res.data?.items || []);
    setTotal(res.data?.total || 0);
  }

  useEffect(() => {
    if (!getAccessToken()) {
      router.replace("/login/");
      return;
    }
    (async () => {
      const meRes = await api<MeData>("/api/v1/auth/me", {}, true);
      if (!meRes.success) {
        clearTokens();
        router.replace("/login/");
        return;
      }
      setMe(meRes.data || null);
      await load();
    })();
  }, [router]);

  async function onCreate(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    const body: Record<string, unknown> = { kind, count };
    if (kind === "register") body.driver = "chatgpt.cloak";
    const res = await api("/api/v1/jobs", { method: "POST", body: JSON.stringify(body) }, true);
    setBusy(false);
    if (!res.success) {
      setError(res.error?.message || "创建失败");
      return;
    }
    await load();
  }

  return (
    <Shell title="任务" subtitle={`共 ${total} 条`} username={me?.username}>
      <form className="card" onSubmit={onCreate} style={{ marginBottom: 16 }}>
        <div style={{ display: "flex", gap: 12, flexWrap: "wrap", alignItems: "end" }}>
          <div className="field" style={{ margin: 0, minWidth: 160 }}>
            <label className="label">类型</label>
            <select className="input" value={kind} onChange={(e) => setKind(e.target.value)}>
              <option value="noop">noop</option>
              <option value="register">register</option>
            </select>
          </div>
          <div className="field" style={{ margin: 0, width: 100 }}>
            <label className="label">数量</label>
            <input className="input" type="number" min={1} max={20} value={count} onChange={(e) => setCount(Number(e.target.value))} />
          </div>
          <button className="btn" type="submit" disabled={busy}>
            创建任务
          </button>
          <button className="btn btn-ghost" type="button" disabled={busy} onClick={() => load()}>
            刷新
          </button>
        </div>
        {kind === "register" ? (
          <p className="muted" style={{ marginBottom: 0, marginTop: 10, fontSize: 13 }}>
            register 需 YYDS Key；无 Cloak 二进制时请设 `GROKFORGE_CLOAK_DRY_RUN=1`。
          </p>
        ) : null}
      </form>
      {error ? <div className="error">{error}</div> : null}
      <div className="card" style={{ padding: 0, overflow: "auto" }}>
        <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 14 }}>
          <thead>
            <tr style={{ textAlign: "left", color: "var(--muted)" }}>
              <th style={th}>ID</th>
              <th style={th}>Kind</th>
              <th style={th}>Status</th>
              <th style={th}>Progress</th>
              <th style={th}>Step</th>
              <th style={th}>Error</th>
            </tr>
          </thead>
          <tbody>
            {items.map((j) => (
              <tr key={j.id} style={{ borderTop: "1px solid var(--border)" }}>
                <td style={td}>{j.id}</td>
                <td style={td}>{j.kind}</td>
                <td style={td}>{j.status}</td>
                <td style={td}>{j.progress}%</td>
                <td style={td}>{j.step}</td>
                <td style={td}>{j.error_code || "—"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Shell>
  );
}

const th: React.CSSProperties = { padding: "12px 14px", fontWeight: 500 };
const td: React.CSSProperties = { padding: "12px 14px" };
