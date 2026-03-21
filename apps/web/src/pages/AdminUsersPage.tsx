import { useEffect, useState } from "react";
import { apiFetch } from "../lib/api";
import type { AdminUserItem } from "../lib/types";

const emptyPatch = {
  email: "",
  name: "",
  password: "",
  role: "",
  status: "",
  balance: "",
  concurrency: "",
};

export default function AdminUsersPage() {
  const [users, setUsers] = useState<AdminUserItem[]>([]);
  const [selected, setSelected] = useState<AdminUserItem | null>(null);
  const [patch, setPatch] = useState(emptyPatch);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  const load = async () => {
    const payload = await apiFetch<AdminUserItem[]>("/api/admin/users");
    setUsers(payload ?? []);
  };

  useEffect(() => {
    load().catch((err: Error) => setError(err.message));
  }, []);

  const choose = (user: AdminUserItem) => {
    setSelected(user);
    setPatch({
      email: user.email ?? "",
      name: user.name ?? "",
      password: "",
      role: user.role ?? "",
      status: user.status ?? "",
      balance: String(user.balance ?? 0),
      concurrency: String(user.concurrency ?? 0),
    });
    setMessage("");
    setError("");
  };

  const submit = async () => {
    if (!selected) return;
    setMessage("");
    setError("");
    try {
      await apiFetch(`/api/admin/users/${selected.id}`, {
        method: "PATCH",
        body: JSON.stringify({
          email: patch.email,
          name: patch.name,
          password: patch.password || undefined,
          role: patch.role,
          status: patch.status,
          balance: Number(patch.balance || 0),
          concurrency: Number(patch.concurrency || 0),
        }),
      });
      await load();
      setMessage("用户信息已更新。");
    } catch (err) {
      setError(err instanceof Error ? err.message : "更新用户失败。");
    }
  };

  return (
    <div className="grid gap-6 xl:grid-cols-[1.1fr_0.9fr]">
      <section className="rounded-2xl border border-slate-200 bg-white shadow-sm">
        <div className="border-b border-slate-100 px-6 py-4">
          <h2 className="text-xl font-semibold text-slate-800">用户管理</h2>
          <p className="mt-1 text-sm text-slate-500">查看用户基础信息，并修改角色、状态、账号和并发配置。</p>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full text-left">
            <thead className="bg-slate-50 text-sm text-slate-500">
              <tr>
                <th className="px-6 py-4 font-medium">用户</th>
                <th className="px-6 py-4 font-medium">角色</th>
                <th className="px-6 py-4 font-medium">状态</th>
                <th className="px-6 py-4 font-medium">余额</th>
                <th className="px-6 py-4 font-medium">并发</th>
              </tr>
            </thead>
            <tbody className="text-sm text-slate-700">
              {users.map((user) => (
                <tr
                  key={user.id}
                  className={`cursor-pointer border-t border-slate-100 ${selected?.id === user.id ? "bg-blue-50" : "hover:bg-slate-50"}`}
                  onClick={() => choose(user)}
                >
                  <td className="px-6 py-4">
                    <div className="font-medium text-slate-800">{user.name || user.email}</div>
                    <div className="text-xs text-slate-500">{user.email}</div>
                    <div className="mt-1 font-mono text-[11px] text-slate-400">{user.id}</div>
                  </td>
                  <td className="px-6 py-4">{user.role}</td>
                  <td className="px-6 py-4">{user.status || "未设置"}</td>
                  <td className="px-6 py-4">${(user.balance ?? 0).toFixed(2)}</td>
                  <td className="px-6 py-4">{user.concurrency ?? 0}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
        <h3 className="text-lg font-semibold text-slate-800">编辑用户</h3>
        {!selected ? (
          <p className="mt-4 text-sm text-slate-500">从左侧表格选择一个用户开始编辑。</p>
        ) : (
          <div className="mt-4 space-y-4">
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-slate-600">邮箱 / 账号</span>
              <input className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3" value={patch.email} onChange={(e) => setPatch((prev) => ({ ...prev, email: e.target.value }))} />
            </label>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-slate-600">名称</span>
              <input className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3" value={patch.name} onChange={(e) => setPatch((prev) => ({ ...prev, name: e.target.value }))} />
            </label>
            <label className="block">
              <span className="mb-2 block text-sm font-medium text-slate-600">新密码</span>
              <input className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3" type="text" value={patch.password} onChange={(e) => setPatch((prev) => ({ ...prev, password: e.target.value }))} placeholder="留空则不修改" />
            </label>
            <div className="grid gap-4 md:grid-cols-2">
              <label className="block">
                <span className="mb-2 block text-sm font-medium text-slate-600">角色</span>
                <select className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3" value={patch.role} onChange={(e) => setPatch((prev) => ({ ...prev, role: e.target.value }))}>
                  <option value="user">user</option>
                  <option value="admin">admin</option>
                </select>
              </label>
              <label className="block">
                <span className="mb-2 block text-sm font-medium text-slate-600">状态</span>
                <select className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3" value={patch.status} onChange={(e) => setPatch((prev) => ({ ...prev, status: e.target.value }))}>
                  <option value="">未设置</option>
                  <option value="active">active</option>
                  <option value="disabled">disabled</option>
                </select>
              </label>
            </div>
            <div className="grid gap-4 md:grid-cols-2">
              <label className="block">
                <span className="mb-2 block text-sm font-medium text-slate-600">余额</span>
                <input className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3" type="number" value={patch.balance} onChange={(e) => setPatch((prev) => ({ ...prev, balance: e.target.value }))} />
              </label>
              <label className="block">
                <span className="mb-2 block text-sm font-medium text-slate-600">并发</span>
                <input className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3" type="number" value={patch.concurrency} onChange={(e) => setPatch((prev) => ({ ...prev, concurrency: e.target.value }))} />
              </label>
            </div>
            {message ? <div className="rounded-xl border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-700">{message}</div> : null}
            {error ? <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div> : null}
            <button className="w-full rounded-xl bg-blue-600 px-4 py-3 text-sm font-semibold text-white transition hover:bg-blue-700" onClick={() => void submit()}>
              保存修改
            </button>
          </div>
        )}
      </section>
    </div>
  );
}
