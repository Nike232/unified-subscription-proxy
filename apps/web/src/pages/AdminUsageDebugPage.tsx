import { useEffect, useState } from "react";
import { apiFetch } from "../lib/api";
import { formatDate, mapUsageStatus } from "../lib/format";
import type { AdminOrderItem, AdminPaymentItem, AdminUsageLogItem, DispatchDebugResult } from "../lib/types";

export default function AdminUsageDebugPage() {
  const [logs, setLogs] = useState<AdminUsageLogItem[]>([]);
  const [orders, setOrders] = useState<AdminOrderItem[]>([]);
  const [payments, setPayments] = useState<AdminPaymentItem[]>([]);
  const [trace, setTrace] = useState<DispatchDebugResult | null>(null);
  const [apiKey, setApiKey] = useState("usp_demo_key");
  const [modelAlias, setModelAlias] = useState("gpt-fast");
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  const load = async () => {
    const [logData, orderData, paymentData] = await Promise.all([
      apiFetch<AdminUsageLogItem[]>("/api/admin/usage-logs"),
      apiFetch<AdminOrderItem[]>("/api/admin/orders"),
      apiFetch<AdminPaymentItem[]>("/api/admin/payments"),
    ]);
    setLogs(logData ?? []);
    setOrders(orderData ?? []);
    setPayments(paymentData ?? []);
  };

  useEffect(() => {
    load().catch((err: Error) => setError(err.message));
  }, []);

  const debugDispatch = async () => {
    setError("");
    try {
      const payload = await apiFetch<DispatchDebugResult>("/api/admin/dispatch-debug", {
        method: "POST",
        body: JSON.stringify({ api_key: apiKey, model_alias: modelAlias }),
      });
      setTrace(payload);
    } catch (err) {
      setError(err instanceof Error ? err.message : "调度调试失败。");
    }
  };

  const confirmOrder = async (orderId: string) => {
    setError("");
    setMessage("");
    try {
      await apiFetch(`/api/admin/orders/${orderId}/confirm-payment`, {
        method: "POST",
        body: JSON.stringify({}),
      });
      await load();
      setMessage(`订单 ${orderId} 已完成手动核销。`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "核销订单失败。");
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-slate-800">用量与调试</h2>
        <p className="mt-2 text-sm text-slate-500">查看最近调用记录、待处理订单与支付，并执行路由调试。</p>
      </div>
      {message ? <div className="rounded-xl border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-700">{message}</div> : null}
      {error ? <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div> : null}

      <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <h3 className="text-lg font-semibold text-slate-800">调度调试</h3>
        <div className="mt-4 grid gap-4 lg:grid-cols-[1fr_1fr_auto]">
          <input className="rounded-xl border border-slate-200 bg-slate-50 px-4 py-3" value={apiKey} onChange={(e) => setApiKey(e.target.value)} placeholder="API Key" />
          <input className="rounded-xl border border-slate-200 bg-slate-50 px-4 py-3" value={modelAlias} onChange={(e) => setModelAlias(e.target.value)} placeholder="模型别名" />
          <button className="rounded-xl bg-blue-600 px-4 py-3 text-sm font-semibold text-white hover:bg-blue-700" onClick={() => void debugDispatch()}>
            调试
          </button>
        </div>
        {trace ? (
          <div className="mt-5 rounded-2xl bg-slate-950 p-5 text-sm text-slate-200">
            <p>模型别名：{trace.model_alias}</p>
            <p className="mt-1">命中 Provider：{trace.selected_provider}</p>
            <p className="mt-1">命中账号：{trace.selected_account_id}</p>
            <p className="mt-1">上游模型：{trace.upstream_model}</p>
            <div className="mt-4 space-y-2">
              {trace.candidates.map((candidate) => (
                <div key={`${candidate.account_id}-${candidate.provider}`} className="rounded-lg border border-white/10 bg-white/5 px-4 py-3">
                  <div>{candidate.provider} / {candidate.account_id}</div>
                  <div className="text-xs text-slate-400">状态：{candidate.account_status || "未知"} / skip: {candidate.skip_reason || "无"}</div>
                </div>
              ))}
            </div>
          </div>
        ) : null}
      </section>

      <div className="grid gap-6 xl:grid-cols-2">
        <section className="rounded-2xl border border-slate-200 bg-white shadow-sm">
          <div className="border-b border-slate-100 px-6 py-4">
            <h3 className="text-lg font-semibold text-slate-800">订单与支付</h3>
          </div>
          <div className="max-h-[480px] overflow-auto">
            <table className="w-full text-left">
              <thead className="bg-slate-50 text-sm text-slate-500">
                <tr>
                  <th className="px-6 py-4 font-medium">订单</th>
                  <th className="px-6 py-4 font-medium">用户</th>
                  <th className="px-6 py-4 font-medium">状态</th>
                  <th className="px-6 py-4 font-medium">操作</th>
                </tr>
              </thead>
              <tbody className="text-sm text-slate-700">
                {orders.map((order) => {
                  const payment = payments.find((item) => item.order_id === order.id);
                  return (
                    <tr key={order.id} className="border-t border-slate-100">
                      <td className="px-6 py-4">
                        <div className="font-mono text-xs">{order.id}</div>
                        <div className="mt-1 text-xs text-slate-500">{order.package_id}</div>
                      </td>
                      <td className="px-6 py-4 font-mono text-xs">{order.user_id}</td>
                      <td className="px-6 py-4">
                        <div>{order.status}</div>
                        <div className="mt-1 text-xs text-slate-500">{payment?.status || "无支付"}</div>
                      </td>
                      <td className="px-6 py-4">
                        <button
                          className="rounded-lg border border-slate-200 px-3 py-2 text-xs hover:bg-slate-50 disabled:opacity-50"
                          disabled={order.status === "paid"}
                          onClick={() => void confirmOrder(order.id)}
                        >
                          手动核销
                        </button>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </section>

        <section className="rounded-2xl border border-slate-200 bg-white shadow-sm">
          <div className="border-b border-slate-100 px-6 py-4">
            <h3 className="text-lg font-semibold text-slate-800">最近用量</h3>
          </div>
          <div className="max-h-[480px] overflow-auto">
            <table className="w-full text-left">
              <thead className="bg-slate-50 text-sm text-slate-500">
                <tr>
                  <th className="px-6 py-4 font-medium">时间</th>
                  <th className="px-6 py-4 font-medium">模型</th>
                  <th className="px-6 py-4 font-medium">用户</th>
                  <th className="px-6 py-4 font-medium">状态</th>
                </tr>
              </thead>
              <tbody className="text-sm text-slate-700">
                {logs.map((log) => (
                  <tr key={log.id} className="border-t border-slate-100">
                    <td className="px-6 py-4">{formatDate(log.created_at)}</td>
                    <td className="px-6 py-4">{log.model_alias}</td>
                    <td className="px-6 py-4 font-mono text-xs">{log.user_id || "-"}</td>
                    <td className="px-6 py-4">{mapUsageStatus(log.status)}</td>
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
