import { Navigate, Routes, Route, useLocation } from "react-router-dom";
import { UIProvider } from "./context/UIContext";
import { ThemeProvider } from "./context/ThemeContext";
import { NotificationProvider } from "./context/NotificationContext";
import { UploadProvider } from "./context/UploadContext";
import { AuthProvider, useAuth } from "./context/AuthContext";
import Layout from "./components/Layout";
import UploadDrawer from "./components/UploadDrawer";
import EnvironmentsPage from "./pages/EnvironmentsPage";
import ProjectsPage from "./pages/ProjectsPage";
import ProjectDetailPage from "./pages/ProjectDetailPage";
import UploadsPage from "./pages/UploadsPage";
import SettingsPage from "./pages/SettingsPage";
import LoginPage from "./pages/LoginPage";
import ProfilePage from "./pages/ProfilePage";
import OverviewPage from "./pages/OverviewPage";

function RequireAuth({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth();
  const location = useLocation();

  if (loading) return null;
  if (!user) return <Navigate to="/login" state={{ from: location }} replace />;
  return <>{children}</>;
}

export default function App() {
  return (
    <ThemeProvider>
      <AuthProvider>
        <UIProvider>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route
              path="/*"
              element={
                <RequireAuth>
                  <NotificationProvider>
                  <UploadProvider>
                    <Layout>
                      <Routes>
                        <Route path="/" element={<Navigate to="/overview" replace />} />
                        <Route path="/environments" element={<EnvironmentsPage />} />
                        <Route path="/overview" element={<OverviewPage />} />
                        <Route
                          path="/environments/:envId"
                          element={<ProjectsPage />}
                        />
                        <Route
                          path="/environments/:envId/projects/:projectId"
                          element={<ProjectDetailPage />}
                        />
                        <Route path="/uploads" element={<UploadsPage />} />
                        <Route path="/settings" element={<SettingsPage />} />
                        <Route path="/profile" element={<ProfilePage />} />
                      </Routes>
                    </Layout>
                    <UploadDrawer />
                  </UploadProvider>
                  </NotificationProvider>
                </RequireAuth>
              }
            />
          </Routes>
        </UIProvider>
      </AuthProvider>
    </ThemeProvider>
  );
}
