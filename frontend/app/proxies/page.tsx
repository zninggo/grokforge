"use client";

import { FormEvent, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Shell } from "@/components/Shell";
import { api, clearTokens, getAccessToken, type MeData } from "@/lib/api";

type ProxyItem = {
  id: number;
  display: string;
  scheme: string;
  host: string;
  port: number;
  enabled: boolean;
};

export default function ProxiesPage() {
  const router = useRouter();
  const [me, setMe] = useState<MeData | null>(null);
  const [text, setText] = useState("http://user:pass@1.2.3.4:8080\nsocks5://2.2.2.2:1080\n");
  const [items, setItems] = useState<ProxyItem[]>([]);
  const [error, setError] = useState("");
  const [msg, setMsg] = useState("");

  async function load() {
    const res = await api<{ items: ProxyItem[] }>("/api/v1/proxies", {}, true);
    if (res.success) setItems(res.data?.items || []);
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

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setMsg("");
    const res = await api<{ imported: number; errors?: string[] }>("/api/v1/proxies", {
      method: "PUT",
      body: JSON.stringify({ text }),
    }, true);
    if (!res.success) {
      setError(res.error?.message || "导入失败");
      return;
    }
    setMsg(`已导入 ${res.data?.imported || 0} 条`);
    await load();
  }

  return (
    <Shell title="代理池" subtitle="静态列表 · 密码脱敏展示" username={me?.username}>
      <form className="card" onSubmit={onSubmit} style={{ marginBottom: 16 }}>
        <label className="label">每行一个代理（# 注释）</label>
        <textarea
          className="input"
          style={{ minHeight: 140, fontFamily: "ui-monospace, monospace" }}
          value={text}
          onChange={(e) => setText(e.target.value)}
        />
        <button className="btn" type="submit" style={{ marginTop: 12 }}>
          整表替换导入
        </button>
      </form>
      {error ? <div className="error">{error}</div> : null}
      {msg ? <div className="ok" style={{ marginBottom: 10 }}>{msg}</div> : null}
      <div className="card">
        {items.length === 0 ? (
          <p className="muted" style={{ margin: 0 }}>暂无代理</p>
        ) : (
          <ul style={{ margin: 0, paddingLeft: 18 }}>
            {items.map((p) => (
              <li key={p.id} style={{ marginBottom: 6 }}>
                <code>{p.display}</code>
              </li>
            ))}
          </ul>
        )}
      </div>
    </Shell>
  );
}
