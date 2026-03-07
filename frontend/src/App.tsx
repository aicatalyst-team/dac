import { BrowserRouter, Routes, Route } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ThemeProvider } from "./themes/ThemeProvider";
import { bruinLight } from "./themes/bruin";
import { DashboardList } from "./components/DashboardList";
import { DashboardView } from "./components/DashboardView";
import { useLiveReload } from "./hooks/useLiveReload";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 1,
    },
  },
});

function AppContent() {
  useLiveReload();

  return (
    <Routes>
      <Route path="/" element={<DashboardList />} />
      <Route path="/d/:name" element={<DashboardView />} />
    </Routes>
  );
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider theme={bruinLight}>
        <BrowserRouter>
          <AppContent />
        </BrowserRouter>
      </ThemeProvider>
    </QueryClientProvider>
  );
}
