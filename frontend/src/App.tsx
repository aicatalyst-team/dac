import { BrowserRouter, Routes, Route } from "react-router-dom";
import { QueryClient, QueryClientProvider, useQuery } from "@tanstack/react-query";
import { TemplateProvider } from "./themes/TemplateProvider";
import { resolveTemplate } from "./themes/registry";
import { fetchConfig } from "./api/client";
import { DashboardList } from "./components/DashboardList";
import { DashboardView } from "./components/DashboardView";
import { Admin } from "./components/Admin";
import { useLiveReload } from "./hooks/useLiveReload";

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

  return (
    <Routes>
      <Route path="/" element={<DashboardList />} />
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
    <BrowserRouter>
      <Routes>
        <Route path="/admin" element={<Admin />} />
        <Route path="/*" element={<AppWithTemplate />} />
      </Routes>
    </BrowserRouter>
  );
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AppRouter />
    </QueryClientProvider>
  );
}
