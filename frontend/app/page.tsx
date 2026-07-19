"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { api, getAccessToken, type SetupStatus } from "@/lib/api";

export default function HomePage() {
  const router = useRouter();
  const [msg, setMsg] = useState("加载中…");

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const st = await api<SetupStatus>("/api/v1/setup/status");
        if (cancelled) return;
        if (!st.success || !st.data?.completed) {
          router.replace("/setup/");
          return;
        }
        if (!getAccessToken()) {
          router.replace("/login/");
          return;
        }
        router.replace("/dashboard/");
      } catch {
        if (!cancelled) setMsg("无法连接后端，请确认服务已在 :17890 运行");
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [router]);

  return (
    <main className="center-page">
      <div className="muted">{msg}</div>
    </main>
  );
}
