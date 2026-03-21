import { BrowserRouter as Router, Navigate, Outlet, Route, Routes, useLocation } from "react-router-dom";
import RootLayout from "./layouts/RootLayout";
import AdminDashboard from "./pages/AdminDashboard";
import UserDashboard from "./pages/UserDashboard";
import LoginPage from "./pages/LoginPage";
import UserCatalogPage from "./pages/UserCatalogPage";
import UserOrdersPage from "./pages/UserOrdersPage";
import UserOrderDetailPage from "./pages/UserOrderDetailPage";
import UserSubscriptionsPage from "./pages/UserSubscriptionsPage";
import UserKeysPage from "./pages/UserKeysPage";
import UserUsagePage from "./pages/UserUsagePage";
import { AuthProvider, useAuth } from "./contexts/AuthContext";

function RequireAuth() {
  const { user, loading } = useAuth();
  const location = useLocation();

  if (loading) {
    return <div className="flex min-h-screen items-center justify-center bg-slate-50 text-slate-500">正在检查登录状态...</div>;
  }

  if (!user) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />;
  }

  return <Outlet />;
}

function RequireAdmin() {
  const { user } = useAuth();
  if (user?.role !== "admin") {
    return <Navigate to="/user" replace />;
  }
  return <Outlet />;
}

function App() {
  return (
    <AuthProvider>
      <Router>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route element={<RequireAuth />}>
            <Route path="/" element={<RootLayout />}>
              <Route index element={<Navigate to="/user" replace />} />
              <Route element={<RequireAdmin />}>
                <Route path="admin" element={<AdminDashboard />} />
              </Route>
              <Route path="user" element={<UserDashboard />} />
              <Route path="user/catalog" element={<UserCatalogPage />} />
              <Route path="user/orders" element={<UserOrdersPage />} />
              <Route path="user/orders/:orderId" element={<UserOrderDetailPage />} />
              <Route path="user/subscriptions" element={<UserSubscriptionsPage />} />
              <Route path="user/keys" element={<UserKeysPage />} />
              <Route path="user/usage" element={<UserUsagePage />} />
            </Route>
          </Route>
        </Routes>
      </Router>
    </AuthProvider>
  );
}

export default App;
