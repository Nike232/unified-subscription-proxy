import { useEffect, useState } from "react";
import { apiFetch } from "../lib/api";

interface UserProfile {
  user: {
    email: string;
    role: string;
  };
  subscriptions: Array<{ id: string; package_id: string; status: string; expires_at: string }>;
  api_keys: Array<{ id: string; status: string; last_used_at?: string }>;
}

export default function UserDashboard() {
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    apiFetch<UserProfile>("/api/user/profile")
      .then(setProfile)
      .catch((err: Error) => setError(err.message));
  }, []);

  const subscriptionCount = profile?.subscriptions.length ?? 0;
  const activeKeyCount = profile?.api_keys.filter((item) => item.status === "active").length ?? 0;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold text-slate-800">Usage Overview</h2>
        <button className="px-4 py-2 bg-blue-500 text-white rounded-lg text-sm font-medium hover:bg-blue-600 transition-colors shadow-sm">
          Generate API Key
        </button>
      </div>
      
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <div className="p-6 bg-white rounded-xl shadow-sm border border-slate-100">
          <p className="text-sm text-slate-500 font-medium mb-1">Subscriptions</p>
          <div className="text-2xl font-bold text-slate-700">{subscriptionCount}</div>
        </div>
        <div className="p-6 bg-white rounded-xl shadow-sm border border-slate-100">
          <p className="text-sm text-slate-500 font-medium mb-1">User</p>
          <div className="text-2xl font-bold text-slate-700">{profile?.user.email ?? "-"}</div>
        </div>
        <div className="p-6 bg-white rounded-xl shadow-sm border border-slate-100">
          <p className="text-sm text-slate-500 font-medium mb-1">Role</p>
          <div className="text-2xl font-bold text-emerald-500">{profile?.user.role ?? "-"}</div>
        </div>
        <div className="p-6 bg-white rounded-xl shadow-sm border border-slate-100">
          <p className="text-sm text-slate-500 font-medium mb-1">Active Keys</p>
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
          <h3 className="text-lg font-semibold text-slate-800 mb-4">Subscriptions</h3>
          <div className="space-y-3">
            {profile?.subscriptions.map((item) => (
              <div key={item.id} className="rounded-lg border border-slate-100 p-4">
                <div className="font-medium text-slate-800">{item.package_id}</div>
                <div className="text-sm text-slate-500">status: {item.status}</div>
                <div className="text-sm text-slate-500">expires: {new Date(item.expires_at).toLocaleString()}</div>
              </div>
            )) ?? <p className="text-slate-400">No subscriptions found.</p>}
          </div>
        </div>
        <div className="bg-white rounded-xl shadow-sm border border-slate-100 p-6 min-h-[300px]">
          <h3 className="text-lg font-semibold text-slate-800 mb-4">API Keys</h3>
          <div className="space-y-3">
            {profile?.api_keys.map((item) => (
              <div key={item.id} className="rounded-lg border border-slate-100 p-4">
                <div className="font-medium text-slate-800">{item.id}</div>
                <div className="text-sm text-slate-500">status: {item.status}</div>
                <div className="text-sm text-slate-500">last used: {item.last_used_at ? new Date(item.last_used_at).toLocaleString() : "n/a"}</div>
              </div>
            )) ?? <p className="text-slate-400">No API keys found.</p>}
          </div>
        </div>
      </div>
    </div>
  );
}
