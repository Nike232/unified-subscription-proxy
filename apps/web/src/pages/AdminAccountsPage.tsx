import { useEffect, useMemo, useState } from "react";
import { apiFetch } from "../lib/api";
import { formatDate } from "../lib/format";
import type { AdminUpstreamAccountItem } from "../lib/types";

const providerOptions = ["openai", "gemini", "claude", "codex", "antigravity"] as const;
const statusOptions = ["active", "invalid", "disabled"] as const;

const emptyEditor = {
  provider: "openai",
  display_name: "",
  email: "",
  status: "active",
  auth_mode: "",
  priority: "10",
  weight: "10",
  supports_models: "",
  base_url: "",
  api_key: "",
  access_token: "",
  refresh_token: "",
  expires_at: "",
};

type EditorState = typeof emptyEditor;

function maskSecret(value: string) {
  if (!value) return "未设置";
  if (value.length <= 8) return "••••••••";
  return `${value.slice(0, 4)}••••••${value.slice(-4)}`;
}

function accountMetaValue(account: AdminUpstreamAccountItem, key: string) {
  return account.meta?.[key]?.trim() || "";
}

function buildEditor(account?: AdminUpstreamAccountItem | null): EditorState {
  if (!account) {
    return { ...emptyEditor };
  }
  return {
    provider: account.provider || "openai",
    display_name: account.display_name || "",
    email: account.email || "",
    status: account.status || "active",
    auth_mode: account.auth_mode || "",
    priority: String(account.priority ?? 10),
    weight: String(account.weight ?? 10),
    supports_models: (account.supports_models ?? []).join(", "),
    base_url: account.meta?.base_url || "",
    api_key: account.meta?.api_key || "",
    access_token: account.meta?.access_token || "",
    refresh_token: account.meta?.refresh_token || "",
    expires_at: account.meta?.expires_at || "",
  };
}

