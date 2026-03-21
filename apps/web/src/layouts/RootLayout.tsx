import { Outlet, Link } from "react-router-dom";
import { LogOut, Settings, Users, Activity, BarChart2, Key } from "lucide-react";

export default function RootLayout() {
  return (
    <div className="flex h-screen w-full bg-[#f8fafc] text-slate-800">
      <aside className="w-64 bg-slate-900 text-white flex flex-col transition-all">
        <div className="h-16 flex items-center px-6 border-b border-slate-800">
          <h1 className="text-xl font-bold bg-gradient-to-r from-blue-400 to-emerald-400 bg-clip-text text-transparent">
            Unified Proxy
          </h1>
        </div>
        <nav className="flex-1 py-4 px-3 space-y-1">
          <div className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2 px-3">
            Admin
          </div>
          <Link to="/admin" className="flex items-center gap-3 px-3 py-2 rounded-lg text-slate-300 hover:bg-slate-800 hover:text-white transition-colors">
            <Activity className="w-5 h-5" /> Dashboard
          </Link>
          <Link to="/admin/users" className="flex items-center gap-3 px-3 py-2 rounded-lg text-slate-300 hover:bg-slate-800 hover:text-white transition-colors">
            <Users className="w-5 h-5" /> Users & Quotas
          </Link>
          <Link to="/admin/settings" className="flex items-center gap-3 px-3 py-2 rounded-lg text-slate-300 hover:bg-slate-800 hover:text-white transition-colors">
            <Settings className="w-5 h-5" /> Settings
          </Link>
          
          <div className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2 mt-6 px-3">
            User
          </div>
          <Link to="/user" className="flex items-center gap-3 px-3 py-2 rounded-lg text-slate-300 hover:bg-slate-800 hover:text-white transition-colors">
            <BarChart2 className="w-5 h-5" /> Usage
          </Link>
          <Link to="/user/keys" className="flex items-center gap-3 px-3 py-2 rounded-lg text-slate-300 hover:bg-slate-800 hover:text-white transition-colors">
            <Key className="w-5 h-5" /> API Keys
          </Link>
        </nav>
        <div className="p-4 border-t border-slate-800">
          <button className="flex items-center gap-3 px-3 py-2 w-full rounded-lg text-slate-400 hover:bg-red-500/10 hover:text-red-400 transition-colors">
            <LogOut className="w-5 h-5" /> Logout
          </button>
        </div>
      </aside>
      
      <main className="flex-1 flex flex-col h-screen overflow-hidden">
        <header className="h-16 bg-white border-b border-slate-200 flex items-center px-6 shadow-sm z-10">
          <div className="font-medium text-slate-600">Console</div>
        </header>
        <div className="flex-1 overflow-auto p-6">
          <Outlet />
        </div>
      </main>
    </div>
  );
}
