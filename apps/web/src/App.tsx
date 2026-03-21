import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import RootLayout from './layouts/RootLayout';
import AdminDashboard from './pages/AdminDashboard';
import UserDashboard from './pages/UserDashboard';

function App() {
  return (
    <Router>
      <Routes>
        <Route path="/" element={<RootLayout />}>
          <Route index element={<Navigate to="/user" replace />} />
          <Route path="admin" element={<AdminDashboard />} />
          <Route path="user" element={<UserDashboard />} />
        </Route>
      </Routes>
    </Router>
  );
}

export default App;