export default function AdminAccountsPage() {
  const [accounts, setAccounts] = useState<AdminUpstreamAccountItem[]>([]);
  const [selected, setSelected] = useState<AdminUpstreamAccountItem | null>(null);
  const [editor, setEditor] = useState<EditorState>(emptyEditor);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [saving, setSaving] = useState(false);
  const [creating, setCreating] = useState(false);
  const [busyAction, setBusyAction] = useState("");
  const [showSecrets, setShowSecrets] = useState({
    api_key: false,
    access_token: false,
    refresh_token: false,
  });
  const [filters, setFilters] = useState({
    query: "",
    provider: "all",
    status: "all",
    runtime: "all",
  });

  const load = async (preferredID?: string) => {
    const payload = (await apiFetch<AdminUpstreamAccountItem[]>("/api/admin/upstream-accounts")) ?? [];
    setAccounts(payload);

    const nextSelectedID = preferredID ?? selected?.id;
    if (creating) {
      return;
    }
    if (!nextSelectedID) {
      return;
    }
    const matched = payload.find((item) => item.id === nextSelectedID) ?? null;
    setSelected(matched);
    setEditor(buildEditor(matched));
  };

  useEffect(() => {
    load().catch((err: Error) => setError(err.message));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const filteredAccounts = useMemo(() => {
    const now = Date.now();
    const query = filters.query.trim().toLowerCase();
    return accounts.filter((account) => {
      if (filters.provider !== "all" && account.provider !== filters.provider) {
        return false;
      }
      if (filters.status !== "all" && account.status !== filters.status) {
        return false;
      }
      if (query) {
        const haystack = `${account.display_name} ${account.email}`.toLowerCase();
        if (!haystack.includes(query)) {
          return false;
        }
      }
      if (filters.runtime === "all") {
        return true;
      }
      const cooldown = accountMetaValue(account, "cooldown_until");
      const expiresAt = accountMetaValue(account, "expires_at");
      const inCooldown = cooldown ? Date.parse(cooldown) > now : false;
      const isExpired = expiresAt ? Date.parse(expiresAt) <= now : false;
      const isExpiringSoon = expiresAt ? Date.parse(expiresAt) > now && Date.parse(expiresAt) <= now + 24 * 60 * 60 * 1000 : false;
      if (filters.runtime === "cooldown") return inCooldown;
      if (filters.runtime === "invalid") return account.status === "invalid";
      if (filters.runtime === "expiring") return isExpiringSoon;
      if (filters.runtime === "expired") return isExpired;
      return true;
    });
  }, [accounts, filters]);

  const choose = (account: AdminUpstreamAccountItem) => {
    setCreating(false);
    setSelected(account);
    setEditor(buildEditor(account));
    setError("");
    setMessage("");
  };

  const startCreate = () => {
    setCreating(true);
    setSelected(null);
    setEditor({ ...emptyEditor });
    setShowSecrets({ api_key: false, access_token: false, refresh_token: false });
    setError("");
    setMessage("");
  };

  const runAction = async (id: string, action: "test" | "refresh" | "recover" | "clear-cooldown") => {
    setBusyAction(`${id}:${action}`);
    setError("");
    setMessage("");
    try {
      await apiFetch(`/api/admin/upstream-accounts/${id}/${action}`, {
        method: "POST",
        body: JSON.stringify({}),
      });
      await load(id);
      setMessage(`账号 ${id} 操作完成：${action}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "操作失败。");
    } finally {
      setBusyAction("");
    }
  };

  const saveAccount = async () => {
    setSaving(true);
    setError("");
    setMessage("");
    const payload = {
      provider: editor.provider,
      display_name: editor.display_name,
      email: editor.email,
      status: editor.status,
      auth_mode: editor.auth_mode,
      priority: Number(editor.priority || 0),
      weight: Number(editor.weight || 0),
      supports_models: editor.supports_models.split(",").map((item) => item.trim()).filter(Boolean),
      meta: {
        ...(selected?.meta ?? {}),
        base_url: editor.base_url,
        api_key: editor.api_key,
        access_token: editor.access_token,
        refresh_token: editor.refresh_token,
        expires_at: editor.expires_at,
      },
    };
    try {
      if (creating) {
        const created = await apiFetch<AdminUpstreamAccountItem>("/api/admin/upstream-accounts", {
          method: "POST",
          body: JSON.stringify(payload),
        });
        setCreating(false);
        await load(created.id);
        setMessage(`账号 ${created.id} 已创建。`);
      } else if (selected) {
        const updated = await apiFetch<AdminUpstreamAccountItem>(`/api/admin/upstream-accounts/${selected.id}`, {
          method: "PATCH",
          body: JSON.stringify(payload),
        });
        await load(updated.id);
        setMessage(`账号 ${updated.id} 已保存。`);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "保存账号失败。");
    } finally {
      setSaving(false);
    }
  };

  const activeAccount = creating ? null : selected;

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <h2 className="text-2xl font-bold text-slate-800">Provider 账号池</h2>
          <p className="mt-2 text-sm text-slate-500">
            管理五类上游账号池：openai、gemini、claude、codex、antigravity。这里负责账号状态、凭据和健康动作。
          </p>
        </div>
        <button className="rounded-xl bg-blue-600 px-4 py-3 text-sm font-semibold text-white transition hover:bg-blue-700" onClick={startCreate}>
          新建账号
        </button>
      </div>

      {message ? <div className="rounded-xl border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-700">{message}</div> : null}
      {error ? <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div> : null}

      <section className="rounded-2xl border border-slate-200 bg-white p-4 shadow-sm">
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-slate-600">搜索账号</span>
            <input
              className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3"
              placeholder="搜索显示名称或邮箱"
              value={filters.query}
              onChange={(e) => setFilters((prev) => ({ ...prev, query: e.target.value }))}
            />
          </label>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-slate-600">Provider</span>
            <select className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3" value={filters.provider} onChange={(e) => setFilters((prev) => ({ ...prev, provider: e.target.value }))}>
              <option value="all">全部</option>
              {providerOptions.map((provider) => (
                <option key={provider} value={provider}>
                  {provider}
                </option>
              ))}
            </select>
          </label>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-slate-600">状态</span>
            <select className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3" value={filters.status} onChange={(e) => setFilters((prev) => ({ ...prev, status: e.target.value }))}>
              <option value="all">全部</option>
              {statusOptions.map((status) => (
                <option key={status} value={status}>
                  {status}
                </option>
              ))}
            </select>
          </label>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-slate-600">运行状态</span>
            <select className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3" value={filters.runtime} onChange={(e) => setFilters((prev) => ({ ...prev, runtime: e.target.value }))}>
              <option value="all">全部</option>
              <option value="cooldown">冷却中</option>
              <option value="invalid">已失效</option>
              <option value="expiring">24 小时内过期</option>
              <option value="expired">已过期</option>
            </select>
          </label>
        </div>
      </section>

      <div className="grid gap-6 xl:grid-cols-[1.15fr_0.85fr]">
        <div className="grid gap-4">
          {filteredAccounts.length === 0 ? (
            <div className="rounded-2xl border border-dashed border-slate-300 bg-slate-50 px-6 py-10 text-sm text-slate-500">当前筛选条件下没有账号，可调整筛选条件或直接新建账号。</div>
          ) : (
            filteredAccounts.map((account) => {
              const actionState = (action: string) => busyAction === `${account.id}:${action}`;
              return (
                <article
                  key={account.id}
                  className={`cursor-pointer rounded-2xl border bg-white p-6 shadow-sm transition ${
                    activeAccount?.id === account.id ? "border-blue-400 ring-2 ring-blue-100" : "border-slate-200 hover:border-slate-300"
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
                      <div className="mt-3 grid gap-1 text-sm text-slate-500">
                        <p>账号邮箱：{account.email || "未设置"}</p>
                        <p>鉴权方式：{account.auth_mode || "未设置"}</p>
                        <p>支持模型：{account.supports_models?.join(", ") || "未设置"}</p>
                        <p>最近刷新：{formatDate(account.last_refreshed_at)}</p>
                        <p>过期时间：{formatDate(accountMetaValue(account, "expires_at"))}</p>
                        <p>最近失败：{accountMetaValue(account, "last_failure_reason") || "无"}</p>
                        <p>冷却到：{formatDate(accountMetaValue(account, "cooldown_until"))}</p>
                        <p>失败次数：{accountMetaValue(account, "consecutive_failures") || "0"}</p>
                        <p>健康检查：{accountMetaValue(account, "last_healthcheck_error") || "最近检查正常"}</p>
                      </div>
                    </div>
                    <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-2" onClick={(e) => e.stopPropagation()}>
                      <button className="rounded-xl border border-slate-200 px-4 py-2 text-sm hover:bg-slate-50 disabled:opacity-50" disabled={actionState("test")} onClick={() => void runAction(account.id, "test")}>
                        {actionState("test") ? "检查中..." : "健康检查"}
                      </button>
                      <button className="rounded-xl border border-slate-200 px-4 py-2 text-sm hover:bg-slate-50 disabled:opacity-50" disabled={actionState("refresh")} onClick={() => void runAction(account.id, "refresh")}>
                        {actionState("refresh") ? "刷新中..." : "刷新"}
                      </button>
                      <button className="rounded-xl border border-slate-200 px-4 py-2 text-sm hover:bg-slate-50 disabled:opacity-50" disabled={actionState("recover")} onClick={() => void runAction(account.id, "recover")}>
                        {actionState("recover") ? "恢复中..." : "恢复"}
                      </button>
                      <button className="rounded-xl border border-slate-200 px-4 py-2 text-sm hover:bg-slate-50 disabled:opacity-50" disabled={actionState("clear-cooldown")} onClick={() => void runAction(account.id, "clear-cooldown")}>
                        {actionState("clear-cooldown") ? "处理中..." : "解除冷却"}
                      </button>
                    </div>
                  </div>
                </article>
              );
            })
          )}
        </div>

        <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
          <div className="flex items-center justify-between gap-3">
            <h3 className="text-lg font-semibold text-slate-800">{creating ? "新建账号" : "编辑账号"}</h3>
            {!creating && selected ? (
              <button className="text-sm font-medium text-slate-500 hover:text-slate-700" onClick={startCreate}>
                切换为新建
              </button>
            ) : null}
          </div>
          {!creating && !selected ? (
            <p className="mt-4 text-sm text-slate-500">从左侧选择一个 provider 账号，即可编辑凭据和运行参数。</p>
          ) : (
            <div className="mt-4 space-y-4">
              <div className="grid gap-4 md:grid-cols-2">
                <label className="block">
                  <span className="mb-2 block text-sm font-medium text-slate-600">Provider</span>
                  <select className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3" value={editor.provider} onChange={(e) => setEditor((prev) => ({ ...prev, provider: e.target.value }))} disabled={!creating}>
                    {providerOptions.map((provider) => (
                      <option key={provider} value={provider}>
                        {provider}
                      </option>
                    ))}
                  </select>
                </label>
                <label className="block">
                  <span className="mb-2 block text-sm font-medium text-slate-600">账号邮箱</span>
                  <input className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3" value={editor.email} onChange={(e) => setEditor((prev) => ({ ...prev, email: e.target.value }))} />
                </label>
              </div>
              <div className="grid gap-4 md:grid-cols-2">
                <label className="block">
                  <span className="mb-2 block text-sm font-medium text-slate-600">显示名称</span>
                  <input className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3" value={editor.display_name} onChange={(e) => setEditor((prev) => ({ ...prev, display_name: e.target.value }))} />
                </label>
                <label className="block">
                  <span className="mb-2 block text-sm font-medium text-slate-600">鉴权方式</span>
                  <input className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3" value={editor.auth_mode} onChange={(e) => setEditor((prev) => ({ ...prev, auth_mode: e.target.value }))} placeholder="api_key / oauth / bearer" />
                </label>
              </div>
              <div className="grid gap-4 md:grid-cols-2">
                <label className="block">
                  <span className="mb-2 block text-sm font-medium text-slate-600">状态</span>
                  <select className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3" value={editor.status} onChange={(e) => setEditor((prev) => ({ ...prev, status: e.target.value }))}>
                    {statusOptions.map((status) => (
                      <option key={status} value={status}>
                        {status}
                      </option>
                    ))}
                  </select>
                </label>
                <label className="block">
                  <span className="mb-2 block text-sm font-medium text-slate-600">支持模型</span>
                  <input className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3" value={editor.supports_models} onChange={(e) => setEditor((prev) => ({ ...prev, supports_models: e.target.value }))} placeholder="逗号分隔，例如 gpt-fast, gpt-reasoning" />
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
                <span className="mb-2 block text-sm font-medium text-slate-600">Base URL</span>
                <input className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3" value={editor.base_url} onChange={(e) => setEditor((prev) => ({ ...prev, base_url: e.target.value }))} />
              </label>

              {(["api_key", "access_token", "refresh_token"] as const).map((key) => (
                <label key={key} className="block">
                  <span className="mb-2 flex items-center justify-between gap-3 text-sm font-medium text-slate-600">
                    <span>
                      {key === "api_key" ? "API Key" : key === "access_token" ? "Access Token" : "Refresh Token"}
                    </span>
                    <button
                      type="button"
                      className="text-xs font-medium text-blue-600 hover:text-blue-700"
                      onClick={() => setShowSecrets((prev) => ({ ...prev, [key]: !prev[key] }))}
                    >
                      {showSecrets[key] ? "隐藏" : "显示"}
                    </button>
                  </span>
                  {showSecrets[key] ? (
                    <textarea
                      className="min-h-20 w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 font-mono text-xs"
                      value={editor[key]}
                      onChange={(e) => setEditor((prev) => ({ ...prev, [key]: e.target.value }))}
                    />
                  ) : (
                    <div className="rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 font-mono text-xs text-slate-500">{maskSecret(editor[key])}</div>
                  )}
                </label>
              ))}

              <label className="block">
                <span className="mb-2 block text-sm font-medium text-slate-600">过期时间</span>
                <input className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3" value={editor.expires_at} onChange={(e) => setEditor((prev) => ({ ...prev, expires_at: e.target.value }))} placeholder="RFC3339 时间，例如 2026-03-22T12:00:00Z" />
              </label>

              {selected ? (
                <div className="rounded-xl bg-slate-50 px-4 py-3 text-xs text-slate-600">
                  <div>最近失败原因：{accountMetaValue(selected, "last_failure_reason") || "无"}</div>
                  <div>最近健康检查：{accountMetaValue(selected, "last_healthcheck_error") || "最近检查正常"}</div>
                  <div>冷却到：{formatDate(accountMetaValue(selected, "cooldown_until"))}</div>
                </div>
              ) : null}

              <button className="w-full rounded-xl bg-blue-600 px-4 py-3 text-sm font-semibold text-white transition hover:bg-blue-700 disabled:cursor-not-allowed disabled:bg-blue-300" disabled={saving} onClick={() => void saveAccount()}>
                {saving ? "保存中..." : creating ? "创建账号" : "保存账号配置"}
              </button>
            </div>
          )}
        </section>
      </div>
    </div>
  );
}
