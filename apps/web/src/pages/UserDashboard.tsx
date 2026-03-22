import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { apiFetch } from "../lib/api";
import { formatDate, mapSubscriptionStatus } from "../lib/format";
import type { UserProfile } from "../lib/types";

export default function UserDashboard() {
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    apiFetch<UserProfile>("/api/user/profile")
      .then(setProfile)
      .catch((err: Error) => setError(err.message));
  }, []);

  const subscriptions = Array.isArray(profile?.subscriptions) ? profile.subscriptions : [];
  const apiKeys = Array.isArray(profile?.api_keys) ? profile.api_keys : [];
  const subscriptionCount = subscriptions.length;
  const activeKeyCount = apiKeys.filter((item) => item.status === "active").length;
  const recentSubscription = subscriptions[0];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-slate-800">用户中心</h2>
          <p className="mt-2 text-sm text-slate-500">从这里开始管理套餐、付款、订阅和 API 密钥。</p>
        </div>
        <Link className="px-4 py-2 bg-blue-500 text-white rounded-lg text-sm font-medium hover:bg-blue-600 transition-colors shadow-sm" to="/user/catalog">
          去购买套餐
        </Link>
      </div>
      
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <div className="p-6 bg-white rounded-xl shadow-sm border border-slate-100">
          <p className="text-sm text-slate-500 font-medium mb-1">订阅数量</p>
          <div className="text-2xl font-bold text-slate-700">{subscriptionCount}</div>
        </div>
        <div className="p-6 bg-white rounded-xl shadow-sm border border-slate-100">
          <p className="text-sm text-slate-500 font-medium mb-1">当前用户</p>
          <div className="text-2xl font-bold text-slate-700">{profile?.user.email ?? "-"}</div>
        </div>
        <div className="p-6 bg-white rounded-xl shadow-sm border border-slate-100">
          <p className="text-sm text-slate-500 font-medium mb-1">身份角色</p>
          <div className="text-2xl font-bold text-emerald-500">{profile?.user.role === "admin" ? "管理员" : profile?.user.role === "user" ? "普通用户" : "-"}</div>
        </div>
        <div className="p-6 bg-white rounded-xl shadow-sm border border-slate-100">
          <p className="text-sm text-slate-500 font-medium mb-1">可用密钥</p>
          <div className="text-2xl font-bold text-slate-700">{activeKeyCount}</div>
        </div>
      </div>

      {error ? (
        <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
          {error}
        </div>
      ) : null}
      
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div className="bg-white rounded-xl shadow-sm border border-slate-100 p-6 min-h-[300px]">
          <h3 className="text-lg font-semibold text-slate-800 mb-4">最近订阅</h3>
          {recentSubscription ? (
            <div className="rounded-lg border border-slate-100 p-4">
              <div className="font-medium text-slate-800">{recentSubscription.package_id}</div>
              <div className="mt-2 text-sm text-slate-500">状态：{mapSubscriptionStatus(recentSubscription.status)}</div>
              <div className="text-sm text-slate-500">到期时间：{formatDate(recentSubscription.expires_at)}</div>
              <Link className="mt-4 inline-flex text-sm font-medium text-blue-600 hover:text-blue-800" to="/user/subscriptions">
                查看全部订阅
              </Link>
            </div>
          ) : <p className="text-slate-400">暂无订阅。</p>}
        </div>
        <div className="bg-white rounded-xl shadow-sm border border-slate-100 p-6 min-h-[300px]">
          <h3 className="text-lg font-semibold text-slate-800 mb-4">常用操作</h3>
          <div className="grid gap-3">
            <Link className="rounded-lg border border-slate-200 px-4 py-4 text-sm text-slate-700 transition hover:border-blue-300 hover:bg-blue-50" to="/user/orders">
              查看订单与付款状态
            </Link>
            <Link className="rounded-lg border border-slate-200 px-4 py-4 text-sm text-slate-700 transition hover:border-blue-300 hover:bg-blue-50" to="/user/keys">
              管理 API 密钥
            </Link>
            <Link className="rounded-lg border border-slate-200 px-4 py-4 text-sm text-slate-700 transition hover:border-blue-300 hover:bg-blue-50" to="/user/usage">
              查看最近用量记录
            </Link>
          </div>
        </div>
      </div>
    </div>
  );
}
