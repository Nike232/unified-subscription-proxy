import { useEffect, useState } from "react";

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

export default function AdminDashboard() {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch("http://localhost:8080/api/admin/users", {
      headers: { "Authorization": "Bearer admin" } // Mock admin token for testing
    })
      .then(res => res.json())
      .then(data => {
        setUsers(data || []);
        setLoading(false);
      })
      .catch(err => {
        console.error(err);
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
          <div className="text-3xl font-bold text-slate-700">8</div>
          <p className="text-sm text-slate-500 mt-2 font-medium">Upstream Proxies</p>
        </div>
        <div className="p-6 bg-white rounded-xl shadow-sm border border-slate-100 flex flex-col items-center justify-center min-h-[140px]">
          <div className="text-3xl font-bold text-emerald-500">99.9%</div>
          <p className="text-sm text-slate-500 mt-2 font-medium">System Uptime</p>
        </div>
      </div>
      
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
                      <div className="font-semibold text-emerald-600">${u.balance.toFixed(2)}</div>
                      <div className="text-xs text-slate-400 font-medium">Quota: {u.total_quota}</div>
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
