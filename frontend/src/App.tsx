import { BrowserRouter, HashRouter, Navigate, Routes, Route } from "react-router-dom";
import { QueryClient, QueryClientProvider, useQuery } from "@tanstack/react-query";
import { TemplateProvider } from "./themes/TemplateProvider";
import { resolveTemplate } from "./themes/registry";
import { fetchConfig, getStaticPayload } from "./api/client";
import { DashboardList } from "./components/DashboardList";
import { DashboardView } from "./components/DashboardView";
import { Admin } from "./components/Admin";
import { useLiveReload } from "./hooks/useLiveReload";

const staticPayload = getStaticPayload();
const Router = staticPayload ? HashRouter : BrowserRouter;

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 1,
    },
  },
});

function DashboardContent() {
  useLiveReload();

  // In static mode, skip the list and go straight to the baked dashboard.
  const home = staticPayload
    ? <Navigate to={`/d/${encodeURIComponent(staticPayload.dashboard.name)}`} replace />
    : <DashboardList />;

  return (
    <Routes>
      <Route path="/" element={home} />
      <Route path="/d/:name" element={<DashboardView />} />
    </Routes>
  );
}

function AppWithTemplate() {
  const { data: config, isLoading } = useQuery({
    queryKey: ["config"],
    queryFn: fetchConfig,
    staleTime: Infinity,
  });

  if (isLoading || !config) {
    return null;
  }

  const template = resolveTemplate(config.template, config.tokens);

  return (
    <TemplateProvider template={template}>
      <DashboardContent />
    </TemplateProvider>
  );
}

function AppRouter() {
  return (
    <Router>
      <Routes>
        <Route path="/admin" element={<Admin />} />
        <Route path="/*" element={<AppWithTemplate />} />
      </Routes>
    </Router>
  );
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AppRouter />
    </QueryClientProvider>
  );
}
