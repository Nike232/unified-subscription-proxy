import { useEffect, useMemo, useState } from "react";
import { apiFetch } from "../lib/api";
import { formatDate, mapUsageStatus } from "../lib/format";
import type { AdminUsageLogItem } from "../lib/types";

export default function AdminUsagePage() {
  const [logs, setLogs] = useState<AdminUsageLogItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [statusFilter, setStatusFilter] = useState("");
  const [providerFilter, setProviderFilter] = useState("");

  useEffect(() => {
    apiFetch<AdminUsageLogItem[]>("/api/admin/usage-logs")
      .then((payload) => {
        setLogs(payload ?? []);
        setLoading(false);
      })
      .catch((err: Error) => {
        setError(err.message);
        setLoading(false);
      });
  }, []);

  const safeLogs = Array.isArray(logs) ? logs : [];

  const filtered = useMemo(() => safeLogs.filter((item) => {
    if (statusFilter && item.status !== statusFilter) return false;
    if (providerFilter && item.provider !== providerFilter) return false;
    return true;
  }), [safeLogs, providerFilter, statusFilter]);

  const providers = Array.from(new Set(safeLogs.map((item) => item.provider).filter(Boolean)));

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-slate-800">用量记录</h2>
        <p className="mt-2 text-sm text-slate-500">查看最近模型调用、错误类型、账号和 API Key 使用情况。</p>
      </div>
      {error ? <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div> : null}
      <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-slate-600">状态筛选</span>
            <select className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3" value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)}>
              <option value="">全部状态</option>
              <option value="completed">成功</option>
              <option value="failed">失败</option>
            </select>
          </label>
          <label className="block">
            <span className="mb-2 block text-sm font-medium text-slate-600">Provider 筛选</span>
            <select className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3" value={providerFilter} onChange={(e) => setProviderFilter(e.target.value)}>
              <option value="">全部 Provider</option>
              {providers.map((item) => <option key={item} value={item}>{item}</option>)}
            </select>
          </label>
        </div>
      </section>
      <section className="rounded-2xl border border-slate-200 bg-white shadow-sm">
        <div className="border-b border-slate-100 px-6 py-4">
          <h3 className="text-lg font-semibold text-slate-800">最近调用</h3>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-left">
            <thead className="bg-slate-50 text-sm text-slate-500">
              <tr>
                <th className="px-6 py-4 font-medium">时间</th>
                <th className="px-6 py-4 font-medium">模型</th>
                <th className="px-6 py-4 font-medium">Provider</th>
                <th className="px-6 py-4 font-medium">用户</th>
                <th className="px-6 py-4 font-medium">状态</th>
                <th className="px-6 py-4 font-medium">错误</th>
              </tr>
            </thead>
            <tbody className="text-sm text-slate-700">
              {loading ? (
                <tr><td colSpan={6} className="px-6 py-8 text-center text-slate-500">正在加载用量记录...</td></tr>
              ) : filtered.length === 0 ? (
                <tr><td colSpan={6} className="px-6 py-8 text-center text-slate-500">当前筛选条件下暂无用量记录。</td></tr>
              ) : (
                filtered.map((log) => (
                  <tr key={log.id} className="border-t border-slate-100">
                    <td className="px-6 py-4">{formatDate(log.created_at)}</td>
                    <td className="px-6 py-4">{log.model_alias}</td>
                    <td className="px-6 py-4">{log.provider || "-"}</td>
                    <td className="px-6 py-4 font-mono text-xs">{log.user_id || "-"}</td>
                    <td className="px-6 py-4">{mapUsageStatus(log.status)}</td>
                    <td className="px-6 py-4 text-xs text-slate-500">{log.error_type || "无"}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}
