"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Shell } from "@/components/Shell";
import { api, clearTokens, getAccessToken, type MeData } from "@/lib/api";

type EmailConfig = {
  provider: string;
  base_url: string;
  api_key_set: boolean;
  api_key_masked: string;
  domain: string;
};

export default function EmailPage() {
  const router = useRouter();
  const [me, setMe] = useState<MeData | null>(null);
  const [cfg, setCfg] = useState<EmailConfig | null>(null);
  const [testResult, setTestResult] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

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
      const c = await api<EmailConfig>("/api/v1/email/config", {}, true);
      if (c.success) setCfg(c.data || null);
    })();
  }, [router]);

  async function runTest() {
    setBusy(true);
    setError("");
    setTestResult("");
    const res = await api<{
      ok: boolean;
      domain_total?: number;
      account?: { address: string; id: string };
      create_error?: string;
    }>("/api/v1/email/test", { method: "POST" }, true);
    setBusy(false);
    if (!res.success) {
      setError(res.error?.message || "测试失败");
      return;
    }
    const d = res.data;
    setTestResult(
      `ok=${d?.ok} domains=${d?.domain_total ?? "?"} account=${d?.account?.address || d?.create_error || "—"}`,
    );
  }

  return (
    <Shell title="邮箱 / YYDS" subtitle="Key 仅服务端，界面脱敏" username={me?.username}>
      <div className="card" style={{ marginBottom: 16 }}>
        <div className="grid">
          <div className="stat">
            <div className="k">Provider</div>
            <div className="v" style={{ fontSize: 16 }}>{cfg?.provider || "—"}</div>
          </div>
          <div className="stat">
            <div className="k">API Key</div>
            <div className="v" style={{ fontSize: 16 }}>{cfg?.api_key_masked || (cfg?.api_key_set ? "****" : "未配置")}</div>
          </div>
          <div className="stat">
            <div className="k">Domain</div>
            <div className="v" style={{ fontSize: 16 }}>{cfg?.domain || "自动"}</div>
          </div>
        </div>
        <button className="btn" type="button" disabled={busy} onClick={runTest} style={{ marginTop: 14 }}>
          {busy ? "测试中…" : "连通测试（列域名+建邮）"}
        </button>
      </div>
      {error ? <div className="error">{error}</div> : null}
      {testResult ? <div className="card ok">{testResult}</div> : null}
    </Shell>
  );
}
