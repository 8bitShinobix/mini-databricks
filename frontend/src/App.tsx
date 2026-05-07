import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import { LoginPage } from "@/pages/LoginPage";
import { JobsPage } from "@/pages/JobsPage";
import { JobDetailPage } from "@/pages/JobDetailPage";

function PrivateRoute({ children }: { children: React.ReactNode }) {
  const token = localStorage.getItem("token");
  return token ? <>{children}</> : <Navigate to="/login" />;
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route
          path="/jobs"
          element={
            <PrivateRoute>
              <JobsPage />
            </PrivateRoute>
          }
        />
        <Route
          path="/jobs/:id"
          element={
            <PrivateRoute>
              <JobDetailPage />
            </PrivateRoute>
          }
        />
        <Route path="*" element={<Navigate to="/jobs" />} />
      </Routes>
    </BrowserRouter>
  );
}
