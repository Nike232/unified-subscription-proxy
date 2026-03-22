import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { apiFetch } from "../lib/api";

interface User {
  id: string;
  email: string;
  role: string;
  status: string;
  balance: number;
  group: string;
  total_quota: number;
  rpm: number;
  tpm: number;
}

interface KernelStatus {
  mode: string;
  primary: string;
  kernels: Record<string, {
    origin: string;
    healthy: boolean;
    configured: boolean;
    role: string;
    error?: string;
  }>;
}

export default function AdminDashboard() {
  const [users, setUsers] = useState<User[]>([]);
  const [kernelStatus, setKernelStatus] = useState<KernelStatus | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.all([
      apiFetch<User[]>("/api/admin/users"),
      apiFetch<KernelStatus>("/api/admin/kernel-status"),
    ])
      .then(([userData, kernelData]) => {
        setUsers(userData || []);
        setKernelStatus(kernelData);
        setLoading(false);
      })
      .catch((err: Error) => {
        console.error(err);
        setError(err.message);
        setLoading(false);
      });
  }, []);

  const safeUsers = Array.isArray(users) ? users : [];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold text-slate-800">管理后台总览</h2>
      </div>
      
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="p-6 bg-white rounded-xl shadow-sm border border-slate-100 flex flex-col items-center justify-center min-h-[140px]">
          <div className="text-3xl font-bold text-slate-700">{safeUsers.length}</div>
          <p className="text-sm text-slate-500 mt-2 font-medium">注册用户数</p>
        </div>
        <div className="p-6 bg-white rounded-xl shadow-sm border border-slate-100 flex flex-col items-center justify-center min-h-[140px]">
          <div className="text-3xl font-bold text-slate-700">{kernelStatus ? Object.keys(kernelStatus.kernels).length : "-"}</div>
          <p className="text-sm text-slate-500 mt-2 font-medium">代理内核数</p>
        </div>
        <div className="p-6 bg-white rounded-xl shadow-sm border border-slate-100 flex flex-col items-center justify-center min-h-[140px]">
          <div className="text-3xl font-bold text-emerald-500">{kernelStatus?.primary ?? "-"}</div>
          <p className="text-sm text-slate-500 mt-2 font-medium">当前主内核</p>
        </div>
      </div>

      <div className="bg-white rounded-xl shadow-sm border border-slate-100 overflow-hidden">
        <div className="px-6 py-4 border-b border-slate-100">
          <h3 className="font-semibold text-slate-800">内核运行状态</h3>
          <p className="text-sm text-slate-500 mt-1">运行模式：{kernelStatus?.mode ?? "未获取"}</p>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 p-6">
          {kernelStatus ? Object.entries(kernelStatus.kernels).map(([name, kernel]) => (
            <article key={name} className="rounded-xl border border-slate-100 p-4 bg-slate-50">
              <div className="flex items-center justify-between">
                <h4 className="font-semibold text-slate-800">{name}</h4>
                <span className={`px-2 py-1 rounded-full text-xs font-medium ${kernel.healthy ? 'bg-emerald-100 text-emerald-700' : 'bg-amber-100 text-amber-700'}`}>
                  {kernel.healthy ? '健康' : '异常'}
                </span>
              </div>
              <p className="text-sm text-slate-500 mt-2">角色：{kernel.role}</p>
              <p className="text-sm text-slate-500">地址：{kernel.origin || '未配置'}</p>
              {kernel.error ? <p className="text-sm text-red-500 mt-2">{kernel.error}</p> : null}
            </article>
          )) : <div className="text-slate-500">正在加载内核状态...</div>}
        </div>
      </div>

      {error ? (
        <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
          {error}
        </div>
      ) : null}
      
      <div className="bg-white rounded-xl shadow-sm border border-slate-100 overflow-hidden">
        <div className="px-6 py-4 border-b border-slate-100">
          <h3 className="font-semibold text-slate-800">用户管理</h3>
        </div>
        <div className="p-0 overflow-x-auto">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="bg-slate-50 text-slate-500 text-sm">
                <th className="px-6 py-3 font-medium">用户 ID</th>
                <th className="px-6 py-3 font-medium">邮箱</th>
                <th className="px-6 py-3 font-medium">分组</th>
                <th className="px-6 py-3 font-medium">余额与配额</th>
                <th className="px-6 py-3 font-medium">速率限制</th>
                <th className="px-6 py-3 font-medium">状态</th>
                <th className="px-6 py-3 font-medium w-24">操作</th>
              </tr>
            </thead>
            <tbody className="text-sm text-slate-700">
              {loading ? (
                <tr><td colSpan={7} className="px-6 py-8 text-center text-slate-500">正在加载用户...</td></tr>
              ) : safeUsers.length === 0 ? (
                <tr><td colSpan={7} className="px-6 py-8 text-center text-slate-500">暂无用户数据。</td></tr>
              ) : (
                safeUsers.map(u => (
                  <tr key={u.id} className="border-t border-slate-100 hover:bg-slate-50/50">
                    <td className="px-6 py-3 font-mono text-xs">{u.id}</td>
                    <td className="px-6 py-3">{u.email}</td>
                    <td className="px-6 py-3">
                      <span className="px-2 py-1 rounded bg-indigo-50 text-indigo-700 text-xs font-semibold">{u.group || '默认'}</span>
                    </td>
                    <td className="px-6 py-3">
                      <div className="font-semibold text-emerald-600">${(u.balance ?? 0).toFixed(2)}</div>
                      <div className="text-xs text-slate-400 font-medium">配额：{u.total_quota ?? 0}</div>
                    </td>
                    <td className="px-6 py-3">
                      <div className="text-xs text-slate-600 font-medium">RPM：{u.rpm > 0 ? u.rpm : '∞'}</div>
                      <div className="text-xs text-slate-600 font-medium">TPM：{u.tpm > 0 ? u.tpm : '∞'}</div>
                    </td>
                    <td className="px-6 py-3">
                      <span className={`px-2 py-1 rounded-full text-xs font-medium ${u.status === 'active' ? 'bg-emerald-100 text-emerald-700' : 'bg-red-100 text-red-700'}`}>
                        {u.status === "active" ? "正常" : u.status || "未知"}
                      </span>
                    </td>
                    <td className="px-6 py-3">
                      <Link className="text-blue-500 hover:text-blue-700 text-xs font-medium" to="/admin/users">
                        前往编辑
                      </Link>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
