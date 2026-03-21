import { useEffect, useState } from "react";
import { apiFetch } from "../lib/api";
import { formatCurrency, formatDate, mapKeyStatus, mapSubscriptionStatus } from "../lib/format";
import type { AdminAPIKeyItem, AdminPackageItem, AdminSubscriptionItem } from "../lib/types";

export default function AdminPackagesPage() {
  const [packages, setPackages] = useState<AdminPackageItem[]>([]);
  const [subscriptions, setSubscriptions] = useState<AdminSubscriptionItem[]>([]);
  const [keys, setKeys] = useState<AdminAPIKeyItem[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    Promise.all([
      apiFetch<AdminPackageItem[]>("/api/admin/packages"),
      apiFetch<AdminSubscriptionItem[]>("/api/admin/subscriptions"),
      apiFetch<AdminAPIKeyItem[]>("/api/admin/api-keys"),
    ])
      .then(([packageData, subscriptionData, keyData]) => {
        setPackages(packageData ?? []);
        setSubscriptions(subscriptionData ?? []);
        setKeys(keyData ?? []);
      })
      .catch((err: Error) => setError(err.message));
  }, []);

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-slate-800">套餐、订阅与密钥</h2>
        <p className="mt-2 text-sm text-slate-500">快速查看当前平台套餐、用户订阅以及已签发的 API 密钥。</p>
      </div>
      {error ? <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div> : null}

      <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <h3 className="text-lg font-semibold text-slate-800">套餐列表</h3>
        <div className="mt-4 grid gap-4 lg:grid-cols-3">
          {packages.map((pkg) => (
            <article key={pkg.id} className="rounded-xl border border-slate-100 bg-slate-50 p-4">
              <div className="flex items-center justify-between gap-4">
                <div>
                  <h4 className="font-semibold text-slate-800">{pkg.display_name || pkg.name}</h4>
                  <p className="mt-1 text-sm text-slate-500">{pkg.description}</p>
                </div>
                <span className={`rounded-full px-3 py-1 text-xs font-medium ${pkg.is_active ? "bg-emerald-100 text-emerald-700" : "bg-slate-200 text-slate-600"}`}>
                  {pkg.is_active ? "启用中" : "已停用"}
                </span>
              </div>
              <div className="mt-4 text-sm text-slate-600">
                <p>价格：{formatCurrency(pkg.price_cents)}</p>
                <p className="mt-1">周期：{pkg.billing_cycle === "yearly" ? "年付" : "月付"}</p>
                <p className="mt-1">模型：{pkg.provider_access.flatMap((item) => item.models).join(", ") || "未配置"}</p>
              </div>
            </article>
          ))}
        </div>
      </section>

      <div className="grid gap-6 xl:grid-cols-2">
        <section className="rounded-2xl border border-slate-200 bg-white shadow-sm">
          <div className="border-b border-slate-100 px-6 py-4">
            <h3 className="text-lg font-semibold text-slate-800">订阅列表</h3>
          </div>
          <div className="max-h-[480px] overflow-auto">
            <table className="w-full text-left">
              <thead className="bg-slate-50 text-sm text-slate-500">
                <tr>
                  <th className="px-6 py-4 font-medium">用户</th>
                  <th className="px-6 py-4 font-medium">套餐</th>
                  <th className="px-6 py-4 font-medium">状态</th>
                  <th className="px-6 py-4 font-medium">到期时间</th>
                </tr>
              </thead>
              <tbody className="text-sm text-slate-700">
                {subscriptions.map((item) => (
                  <tr key={item.id} className="border-t border-slate-100">
                    <td className="px-6 py-4 font-mono text-xs">{item.user_id}</td>
                    <td className="px-6 py-4">{item.package_id}</td>
                    <td className="px-6 py-4">{mapSubscriptionStatus(item.status)}</td>
                    <td className="px-6 py-4">{formatDate(item.expires_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>

        <section className="rounded-2xl border border-slate-200 bg-white shadow-sm">
          <div className="border-b border-slate-100 px-6 py-4">
            <h3 className="text-lg font-semibold text-slate-800">API 密钥</h3>
          </div>
          <div className="max-h-[480px] overflow-auto">
            <table className="w-full text-left">
              <thead className="bg-slate-50 text-sm text-slate-500">
                <tr>
                  <th className="px-6 py-4 font-medium">密钥</th>
                  <th className="px-6 py-4 font-medium">用户</th>
                  <th className="px-6 py-4 font-medium">套餐</th>
                  <th className="px-6 py-4 font-medium">状态</th>
                </tr>
              </thead>
              <tbody className="text-sm text-slate-700">
                {keys.map((item) => (
                  <tr key={item.id} className="border-t border-slate-100">
                    <td className="px-6 py-4">
                      <div className="font-mono text-xs text-slate-500">{item.id}</div>
                      <div className="mt-1 font-mono text-[11px] text-slate-400">{item.key}</div>
                    </td>
                    <td className="px-6 py-4 font-mono text-xs">{item.user_id}</td>
                    <td className="px-6 py-4">{item.package_id}</td>
                    <td className="px-6 py-4">{mapKeyStatus(item.status)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      </div>
    </div>
  );
}
