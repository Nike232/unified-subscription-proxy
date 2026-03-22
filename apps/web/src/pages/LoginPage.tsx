import { useState } from "react";
import { Navigate, useLocation, useNavigate } from "react-router-dom";
import { useAuth } from "../contexts/AuthContext";

export default function LoginPage() {
  const { user, login, loading } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  if (!loading && user) {
    return <Navigate to={user.role === "admin" ? "/admin" : "/user"} replace />;
  }

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setSubmitting(true);
    setError("");
    try {
      const nextUser = await login(email, password);
      const nextPath = (location.state as { from?: string } | null)?.from;
      navigate(nextPath || (nextUser.role === "admin" ? "/admin" : "/user"), { replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : "登录失败，请重试。");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="min-h-screen bg-slate-950 text-white">
      <div className="mx-auto flex min-h-screen max-w-6xl items-center px-6 py-16">
        <div className="grid w-full gap-10 lg:grid-cols-[1.2fr_0.9fr]">
          <section className="rounded-3xl border border-white/10 bg-[radial-gradient(circle_at_top,_rgba(59,130,246,0.35),_transparent_45%),linear-gradient(135deg,#0f172a,#111827_45%,#052e16)] p-10 shadow-2xl">
            <p className="mb-4 inline-flex rounded-full border border-emerald-400/30 bg-emerald-400/10 px-4 py-1 text-sm text-emerald-200">
              云钥平台
            </p>
            <h1 className="text-4xl font-bold leading-tight md:text-5xl">一个入口，统一管理套餐、密钥与模型调用。</h1>
            <p className="mt-6 max-w-2xl text-lg text-slate-300">
              登录后即可浏览套餐、完成订单、确认付款、查看订阅状态，并管理自己的 API 密钥与最近用量。
            </p>
            <div className="mt-10 grid gap-4 md:grid-cols-3">
              {[
                ["购买套餐", "套餐、价格、可用模型一目了然"],
                ["确认支付", "支持简化付款确认流程，适合内部测试"],
                ["管理密钥", "创建、吊销、查看最近使用情况"],
              ].map(([title, desc]) => (
                <article key={title} className="rounded-2xl border border-white/10 bg-white/5 p-4">
                  <h2 className="text-base font-semibold text-white">{title}</h2>
                  <p className="mt-2 text-sm text-slate-300">{desc}</p>
                </article>
              ))}
            </div>
          </section>

          <section className="rounded-3xl border border-slate-200 bg-white p-8 text-slate-900 shadow-xl">
            <h2 className="text-2xl font-semibold">登录控制台</h2>
            <p className="mt-2 text-sm text-slate-500">请输入账号与密码进入用户中心或管理后台。</p>
            <form className="mt-8 space-y-5" onSubmit={handleSubmit}>
              <label className="block">
                <span className="mb-2 block text-sm font-medium text-slate-600">账号 / 邮箱</span>
                <input
                  className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 outline-none transition focus:border-blue-500 focus:bg-white"
                  type="text"
                  value={email}
                  onChange={(event) => setEmail(event.target.value)}
                  placeholder="请输入账号或邮箱"
                  required
                />
              </label>
              <label className="block">
                <span className="mb-2 block text-sm font-medium text-slate-600">密码</span>
                <input
                  className="w-full rounded-xl border border-slate-200 bg-slate-50 px-4 py-3 outline-none transition focus:border-blue-500 focus:bg-white"
                  type="password"
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  placeholder="请输入密码"
                  required
                />
              </label>
              {error ? (
                <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">{error}</div>
              ) : null}
              <button
                className="w-full rounded-xl bg-blue-600 px-4 py-3 text-sm font-semibold text-white transition hover:bg-blue-700 disabled:cursor-not-allowed disabled:bg-blue-300"
                type="submit"
                disabled={submitting}
              >
                {submitting ? "登录中..." : "登录"}
              </button>
            </form>
            <div className="mt-6 rounded-2xl border border-slate-200 bg-slate-50 p-4 text-sm text-slate-500">
              <p className="font-medium text-slate-700">说明</p>
              <p className="mt-2">当前环境为内部测试环境，付款流程采用简化确认方式，适合先跑通完整业务链路。</p>
            </div>
          </section>
        </div>
      </div>
    </div>
  );
}
