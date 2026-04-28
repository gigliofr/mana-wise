import React from "react";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import App from "./App";
import SharedAnalysisPage from "./pages/SharedAnalysisPage";

export default function AppRouter() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/share/:token" element={<SharedAnalysisPage />} />
        <Route path="/*" element={<App />} />
      </Routes>
    </BrowserRouter>
  );
}
