import React from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import { Layout } from './components/Layout';
import { useConfig } from './context/ConfigContext';
import { AdminPage } from './pages/AdminPage';
import { ClientsPage } from './pages/ClientsPage';
import { HomePage } from './pages/HomePage';
import { LoginPage } from './pages/LoginPage';
import { MainPage } from './pages/MainPage';

export const App: React.FC = () => {
  const { featureEnabled, user } = useConfig();

  const requireAuth = (element: React.ReactNode) => {
    if (featureEnabled('authentication') && !user) {
      return <Navigate to="/login" replace />;
    }
    return element;
  };

  return (
    <Layout>
      <Routes>
        <Route path="/" element={<MainPage />} />
        <Route path="/clients" element={<ClientsPage />} />
        <Route path="/login" element={<LoginPage />} />
        <Route path="/home" element={requireAuth(<HomePage />)} />
        <Route path="/admin" element={requireAuth(<AdminPage />)} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Layout>
  );
};
