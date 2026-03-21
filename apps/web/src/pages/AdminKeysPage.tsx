import { useEffect, useState } from "react";
import { apiFetch } from "../lib/api";
import { formatDate, mapKeyStatus } from "../lib/format";
import type { AdminAPIKeyItem } from "../lib/types";

export default function AdminKeysPage() {
  const [keys, setKeys] = useState<AdminAPIKeyItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    apiFetch<AdminAPIKeyItem[]>("/api/admin/api-keys")
      .then((payload) => {
        setKeys(payload ?? []);
        setLoading(false);
      })
      .catch((err: Error) => {
        setError(err.message);
        setLoading(false);
      });
  }, []);

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-slate-800">API 密钥管理</h2>
        <p className="mt-2 text-sm text-slate-500">查看所有已签发密钥、所属用户、套餐与最近使用时间。</p>
      </div>
      {error ? <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div> : null}
      <section className="rounded-2xl border border-slate-200 bg-white shadow-sm">
        <div className="border-b border-slate-100 px-6 py-4">
          <h3 className="text-lg font-semibold text-slate-800">密钥列表</h3>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-left">
            <thead className="bg-slate-50 text-sm text-slate-500">
              <tr>
                <th className="px-6 py-4 font-medium">密钥</th>
                <th className="px-6 py-4 font-medium">用户</th>
                <th className="px-6 py-4 font-medium">套餐</th>
                <th className="px-6 py-4 font-medium">状态</th>
                <th className="px-6 py-4 font-medium">最近使用</th>
              </tr>
            </thead>
            <tbody className="text-sm text-slate-700">
              {loading ? (
                <tr><td colSpan={5} className="px-6 py-8 text-center text-slate-500">正在加载密钥...</td></tr>
              ) : keys.length === 0 ? (
                <tr><td colSpan={5} className="px-6 py-8 text-center text-slate-500">暂无 API 密钥。</td></tr>
              ) : (
                keys.map((item) => (
                  <tr key={item.id} className="border-t border-slate-100">
                    <td className="px-6 py-4">
                      <div className="font-mono text-xs text-slate-500">{item.id}</div>
                      <div className="mt-1 font-mono text-[11px] text-slate-400">{item.key}</div>
                    </td>
                    <td className="px-6 py-4 font-mono text-xs">{item.user_id}</td>
                    <td className="px-6 py-4">{item.package_id}</td>
                    <td className="px-6 py-4">{mapKeyStatus(item.status)}</td>
                    <td className="px-6 py-4">{formatDate(item.last_used_at)}</td>
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
