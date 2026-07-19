"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Shell } from "@/components/Shell";
import {
  api,
  clearTokens,
  getAccessToken,
  type MeData,
  type SystemInfo,
} from "@/lib/api";

type JobsPage = { total: number; items: { status: string }[] };
type AccountsPage = { total: number };

export default function DashboardPage() {
  const router = useRouter();
  const [me, setMe] = useState<MeData | null>(null);
  const [info, setInfo] = useState<SystemInfo | null>(null);
  const [jobsTotal, setJobsTotal] = useState(0);
  const [accountsTotal, setAccountsTotal] = useState(0);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!getAccessToken()) {
      router.replace("/login/");
      return;
    }
    (async () => {
      const [meRes, infoRes, jobsRes, accRes] = await Promise.all([
        api<MeData>("/api/v1/auth/me", {}, true),
        api<SystemInfo>("/api/v1/system/info", {}, true),
        api<JobsPage>("/api/v1/jobs?page_size=1", {}, true),
        api<AccountsPage>("/api/v1/accounts?page_size=1", {}, true),
      ]);
      if (!meRes.success) {
        clearTokens();
        router.replace("/login/");
        return;
      }
      setMe(meRes.data || null);
      if (infoRes.success) setInfo(infoRes.data || null);
      else setError(infoRes.error?.message || "加载系统信息失败");
      if (jobsRes.success) setJobsTotal(jobsRes.data?.total || 0);
      if (accRes.success) setAccountsTotal(accRes.data?.total || 0);
    })();
  }, [router]);

  return (
    <Shell title="总览" subtitle="注册控制面 · 单机" username={me?.username}>
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
          <div className="k">任务总数</div>
          <div className="v">{jobsTotal}</div>
        </div>
        <div className="stat">
          <div className="k">账号数</div>
          <div className="v">{accountsTotal}</div>
        </div>
      </div>
      <div className="card" style={{ marginTop: 18 }}>
        <h2 style={{ marginTop: 0, fontSize: 16 }}>快捷入口</h2>
        <p className="muted">
          <a href="/jobs/">创建 noop / register 任务</a>
          {" · "}
          <a href="/accounts/">查看账号与导出</a>
          {" · "}
          <a href="/proxies/">导入代理</a>
          {" · "}
          <a href="/email/">YYDS 连通测试</a>
        </p>
      </div>
    </Shell>
  );
}
