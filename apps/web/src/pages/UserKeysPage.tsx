import { useEffect, useMemo, useState } from "react";
import { apiFetch } from "../lib/api";
import { formatDate, mapKeyStatus, mapSubscriptionStatus } from "../lib/format";
import type { UserAPIKeyItem, UserSubscriptionItem } from "../lib/types";

export default function UserKeysPage() {
  const [keys, setKeys] = useState<UserAPIKeyItem[]>([]);
  const [subscriptions, setSubscriptions] = useState<UserSubscriptionItem[]>([]);
  const [selectedPackage, setSelectedPackage] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  const load = async () => {
    const [keyData, subscriptionData] = await Promise.all([
      apiFetch<UserAPIKeyItem[]>("/api/user/api-keys"),
      apiFetch<UserSubscriptionItem[]>("/api/user/subscriptions"),
    ]);
    setKeys(keyData ?? []);
    setSubscriptions(subscriptionData ?? []);
  };

  useEffect(() => {
    load().catch((err: Error) => setError(err.message));
  }, []);

  const safeKeys = Array.isArray(keys) ? keys : [];
  const safeSubscriptions = Array.isArray(subscriptions) ? subscriptions : [];
  const activeSubscriptions = useMemo(
    () => safeSubscriptions.filter((item) => item.status === "active"),
    [safeSubscriptions],
  );

  useEffect(() => {
    if (!selectedPackage && activeSubscriptions[0]?.package_id) {
      setSelectedPackage(activeSubscriptions[0].package_id);
    }
  }, [activeSubscriptions, selectedPackage]);

  const createKey = async () => {
    if (!selectedPackage) {
      setError("请先选择一个已生效的套餐。");
      return;
    }
    setSubmitting(true);
    setError("");
    try {
      await apiFetch("/api/user/api-keys", {
        method: "POST",
        body: JSON.stringify({ package_id: selectedPackage }),
      });
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "创建 API 密钥失败。");
    } finally {
      setSubmitting(false);
    }
  };

  const revokeKey = async (keyId: string) => {
    setSubmitting(true);
    setError("");
    try {
      await apiFetch(`/api/user/api-keys/${keyId}/revoke`, {
        method: "POST",
        body: JSON.stringify({}),
      });
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "吊销 API 密钥失败。");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold text-slate-800">API 密钥管理</h2>
        <p className="mt-2 text-sm text-slate-500">为已生效的套餐创建密钥，或吊销不再使用的密钥。</p>
      </div>
      {error ? <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div> : null}

      <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <h3 className="text-lg font-semibold text-slate-800">创建新密钥</h3>
        <div className="mt-4 grid gap-4 md:grid-cols-[1fr_auto]">
          <select
            className="rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm outline-none transition focus:border-blue-500 focus:bg-white"
            value={selectedPackage}
            onChange={(event) => setSelectedPackage(event.target.value)}
          >
            <option value="">请选择已生效套餐</option>
            {activeSubscriptions.map((item) => (
              <option key={item.id} value={item.package_id}>
                {item.package_id} · {mapSubscriptionStatus(item.status)}
              </option>
            ))}
          </select>
          <button
            className="rounded-xl bg-blue-600 px-5 py-3 text-sm font-semibold text-white transition hover:bg-blue-700 disabled:cursor-not-allowed disabled:bg-slate-300"
            onClick={createKey}
            disabled={submitting || !selectedPackage}
          >
            {submitting ? "处理中..." : "创建 API 密钥"}
          </button>
        </div>
      </section>

      <div className="grid gap-4">
        {safeKeys.length === 0 ? (
          <div className="rounded-2xl border border-slate-200 bg-white p-8 text-sm text-slate-400 shadow-sm">暂无 API 密钥。</div>
        ) : safeKeys.map((item) => (
          <article key={item.id} className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
            <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
              <div>
                <div className="text-sm text-slate-500">密钥 ID</div>
                <div className="mt-1 font-mono text-sm text-slate-800">{item.id}</div>
                <div className="mt-4 text-sm text-slate-500">密钥值</div>
                <div className="mt-1 rounded-lg bg-slate-950 px-4 py-3 font-mono text-xs text-emerald-300">{item.key}</div>
              </div>
              <div className="space-y-2 text-sm text-slate-500 md:text-right">
                <p>关联套餐：{item.package_id}</p>
                <p>状态：{mapKeyStatus(item.status)}</p>
                <p>创建时间：{formatDate(item.created_at)}</p>
                <p>最近使用：{formatDate(item.last_used_at)}</p>
              </div>
            </div>
            <div className="mt-5 flex justify-end">
              <button
                className="rounded-xl border border-red-200 px-4 py-2 text-sm font-medium text-red-600 transition hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-50"
                onClick={() => revokeKey(item.id)}
                disabled={submitting || item.status !== "active"}
              >
                吊销密钥
              </button>
            </div>
          </article>
        ))}
      </div>
    </div>
  );
}
