import { useState } from "react";
import { apiFetch } from "../lib/api";
import type { DispatchDebugResult } from "../lib/types";

export default function AdminDebugPage() {
  const [apiKey, setApiKey] = useState("usp_demo_key");
  const [modelAlias, setModelAlias] = useState("gpt-fast");
  const [trace, setTrace] = useState<DispatchDebugResult | null>(null);
  const [error, setError] = useState("");

  const debugDispatch = async () => {
    setError("");
    try {
      const payload = await apiFetch<DispatchDebugResult>("/api/admin/dispatch-debug", {
        method: "POST",
        body: JSON.stringify({ api_key: apiKey, model_alias: modelAlias }),
      });
      setTrace(payload);
    } catch (err) {
      setError(err instanceof Error ? err.message : "调度调试失败。");
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-slate-800">调度调试</h2>
        <p className="mt-2 text-sm text-slate-500">输入 API Key 和模型别名，检查路由命中、候选账号与跳过原因。</p>
      </div>
      {error ? <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div> : null}
      <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <div className="grid gap-4 lg:grid-cols-[1fr_1fr_auto]">
          <input className="rounded-xl border border-slate-200 bg-slate-50 px-4 py-3" value={apiKey} onChange={(e) => setApiKey(e.target.value)} placeholder="API Key" />
          <input className="rounded-xl border border-slate-200 bg-slate-50 px-4 py-3" value={modelAlias} onChange={(e) => setModelAlias(e.target.value)} placeholder="模型别名" />
          <button className="rounded-xl bg-blue-600 px-4 py-3 text-sm font-semibold text-white hover:bg-blue-700" onClick={() => void debugDispatch()}>
            开始调试
          </button>
        </div>
        {!trace ? (
          <div className="mt-5 rounded-xl border border-dashed border-slate-200 bg-slate-50 px-4 py-6 text-sm text-slate-500">
            还没有调试结果。输入参数后点击“开始调试”。
          </div>
        ) : (
          <div className="mt-5 rounded-2xl bg-slate-950 p-5 text-sm text-slate-200">
            <p>模型别名：{trace.model_alias}</p>
            <p className="mt-1">套餐：{trace.package_id}</p>
            <p className="mt-1">命中 Provider：{trace.selected_provider}</p>
            <p className="mt-1">命中账号：{trace.selected_account_id}</p>
            <p className="mt-1">上游模型：{trace.upstream_model}</p>
            <p className="mt-1">允许回退：{trace.fallback_allowed ? "是" : "否"}</p>
            <div className="mt-4 space-y-2">
              {trace.candidates.map((candidate) => (
                <div key={`${candidate.account_id}-${candidate.provider}`} className="rounded-lg border border-white/10 bg-white/5 px-4 py-3">
                  <div>{candidate.provider} / {candidate.account_id}</div>
                  <div className="text-xs text-slate-400">状态：{candidate.account_status || "未知"} / skip：{candidate.skip_reason || "无"}</div>
                  <div className="mt-1 text-xs text-slate-500">失败次数：{candidate.consecutive_failures ?? 0} / 冷却到：{candidate.cooldown_until || "无"}</div>
                </div>
              ))}
            </div>
          </div>
        )}
      </section>
    </div>
  );
}
