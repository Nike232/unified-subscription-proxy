import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { apiFetch } from "../lib/api";
import { formatCurrency, mapSubscriptionStatus } from "../lib/format";
import type { UserCatalogItem, UserOrderDetail } from "../lib/types";

export default function UserCatalogPage() {
  const navigate = useNavigate();
  const [packages, setPackages] = useState<UserCatalogItem[]>([]);
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState<string | null>(null);

  useEffect(() => {
    apiFetch<{ packages: UserCatalogItem[] }>("/api/user/catalog")
      .then((payload) => setPackages(payload.packages ?? []))
      .catch((err: Error) => setError(err.message));
  }, []);

  const createOrder = async (packageId: string) => {
    setSubmitting(packageId);
    setError("");
    try {
      const checkout = await apiFetch<UserOrderDetail & { order: { id: string } }>("/api/user/orders", {
        method: "POST",
        body: JSON.stringify({
          package_id: packageId,
          create_api_key: true,
          auto_renew: false,
        }),
      });
      navigate(`/user/orders/${checkout.order.id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "下单失败，请稍后重试。");
    } finally {
      setSubmitting(null);
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-slate-800">套餐中心</h2>
        <p className="mt-2 text-sm text-slate-500">选择适合你的套餐，完成付款确认后即可自动生效并生成可用的 API 密钥。</p>
      </div>

      {error ? <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div> : null}

      <div className="grid gap-6 lg:grid-cols-3">
        {packages.map((pkg) => (
          <article key={pkg.id} className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
            <div className="flex items-start justify-between gap-4">
              <div>
                <h3 className="text-xl font-semibold text-slate-900">{pkg.display_name || pkg.name}</h3>
                <p className="mt-2 text-sm text-slate-500">{pkg.description}</p>
              </div>
              <span className="rounded-full bg-slate-100 px-3 py-1 text-xs font-medium text-slate-600">{pkg.tier}</span>
            </div>
            <div className="mt-6">
              <div className="text-3xl font-bold text-slate-900">{formatCurrency(pkg.price_cents)}</div>
              <p className="mt-1 text-sm text-slate-500">计费周期：{pkg.billing_cycle === "yearly" ? "年付" : "月付"}</p>
            </div>
            <div className="mt-6 rounded-xl bg-slate-50 p-4">
              <p className="text-sm font-medium text-slate-700">可用模型</p>
              <div className="mt-3 flex flex-wrap gap-2">
                {pkg.provider_access.flatMap((item) => item.models).map((model) => (
                  <span key={model} className="rounded-full bg-white px-3 py-1 text-xs text-slate-600 ring-1 ring-slate-200">
                    {model}
                  </span>
                ))}
              </div>
            </div>
            <div className="mt-6 flex items-center justify-between">
              <span className={`rounded-full px-3 py-1 text-xs font-medium ${
                pkg.is_subscribed ? "bg-emerald-100 text-emerald-700" : pkg.user_status === "expired" ? "bg-amber-100 text-amber-700" : "bg-slate-100 text-slate-600"
              }`}>
                {pkg.is_subscribed ? "当前已订阅" : pkg.user_status ? mapSubscriptionStatus(pkg.user_status) : "可购买"}
              </span>
              <button
                className="rounded-xl bg-blue-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-blue-700 disabled:cursor-not-allowed disabled:bg-slate-300"
                onClick={() => createOrder(pkg.id)}
                disabled={submitting === pkg.id}
              >
                {submitting === pkg.id ? "创建中..." : "立即购买"}
              </button>
            </div>
          </article>
        ))}
      </div>
    </div>
  );
}
