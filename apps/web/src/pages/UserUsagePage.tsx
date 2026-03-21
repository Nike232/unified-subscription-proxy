import { useEffect, useState } from "react";
import { apiFetch } from "../lib/api";
import { formatDate, mapUsageStatus } from "../lib/format";
import type { UserUsageLogItem } from "../lib/types";

export default function UserUsagePage() {
  const [logs, setLogs] = useState<UserUsageLogItem[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    apiFetch<UserUsageLogItem[]>("/api/user/usage-logs")
      .then((payload) => setLogs(payload ?? []))
      .catch((err: Error) => setError(err.message));
  }, []);

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-slate-800">最近用量</h2>
        <p className="mt-2 text-sm text-slate-500">查看最近请求的模型、provider、执行状态与错误信息。</p>
      </div>
      {error ? <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div> : null}
      <div className="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
        <table className="w-full text-left">
          <thead className="bg-slate-50 text-sm text-slate-500">
            <tr>
              <th className="px-6 py-4 font-medium">时间</th>
              <th className="px-6 py-4 font-medium">模型别名</th>
              <th className="px-6 py-4 font-medium">Provider</th>
              <th className="px-6 py-4 font-medium">状态</th>
              <th className="px-6 py-4 font-medium">令牌</th>
              <th className="px-6 py-4 font-medium">错误类型</th>
            </tr>
          </thead>
          <tbody className="text-sm text-slate-700">
            {logs.length === 0 ? (
              <tr>
                <td className="px-6 py-10 text-center text-slate-400" colSpan={6}>暂无调用记录。</td>
              </tr>
            ) : logs.map((item) => (
              <tr key={item.id} className="border-t border-slate-100">
                <td className="px-6 py-4">{formatDate(item.created_at)}</td>
                <td className="px-6 py-4">{item.model_alias}</td>
                <td className="px-6 py-4">{item.provider}</td>
                <td className="px-6 py-4">{mapUsageStatus(item.status)}</td>
                <td className="px-6 py-4">{item.total_tokens ?? 0}</td>
                <td className="px-6 py-4">{item.error_type || "-"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
