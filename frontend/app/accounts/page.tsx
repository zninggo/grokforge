"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Shell } from "@/components/Shell";
import { api, clearTokens, getAccessToken, type MeData } from "@/lib/api";

type Account = {
  id: number;
  email: string;
  upstream: string;
  status: string;
  health_status: string;
  health_detail: string;
  has_credential: boolean;
  created_at: string;
};

type ListData = { items: Account[]; total: number };

export default function AccountsPage() {
  const router = useRouter();
  const [me, setMe] = useState<MeData | null>(null);
  const [items, setItems] = useState<Account[]>([]);
  const [total, setTotal] = useState(0);
  const [error, setError] = useState("");
  const [msg, setMsg] = useState("");
  const [busy, setBusy] = useState(false);

  async function load() {
    const res = await api<ListData>("/api/v1/accounts?page_size=50", {}, true);
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

  async function exportAccounts(includeSecret: boolean) {
    setBusy(true);
    setMsg("");
    setError("");
    try {
      const res = await api<{ items: unknown[]; count: number }>("/api/v1/accounts/export", {
        method: "POST",
        body: JSON.stringify({ include_secret: includeSecret }),
      }, true);
      if (!res.success) {
        setError(res.error?.message || "导出失败");
        return;
      }
      const blob = new Blob([JSON.stringify(res.data, null, 2)], { type: "application/json" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = includeSecret ? "accounts-with-secrets.json" : "accounts.json";
      a.click();
      URL.revokeObjectURL(url);
      setMsg(`已导出 ${res.data?.count || 0} 条（审计已记录）`);
    } finally {
      setBusy(false);
    }
  }

  async function probeOne(id: number) {
    setBusy(true);
    setError("");
    const res = await api<{ health_status: string; detail: string }>(`/api/v1/accounts/${id}/probe`, {
      method: "POST",
    }, true);
    setBusy(false);
    if (!res.success) {
      setError(res.error?.message || "测活失败");
      return;
    }
    setMsg(`#${id} → ${res.data?.health_status}: ${res.data?.detail}`);
    await load();
  }

  async function removeOne(id: number) {
    if (!confirm(`删除账号 #${id}？`)) return;
    setBusy(true);
    const res = await api(`/api/v1/accounts/${id}`, { method: "DELETE" }, true);
    setBusy(false);
    if (!res.success) {
      setError(res.error?.message || "删除失败");
      return;
    }
    await load();
  }

  return (
    <Shell title="账号" subtitle={`共 ${total} 条`} username={me?.username}>
      <div style={{ display: "flex", gap: 10, marginBottom: 14, flexWrap: "wrap" }}>
        <button className="btn" type="button" disabled={busy} onClick={() => exportAccounts(false)}>
          导出（脱敏）
        </button>
        <button className="btn btn-ghost" type="button" disabled={busy} onClick={() => exportAccounts(true)}>
          导出含凭证
        </button>
        <button className="btn btn-ghost" type="button" disabled={busy} onClick={() => load()}>
          刷新
        </button>
      </div>
      {error ? <div className="error">{error}</div> : null}
      {msg ? <div className="ok" style={{ marginBottom: 10 }}>{msg}</div> : null}
      <div className="card" style={{ padding: 0, overflow: "auto" }}>
        <table style={{ width: "100%", borderCollapse: "collapse", fontSize: 14 }}>
          <thead>
            <tr style={{ textAlign: "left", color: "var(--muted)" }}>
              <th style={th}>ID</th>
              <th style={th}>邮箱</th>
              <th style={th}>健康</th>
              <th style={th}>凭证</th>
              <th style={th}>操作</th>
            </tr>
          </thead>
          <tbody>
            {items.length === 0 ? (
              <tr>
                <td colSpan={5} style={{ padding: 16 }} className="muted">
                  暂无账号。可先 dry-run 注册或完成 live Cloak。
                </td>
              </tr>
            ) : (
              items.map((a) => (
                <tr key={a.id} style={{ borderTop: "1px solid var(--border)" }}>
                  <td style={td}>{a.id}</td>
                  <td style={td}>{a.email}</td>
                  <td style={td}>
                    <span className="badge">{a.health_status}</span>
                  </td>
                  <td style={td}>{a.has_credential ? "有" : "无"}</td>
                  <td style={td}>
                    <button className="btn btn-ghost" type="button" disabled={busy} onClick={() => probeOne(a.id)} style={{ marginRight: 6 }}>
                      测活
                    </button>
                    <button className="btn btn-ghost" type="button" disabled={busy} onClick={() => removeOne(a.id)}>
                      删除
                    </button>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </Shell>
  );
}

const th: React.CSSProperties = { padding: "12px 14px", fontWeight: 500 };
const td: React.CSSProperties = { padding: "12px 14px" };
