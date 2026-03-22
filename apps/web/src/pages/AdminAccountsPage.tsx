import { useEffect, useState } from "react";
import { apiFetch } from "../lib/api";
import { formatDate } from "../lib/format";
import type { AdminUpstreamAccountItem } from "../lib/types";

const emptyEditor = {
  display_name: "",
  email: "",
  status: "active",
  auth_mode: "",
  priority: "10",
  weight: "1",
  supports_models: "",
  base_url: "",
  api_key: "",
  access_token: "",
  refresh_token: "",
  expires_at: "",
};

export default function AdminAccountsPage() {
  const [accounts, setAccounts] = useState<AdminUpstreamAccountItem[]>([]);
  const [selected, setSelected] = useState<AdminUpstreamAccountItem | null>(null);
  const [editor, setEditor] = useState(emptyEditor);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [saving, setSaving] = useState(false);

  const load = async () => {
    const payload = await apiFetch<AdminUpstreamAccountItem[]>("/api/admin/upstream-accounts");
    setAccounts(payload ?? []);
  };

  useEffect(() => {
    load().catch((err: Error) => setError(err.message));
  }, []);

  const choose = (account: AdminUpstreamAccountItem) => {
    setSelected(account);
    setEditor({
      display_name: account.display_name || "",
      email: account.email || "",
      status: account.status || "active",
      auth_mode: account.auth_mode || "",
      priority: String(account.priority ?? 10),
      weight: String(account.weight ?? 1),
      supports_models: (account.supports_models ?? []).join(", "),
      base_url: account.meta?.base_url || "",
      api_key: account.meta?.api_key || "",
      access_token: account.meta?.access_token || "",
      refresh_token: account.meta?.refresh_token || "",
      expires_at: account.meta?.expires_at || "",
    });
    setError("");
    setMessage("");
  };

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

  const metaValue = (account: AdminUpstreamAccountItem, key: string) => account.meta?.[key]?.trim() || "";

  const saveAccount = async () => {
    if (!selected) return;
    setSaving(true);
    setError("");
    setMessage("");
    try {
      const updated = await apiFetch<AdminUpstreamAccountItem>(`/api/admin/upstream-accounts/${selected.id}`, {
        method: "PATCH",
        body: JSON.stringify({
          display_name: editor.display_name,
          email: editor.email,
          status: editor.status,
          auth_mode: editor.auth_mode,
          priority: Number(editor.priority || 0),
          weight: Number(editor.weight || 0),
          supports_models: editor.supports_models.split(",").map((item) => item.trim()).filter(Boolean),
          meta: {
            ...selected.meta,
            base_url: editor.base_url,
            api_key: editor.api_key,
            access_token: editor.access_token,
            refresh_token: editor.refresh_token,
            expires_at: editor.expires_at,
          },
        }),
      });
      await load();
      setSelected(updated);
      choose(updated);
      setMessage(`账号 ${selected.id} 已保存。`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "保存账号失败。");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-slate-800">Provider 账号池</h2>
        <p className="mt-2 text-sm text-slate-500">管理上游 provider 账号的凭据、状态、模型支持范围和健康操作。</p>
      </div>
      {message ? <div className="rounded-xl border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-700">{message}</div> : null}
      {error ? <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div> : null}
      <div className="grid gap-6 xl:grid-cols-[1.15fr_0.85fr]">
        <div className="grid gap-4">
          {accounts.map((account) => (
            <article
              key={account.id}
              className={`cursor-pointer rounded-2xl border bg-white p-6 shadow-sm transition ${
                selected?.id === account.id ? "border-blue-400 ring-2 ring-blue-100" : "border-slate-200 hover:border-slate-300"
              }`}
              onClick={() => choose(account)}
            >
              <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                <div>
                  <div className="flex flex-wrap items-center gap-3">
                    <h3 className="text-lg font-semibold text-slate-800">{account.display_name}</h3>
                    <span className="rounded-full bg-slate-100 px-3 py-1 text-xs font-medium text-slate-700">{account.provider}</span>
                    <span className={`rounded-full px-3 py-1 text-xs font-medium ${account.status === "active" ? "bg-emerald-100 text-emerald-700" : "bg-amber-100 text-amber-700"}`}>
                      {account.status}
                    </span>
                  </div>
                  <div className="mt-3 space-y-1 text-sm text-slate-500">
                    <p>账号邮箱：{account.email || "未设置"}</p>
                    <p>鉴权方式：{account.auth_mode || "未设置"}</p>
                    <p>支持模型：{account.supports_models?.join(", ") || "未设置"}</p>
                    <p>最近刷新：{formatDate(account.last_refreshed_at)}</p>
                    <p>最近失败：{metaValue(account, "last_failure_reason") || "无"}</p>
                    <p>冷却到：{formatDate(metaValue(account, "cooldown_until"))}</p>
                    <p>失败次数：{metaValue(account, "consecutive_failures") || "0"}</p>
                    <p>健康检查：{metaValue(account, "last_healthcheck_error") || "最近检查正常"}</p>
                  </div>
                </div>
                <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-4" onClick={(e) => e.stopPropagation()}>
                  <button className="rounded-xl border border-slate-200 px-4 py-2 text-sm hover:bg-slate-50" onClick={() => void runAction(account.id, "test")}>健康检查</button>
                  <button className="rounded-xl border border-slate-200 px-4 py-2 text-sm hover:bg-slate-50" onClick={() => void runAction(account.id, "refresh")}>刷新</button>
                  <button className="rounded-xl border border-slate-200 px-4 py-2 text-sm hover:bg-slate-50" onClick={() => void runAction(account.id, "recover")}>恢复</button>
                  <button className="rounded-xl border border-slate-200 px-4 py-2 text-sm hover:bg-slate-50" onClick={() => void runAction(account.id, "clear-cooldown")}>解除冷却</button>
                </div>
              </div>
            </article>
          ))}
        </div>
        <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
          <h3 className="text-lg font-semibold text-slate-800">编辑账号</h3>
          {!selected ? (
            <p className="mt-4 text-sm text-slate-500">从左侧选择一个 provider 账号，即可编辑凭据和运行参数。</p>
          ) : (
            <div className="mt-4 space-y-4">
              <div className="grid gap-4 md:grid-cols-2">
                <label className="block">
                  <span className="mb-2 block text-sm font-medium text-slate-600">显示名称</span>
                  <input className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3" value={editor.display_name} onChange={(e) => setEditor((prev) => ({ ...prev, display_name: e.target.value }))} />
                </label>
                <label className="block">
                  <span className="mb-2 block text-sm font-medium text-slate-600">账号邮箱</span>
                  <input className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3" value={editor.email} onChange={(e) => setEditor((prev) => ({ ...prev, email: e.target.value }))} />
                </label>
              </div>
              <div className="grid gap-4 md:grid-cols-2">
                <label className="block">
                  <span className="mb-2 block text-sm font-medium text-slate-600">状态</span>
                  <select className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3" value={editor.status} onChange={(e) => setEditor((prev) => ({ ...prev, status: e.target.value }))}>
                    <option value="active">active</option>
                    <option value="invalid">invalid</option>
                    <option value="disabled">disabled</option>
                  </select>
                </label>
                <label className="block">
                  <span className="mb-2 block text-sm font-medium text-slate-600">鉴权方式</span>
                  <input className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3" value={editor.auth_mode} onChange={(e) => setEditor((prev) => ({ ...prev, auth_mode: e.target.value }))} />
                </label>
              </div>
              <div className="grid gap-4 md:grid-cols-2">
                <label className="block">
                  <span className="mb-2 block text-sm font-medium text-slate-600">优先级</span>
                  <input className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3" type="number" value={editor.priority} onChange={(e) => setEditor((prev) => ({ ...prev, priority: e.target.value }))} />
                </label>
                <label className="block">
                  <span className="mb-2 block text-sm font-medium text-slate-600">权重</span>
                  <input className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3" type="number" value={editor.weight} onChange={(e) => setEditor((prev) => ({ ...prev, weight: e.target.value }))} />
                </label>
              </div>
              <label className="block">
                <span className="mb-2 block text-sm font-medium text-slate-600">支持模型</span>
                <input className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3" value={editor.supports_models} onChange={(e) => setEditor((prev) => ({ ...prev, supports_models: e.target.value }))} placeholder="逗号分隔，例如 gpt-fast, gpt-reasoning" />
              </label>
              <label className="block">
                <span className="mb-2 block text-sm font-medium text-slate-600">Base URL</span>
                <input className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3" value={editor.base_url} onChange={(e) => setEditor((prev) => ({ ...prev, base_url: e.target.value }))} />
              </label>
              <label className="block">
                <span className="mb-2 block text-sm font-medium text-slate-600">API Key</span>
                <textarea className="min-h-20 w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 font-mono text-xs" value={editor.api_key} onChange={(e) => setEditor((prev) => ({ ...prev, api_key: e.target.value }))} />
              </label>
              <label className="block">
                <span className="mb-2 block text-sm font-medium text-slate-600">Access Token</span>
                <textarea className="min-h-20 w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 font-mono text-xs" value={editor.access_token} onChange={(e) => setEditor((prev) => ({ ...prev, access_token: e.target.value }))} />
              </label>
              <label className="block">
                <span className="mb-2 block text-sm font-medium text-slate-600">Refresh Token</span>
                <textarea className="min-h-20 w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 font-mono text-xs" value={editor.refresh_token} onChange={(e) => setEditor((prev) => ({ ...prev, refresh_token: e.target.value }))} />
              </label>
              <label className="block">
                <span className="mb-2 block text-sm font-medium text-slate-600">过期时间</span>
                <input className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3" value={editor.expires_at} onChange={(e) => setEditor((prev) => ({ ...prev, expires_at: e.target.value }))} placeholder="RFC3339 时间，例如 2026-03-22T12:00:00Z" />
              </label>
              <button className="w-full rounded-xl bg-blue-600 px-4 py-3 text-sm font-semibold text-white transition hover:bg-blue-700 disabled:cursor-not-allowed disabled:bg-blue-300" disabled={saving} onClick={() => void saveAccount()}>
                {saving ? "保存中..." : "保存账号配置"}
              </button>
            </div>
          )}
        </section>
      </div>
    </div>
  );
}
