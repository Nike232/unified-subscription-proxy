import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { apiFetch } from "../lib/api";
import { formatCurrency, formatDate, mapOrderStatus, mapPaymentStatus, mapSubscriptionStatus } from "../lib/format";
import type { UserOrderDetail } from "../lib/types";

export default function UserOrderDetailPage() {
  const { orderId = "" } = useParams();
  const [detail, setDetail] = useState<UserOrderDetail | null>(null);
  const [error, setError] = useState("");
  const [confirming, setConfirming] = useState(false);

  const load = async () => {
    try {
      const payload = await apiFetch<UserOrderDetail>(`/api/user/orders/${orderId}`);
      setDetail(payload);
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载订单失败。");
    }
  };

  useEffect(() => {
    void load();
  }, [orderId]);

  const confirmPayment = async () => {
    setConfirming(true);
    try {
      await apiFetch(`/api/user/orders/${orderId}/confirm-payment`, {
        method: "POST",
        body: JSON.stringify({}),
      });
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "确认付款失败。");
    } finally {
      setConfirming(false);
    }
  };

  if (!detail) {
    return (
      <div className="space-y-4">
        <Link className="text-sm font-medium text-blue-600 hover:text-blue-800" to="/user/orders">返回订单列表</Link>
        {error ? <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div> : <div className="text-slate-500">正在加载订单详情...</div>}
      </div>
    );
  }

  const { order, payment, package: pkg, subscription } = detail;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <Link className="text-sm font-medium text-blue-600 hover:text-blue-800" to="/user/orders">返回订单列表</Link>
          <h2 className="mt-3 text-2xl font-bold text-slate-800">订单详情</h2>
          <p className="mt-2 text-sm text-slate-500">订单号：{order.id}</p>
        </div>
        <span className={`rounded-full px-4 py-2 text-sm font-medium ${order.status === "paid" ? "bg-emerald-100 text-emerald-700" : "bg-amber-100 text-amber-700"}`}>
          {mapOrderStatus(order.status)}
        </span>
      </div>

      {error ? <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div> : null}

      <div className="grid gap-6 lg:grid-cols-[1.2fr_0.8fr]">
        <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
          <h3 className="text-lg font-semibold text-slate-800">订单信息</h3>
          <div className="mt-4 grid gap-4 md:grid-cols-2">
            <div className="rounded-xl bg-slate-50 p-4">
              <p className="text-sm text-slate-500">套餐</p>
              <p className="mt-1 font-semibold text-slate-800">{pkg?.display_name || order.package_id}</p>
            </div>
            <div className="rounded-xl bg-slate-50 p-4">
              <p className="text-sm text-slate-500">订单金额</p>
              <p className="mt-1 font-semibold text-slate-800">{formatCurrency(order.amount_cents, order.currency)}</p>
            </div>
            <div className="rounded-xl bg-slate-50 p-4">
              <p className="text-sm text-slate-500">创建时间</p>
              <p className="mt-1 font-semibold text-slate-800">{formatDate(order.created_at)}</p>
            </div>
            <div className="rounded-xl bg-slate-50 p-4">
              <p className="text-sm text-slate-500">付款状态</p>
              <p className="mt-1 font-semibold text-slate-800">{mapPaymentStatus(payment?.status)}</p>
            </div>
          </div>

          {subscription ? (
            <div className="mt-6 rounded-2xl border border-emerald-200 bg-emerald-50 p-5">
              <h4 className="font-semibold text-emerald-900">订阅已生效</h4>
              <p className="mt-2 text-sm text-emerald-800">状态：{mapSubscriptionStatus(subscription.status)}</p>
              <p className="mt-1 text-sm text-emerald-800">到期时间：{formatDate(subscription.expires_at)}</p>
              <Link className="mt-4 inline-flex text-sm font-medium text-emerald-700 hover:text-emerald-900" to="/user/keys">
                前往管理 API 密钥
              </Link>
            </div>
          ) : null}
        </section>

        <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
          <h3 className="text-lg font-semibold text-slate-800">付款确认</h3>
          <div className="mt-5 rounded-2xl border border-dashed border-slate-300 bg-slate-50 p-6 text-center">
            <div className="mx-auto flex h-44 w-44 items-center justify-center rounded-2xl bg-white text-center text-sm text-slate-500 ring-1 ring-slate-200">
              付款二维码
              <br />
              内部测试占位
            </div>
            <p className="mt-4 text-sm text-slate-500">当前阶段使用简化支付流程。你可以打开付款链接查看测试页，或直接手动确认付款。</p>
            {payment?.checkout_url ? (
              <a className="mt-4 inline-flex text-sm font-medium text-blue-600 hover:text-blue-800" href={payment.checkout_url} target="_blank" rel="noreferrer">
                打开付款页
              </a>
            ) : null}
          </div>
          <div className="mt-6 rounded-xl bg-slate-50 p-4 text-sm text-slate-600">
            <p>订单金额：{formatCurrency(order.amount_cents, order.currency)}</p>
            <p className="mt-2">订单状态：{mapOrderStatus(order.status)}</p>
            <p className="mt-2">付款状态：{mapPaymentStatus(payment?.status)}</p>
          </div>
          <button
            className="mt-6 w-full rounded-xl bg-emerald-600 px-4 py-3 text-sm font-semibold text-white transition hover:bg-emerald-700 disabled:cursor-not-allowed disabled:bg-emerald-300"
            onClick={confirmPayment}
            disabled={confirming || payment?.status === "paid"}
          >
            {payment?.status === "paid" ? "已完成付款" : confirming ? "确认中..." : "我已付款，提交确认"}
          </button>
        </section>
      </div>
    </div>
  );
}
