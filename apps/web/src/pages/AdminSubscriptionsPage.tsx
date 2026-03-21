import { useEffect, useState } from "react";
import { apiFetch } from "../lib/api";
import { formatDate, mapSubscriptionStatus } from "../lib/format";
import type { AdminSubscriptionItem } from "../lib/types";

export default function AdminSubscriptionsPage() {
  const [subscriptions, setSubscriptions] = useState<AdminSubscriptionItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    apiFetch<AdminSubscriptionItem[]>("/api/admin/subscriptions")
      .then((payload) => {
        setSubscriptions(payload ?? []);
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
        <h2 className="text-2xl font-bold text-slate-800">订阅管理</h2>
        <p className="mt-2 text-sm text-slate-500">查看所有用户订阅的状态、来源订单和到期时间。</p>
      </div>
      {error ? <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div> : null}
      <section className="rounded-2xl border border-slate-200 bg-white shadow-sm">
        <div className="border-b border-slate-100 px-6 py-4">
          <h3 className="text-lg font-semibold text-slate-800">订阅列表</h3>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-left">
            <thead className="bg-slate-50 text-sm text-slate-500">
              <tr>
                <th className="px-6 py-4 font-medium">订阅 ID</th>
                <th className="px-6 py-4 font-medium">用户</th>
                <th className="px-6 py-4 font-medium">套餐</th>
                <th className="px-6 py-4 font-medium">状态</th>
                <th className="px-6 py-4 font-medium">来源订单</th>
                <th className="px-6 py-4 font-medium">到期时间</th>
              </tr>
            </thead>
            <tbody className="text-sm text-slate-700">
              {loading ? (
                <tr><td colSpan={6} className="px-6 py-8 text-center text-slate-500">正在加载订阅...</td></tr>
              ) : subscriptions.length === 0 ? (
                <tr><td colSpan={6} className="px-6 py-8 text-center text-slate-500">暂无订阅记录。</td></tr>
              ) : (
                subscriptions.map((item) => (
                  <tr key={item.id} className="border-t border-slate-100">
                    <td className="px-6 py-4 font-mono text-xs">{item.id}</td>
                    <td className="px-6 py-4 font-mono text-xs">{item.user_id}</td>
                    <td className="px-6 py-4">{item.package_id}</td>
                    <td className="px-6 py-4">{mapSubscriptionStatus(item.status)}</td>
                    <td className="px-6 py-4 font-mono text-xs">{item.order_id || "无"}</td>
                    <td className="px-6 py-4">{formatDate(item.expires_at)}</td>
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
