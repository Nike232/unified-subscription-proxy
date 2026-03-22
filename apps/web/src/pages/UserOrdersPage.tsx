import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { apiFetch } from "../lib/api";
import { formatCurrency, formatDate, mapOrderStatus, mapPaymentStatus } from "../lib/format";
import type { PaymentDetail, UserOrderItem } from "../lib/types";

export default function UserOrdersPage() {
  const [orders, setOrders] = useState<UserOrderItem[]>([]);
  const [payments, setPayments] = useState<PaymentDetail[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    Promise.all([
      apiFetch<UserOrderItem[]>("/api/user/orders"),
      apiFetch<PaymentDetail[]>("/api/user/payments"),
    ])
      .then(([orderData, paymentData]) => {
        setOrders(orderData ?? []);
        setPayments(paymentData ?? []);
      })
      .catch((err: Error) => setError(err.message));
  }, []);

  const safeOrders = Array.isArray(orders) ? orders : [];
  const safePayments = Array.isArray(payments) ? payments : [];
  const paymentByOrder = new Map(safePayments.map((item) => [item.order_id, item]));

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-slate-800">订单与付款</h2>
          <p className="mt-2 text-sm text-slate-500">查看你的订单状态、付款进度，并进入详情页完成确认。</p>
        </div>
        <Link className="rounded-xl bg-blue-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-blue-700" to="/user/catalog">
          去购买套餐
        </Link>
      </div>
      {error ? <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div> : null}
      <div className="overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
        <table className="w-full text-left">
          <thead className="bg-slate-50 text-sm text-slate-500">
            <tr>
              <th className="px-6 py-4 font-medium">订单号</th>
              <th className="px-6 py-4 font-medium">套餐</th>
              <th className="px-6 py-4 font-medium">金额</th>
              <th className="px-6 py-4 font-medium">订单状态</th>
              <th className="px-6 py-4 font-medium">付款状态</th>
              <th className="px-6 py-4 font-medium">创建时间</th>
              <th className="px-6 py-4 font-medium">操作</th>
            </tr>
          </thead>
          <tbody className="text-sm text-slate-700">
            {safeOrders.length === 0 ? (
              <tr>
                <td className="px-6 py-10 text-center text-slate-400" colSpan={7}>暂无订单，先去选一个套餐吧。</td>
              </tr>
            ) : safeOrders.map((order) => {
              const payment = paymentByOrder.get(order.id);
              return (
                <tr key={order.id} className="border-t border-slate-100">
                  <td className="px-6 py-4 font-mono text-xs">{order.id}</td>
                  <td className="px-6 py-4">{order.package_id}</td>
                  <td className="px-6 py-4">{formatCurrency(order.amount_cents, order.currency)}</td>
                  <td className="px-6 py-4">{mapOrderStatus(order.status)}</td>
                  <td className="px-6 py-4">{mapPaymentStatus(payment?.status)}</td>
                  <td className="px-6 py-4">{formatDate(order.created_at)}</td>
                  <td className="px-6 py-4">
                    <Link className="text-sm font-medium text-blue-600 hover:text-blue-800" to={`/user/orders/${order.id}`}>
                      查看详情
                    </Link>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}
