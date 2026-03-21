import { useEffect, useState } from "react";
import { apiFetch } from "../lib/api";
import { formatDate } from "../lib/format";
import type { AdminUpstreamAccountItem } from "../lib/types";

export default function AdminAccountsPage() {
  const [accounts, setAccounts] = useState<AdminUpstreamAccountItem[]>([]);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  const load = async () => {
    const payload = await apiFetch<AdminUpstreamAccountItem[]>("/api/admin/upstream-accounts");
    setAccounts(payload ?? []);
  };

  useEffect(() => {
    load().catch((err: Error) => setError(err.message));
  }, []);

  const runAction = async (id: string, action: "test" | "refresh" | "recover" | "clear-cooldown") => {
    setError("");
    setMessage("");
    try {
      await apiFetch(`/api/admin/upstream-accounts/${id}/${action}`, {
        method: "POST",
        body: JSON.stringify({}),
      });
      await load();
      setMessage(`账号 ${id} 操作已执行：${action}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "操作失败。");
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-slate-800">账号池运维</h2>
        <p className="mt-2 text-sm text-slate-500">查看上游账号状态，并执行健康检查、刷新、恢复与清冷却操作。</p>
      </div>
      {message ? <div className="rounded-xl border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-700">{message}</div> : null}
      {error ? <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div> : null}
      <div className="grid gap-4">
        {accounts.map((account) => (
          <article key={account.id} className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
            <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
              <div>
                <div className="flex items-center gap-3">
                  <h3 className="text-lg font-semibold text-slate-800">{account.display_name}</h3>
                  <span className={`rounded-full px-3 py-1 text-xs font-medium ${account.status === "active" ? "bg-emerald-100 text-emerald-700" : "bg-amber-100 text-amber-700"}`}>
                    {account.status}
                  </span>
                </div>
                <div className="mt-3 space-y-1 text-sm text-slate-500">
                  <p>Provider：{account.provider}</p>
                  <p>账号邮箱：{account.email || "未设置"}</p>
                  <p>鉴权方式：{account.auth_mode || "未设置"}</p>
                  <p>支持模型：{account.supports_models?.join(", ") || "未设置"}</p>
                  <p>最近刷新：{formatDate(account.last_refreshed_at)}</p>
                </div>
              </div>
              <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
                <button className="rounded-xl border border-slate-200 px-4 py-2 text-sm hover:bg-slate-50" onClick={() => void runAction(account.id, "test")}>健康检查</button>
                <button className="rounded-xl border border-slate-200 px-4 py-2 text-sm hover:bg-slate-50" onClick={() => void runAction(account.id, "refresh")}>刷新</button>
                <button className="rounded-xl border border-slate-200 px-4 py-2 text-sm hover:bg-slate-50" onClick={() => void runAction(account.id, "recover")}>恢复</button>
                <button className="rounded-xl border border-slate-200 px-4 py-2 text-sm hover:bg-slate-50" onClick={() => void runAction(account.id, "clear-cooldown")}>解除冷却</button>
              </div>
            </div>
          </article>
        ))}
      </div>
    </div>
  );
}
