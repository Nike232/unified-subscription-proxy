import { useEffect, useState } from "react";
import { apiFetch } from "../lib/api";
import { formatDate, mapSubscriptionStatus } from "../lib/format";
import type { UserSubscriptionItem } from "../lib/types";

export default function UserSubscriptionsPage() {
  const [subscriptions, setSubscriptions] = useState<UserSubscriptionItem[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    apiFetch<UserSubscriptionItem[]>("/api/user/subscriptions")
      .then((payload) => setSubscriptions(payload ?? []))
      .catch((err: Error) => setError(err.message));
  }, []);

  const safeSubscriptions = Array.isArray(subscriptions) ? subscriptions : [];

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-slate-800">我的订阅</h2>
        <p className="mt-2 text-sm text-slate-500">查看当前生效中的套餐、续费配置和到期时间。</p>
      </div>
      {error ? <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div> : null}
      <div className="grid gap-5 lg:grid-cols-2">
        {safeSubscriptions.length === 0 ? (
          <div className="rounded-2xl border border-slate-200 bg-white p-8 text-sm text-slate-400 shadow-sm">暂无订阅记录。</div>
        ) : safeSubscriptions.map((item) => (
          <article key={item.id} className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
            <div className="flex items-center justify-between gap-4">
              <h3 className="text-lg font-semibold text-slate-800">{item.package_id}</h3>
              <span className={`rounded-full px-3 py-1 text-xs font-medium ${item.status === "active" ? "bg-emerald-100 text-emerald-700" : "bg-amber-100 text-amber-700"}`}>
                {mapSubscriptionStatus(item.status)}
              </span>
            </div>
            <div className="mt-4 space-y-2 text-sm text-slate-500">
              <p>开始时间：{formatDate(item.starts_at)}</p>
              <p>到期时间：{formatDate(item.expires_at)}</p>
              <p>自动续费：{item.auto_renew ? "开启" : "关闭"}</p>
              <p>来源订单：{item.order_id || "暂无"}</p>
            </div>
          </article>
        ))}
      </div>
    </div>
  );
}
