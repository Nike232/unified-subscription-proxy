import { useEffect, useState } from "react";
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

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold text-slate-800">Admin Dashboard</h2>
      </div>
      
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <div className="p-6 bg-white rounded-xl shadow-sm border border-slate-100 flex flex-col items-center justify-center min-h-[140px]">
          <div className="text-3xl font-bold text-slate-700">{users.length}</div>
          <p className="text-sm text-slate-500 mt-2 font-medium">Registered Users</p>
        </div>
        <div className="p-6 bg-white rounded-xl shadow-sm border border-slate-100 flex flex-col items-center justify-center min-h-[140px]">
          <div className="text-3xl font-bold text-slate-700">{kernelStatus ? Object.keys(kernelStatus.kernels).length : "-"}</div>
          <p className="text-sm text-slate-500 mt-2 font-medium">Proxy Kernels</p>
        </div>
        <div className="p-6 bg-white rounded-xl shadow-sm border border-slate-100 flex flex-col items-center justify-center min-h-[140px]">
          <div className="text-3xl font-bold text-emerald-500">{kernelStatus?.primary ?? "-"}</div>
          <p className="text-sm text-slate-500 mt-2 font-medium">Primary Kernel</p>
        </div>
      </div>

      <div className="bg-white rounded-xl shadow-sm border border-slate-100 overflow-hidden">
        <div className="px-6 py-4 border-b border-slate-100">
          <h3 className="font-semibold text-slate-800">Kernel Runtime</h3>
          <p className="text-sm text-slate-500 mt-1">mode: {kernelStatus?.mode ?? "n/a"}</p>
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 p-6">
          {kernelStatus ? Object.entries(kernelStatus.kernels).map(([name, kernel]) => (
            <article key={name} className="rounded-xl border border-slate-100 p-4 bg-slate-50">
              <div className="flex items-center justify-between">
                <h4 className="font-semibold text-slate-800">{name}</h4>
                <span className={`px-2 py-1 rounded-full text-xs font-medium ${kernel.healthy ? 'bg-emerald-100 text-emerald-700' : 'bg-amber-100 text-amber-700'}`}>
                  {kernel.healthy ? 'healthy' : 'unhealthy'}
                </span>
              </div>
              <p className="text-sm text-slate-500 mt-2">role: {kernel.role}</p>
              <p className="text-sm text-slate-500">origin: {kernel.origin || 'n/a'}</p>
              {kernel.error ? <p className="text-sm text-red-500 mt-2">{kernel.error}</p> : null}
            </article>
          )) : <div className="text-slate-500">Loading kernels...</div>}
        </div>
      </div>

      {error ? (
        <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
          {error}
        </div>
      ) : null}
      
      <div className="bg-white rounded-xl shadow-sm border border-slate-100 overflow-hidden">
        <div className="px-6 py-4 border-b border-slate-100">
          <h3 className="font-semibold text-slate-800">User Management</h3>
        </div>
        <div className="p-0 overflow-x-auto">
          <table className="w-full text-left border-collapse">
            <thead>
              <tr className="bg-slate-50 text-slate-500 text-sm">
                <th className="px-6 py-3 font-medium">User ID</th>
                <th className="px-6 py-3 font-medium">Email</th>
                <th className="px-6 py-3 font-medium">Group</th>
                <th className="px-6 py-3 font-medium">Balance & Quota</th>
                <th className="px-6 py-3 font-medium">Rate Limits</th>
                <th className="px-6 py-3 font-medium">Status</th>
                <th className="px-6 py-3 font-medium w-24">Actions</th>
              </tr>
            </thead>
            <tbody className="text-sm text-slate-700">
              {loading ? (
                <tr><td colSpan={7} className="px-6 py-8 text-center text-slate-500">Loading users...</td></tr>
              ) : users.length === 0 ? (
                <tr><td colSpan={7} className="px-6 py-8 text-center text-slate-500">No users found.</td></tr>
              ) : (
                users.map(u => (
                  <tr key={u.id} className="border-t border-slate-100 hover:bg-slate-50/50">
                    <td className="px-6 py-3 font-mono text-xs">{u.id}</td>
                    <td className="px-6 py-3">{u.email}</td>
                    <td className="px-6 py-3">
                      <span className="px-2 py-1 rounded bg-indigo-50 text-indigo-700 text-xs font-semibold">{u.group || 'default'}</span>
                    </td>
                    <td className="px-6 py-3">
                      <div className="font-semibold text-emerald-600">${(u.balance ?? 0).toFixed(2)}</div>
                      <div className="text-xs text-slate-400 font-medium">Quota: {u.total_quota ?? 0}</div>
                    </td>
                    <td className="px-6 py-3">
                      <div className="text-xs text-slate-600 font-medium">RPM: {u.rpm > 0 ? u.rpm : '∞'}</div>
                      <div className="text-xs text-slate-600 font-medium">TPM: {u.tpm > 0 ? u.tpm : '∞'}</div>
                    </td>
                    <td className="px-6 py-3">
                      <span className={`px-2 py-1 rounded-full text-xs font-medium ${u.status === 'active' ? 'bg-emerald-100 text-emerald-700' : 'bg-red-100 text-red-700'}`}>
                        {u.status}
                      </span>
                    </td>
                    <td className="px-6 py-3">
                      <button className="text-blue-500 hover:text-blue-700 text-xs font-medium">Edit</button>
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
