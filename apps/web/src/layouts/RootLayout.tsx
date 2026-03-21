import { Outlet, Link, useLocation, useNavigate } from "react-router-dom";
import { LogOut, ShoppingCart, PackageCheck, Users, Activity, BarChart2, Key, WalletCards, Boxes, Bug } from "lucide-react";
import { useAuth } from "../contexts/AuthContext";

export default function RootLayout() {
  const { user, logout } = useAuth();
  const location = useLocation();
  const navigate = useNavigate();

  const handleLogout = async () => {
    await logout();
    navigate("/login", { replace: true });
  };

  const navItemClass = (active: boolean) =>
    `flex items-center gap-3 px-3 py-2 rounded-lg transition-colors ${active ? "bg-slate-800 text-white" : "text-slate-300 hover:bg-slate-800 hover:text-white"}`;

  return (
    <div className="flex h-screen w-full bg-[#f8fafc] text-slate-800">
      <aside className="w-64 bg-slate-900 text-white flex flex-col transition-all">
        <div className="h-16 flex items-center px-6 border-b border-slate-800">
          <h1 className="text-xl font-bold bg-gradient-to-r from-blue-400 to-emerald-400 bg-clip-text text-transparent">
            统一反代平台
          </h1>
        </div>
        <nav className="flex-1 py-4 px-3 space-y-1">
          {user?.role === "admin" ? (
            <>
              <div className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2 px-3">
                管理后台
              </div>
              <Link to="/admin" className={navItemClass(location.pathname === "/admin")}>
                <Users className="w-5 h-5" /> 管理总览
              </Link>
              <Link to="/admin/users" className={navItemClass(location.pathname.startsWith("/admin/users"))}>
                <Users className="w-5 h-5" /> 用户管理
              </Link>
              <Link to="/admin/accounts" className={navItemClass(location.pathname.startsWith("/admin/accounts"))}>
                <Activity className="w-5 h-5" /> 账号池
              </Link>
              <Link to="/admin/packages" className={navItemClass(location.pathname.startsWith("/admin/packages"))}>
                <Boxes className="w-5 h-5" /> 套餐管理
              </Link>
              <Link to="/admin/subscriptions" className={navItemClass(location.pathname.startsWith("/admin/subscriptions"))}>
                <PackageCheck className="w-5 h-5" /> 订阅管理
              </Link>
              <Link to="/admin/keys" className={navItemClass(location.pathname.startsWith("/admin/keys"))}>
                <Key className="w-5 h-5" /> 密钥管理
              </Link>
              <Link to="/admin/usage" className={navItemClass(location.pathname.startsWith("/admin/usage"))}>
                <BarChart2 className="w-5 h-5" /> 用量记录
              </Link>
              <Link to="/admin/debug" className={navItemClass(location.pathname.startsWith("/admin/debug"))}>
                <Bug className="w-5 h-5" /> 调度调试
              </Link>
            </>
          ) : null}

          <div className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-2 mt-6 px-3">
            用户中心
          </div>
          <Link to="/user" className={navItemClass(location.pathname === "/user")}>
            <BarChart2 className="w-5 h-5" /> 首页概览
          </Link>
          <Link to="/user/catalog" className={navItemClass(location.pathname.startsWith("/user/catalog"))}>
            <ShoppingCart className="w-5 h-5" /> 套餐购买
          </Link>
          <Link to="/user/orders" className={navItemClass(location.pathname.startsWith("/user/orders"))}>
            <WalletCards className="w-5 h-5" /> 订单与付款
          </Link>
          <Link to="/user/subscriptions" className={navItemClass(location.pathname.startsWith("/user/subscriptions"))}>
            <PackageCheck className="w-5 h-5" /> 我的订阅
          </Link>
          <Link to="/user/keys" className={navItemClass(location.pathname.startsWith("/user/keys"))}>
            <Key className="w-5 h-5" /> API 密钥
          </Link>
          <Link to="/user/usage" className={navItemClass(location.pathname.startsWith("/user/usage"))}>
            <Activity className="w-5 h-5" /> 最近用量
          </Link>
        </nav>
        <div className="p-4 border-t border-slate-800">
          <div className="mb-4 rounded-xl bg-slate-800/70 px-4 py-3 text-sm">
            <div className="font-medium text-white">{user?.name || user?.email || "当前用户"}</div>
            <div className="mt-1 text-xs text-slate-400">{user?.role === "admin" ? "管理员" : "普通用户"}</div>
          </div>
          <button onClick={() => void handleLogout()} className="flex items-center gap-3 px-3 py-2 w-full rounded-lg text-slate-400 hover:bg-red-500/10 hover:text-red-400 transition-colors">
            <LogOut className="w-5 h-5" /> 退出登录
          </button>
        </div>
      </aside>
      
      <main className="flex-1 flex flex-col h-screen overflow-hidden">
        <header className="h-16 bg-white border-b border-slate-200 flex items-center px-6 shadow-sm z-10">
          <div className="font-medium text-slate-600">控制台</div>
        </header>
        <div className="flex-1 overflow-auto p-6">
          <Outlet />
        </div>
      </main>
    </div>
  );
}
