import { useEffect, useState } from "react";
import { apiFetch } from "../lib/api";
import { formatCurrency } from "../lib/format";
import type { AdminPackageItem } from "../lib/types";

export default function AdminPackagesPage() {
  const [packages, setPackages] = useState<AdminPackageItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    apiFetch<AdminPackageItem[]>("/api/admin/packages")
      .then((payload) => {
        setPackages(payload ?? []);
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
        <h2 className="text-2xl font-bold text-slate-800">套餐管理</h2>
        <p className="mt-2 text-sm text-slate-500">查看当前平台套餐、价格、周期和可访问模型范围。</p>
      </div>
      {error ? <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div> : null}

      <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <h3 className="text-lg font-semibold text-slate-800">套餐列表</h3>
        <div className="mt-4 grid gap-4 lg:grid-cols-3">
          {loading ? (
            <div className="rounded-xl border border-slate-100 bg-slate-50 px-4 py-6 text-sm text-slate-500">正在加载套餐数据...</div>
          ) : packages.length === 0 ? (
            <div className="rounded-xl border border-dashed border-slate-200 bg-slate-50 px-4 py-6 text-sm text-slate-500">当前还没有可展示的套餐。</div>
          ) : (
            packages.map((pkg) => (
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
                  <p className="mt-1">Provider：{pkg.provider_access.map((item) => item.provider).join("、") || "未配置"}</p>
                  <p className="mt-1">模型：{pkg.provider_access.flatMap((item) => item.models).join("、") || "未配置"}</p>
                </div>
              </article>
            ))
          )}
        </div>
      </section>
    </div>
  );
}
