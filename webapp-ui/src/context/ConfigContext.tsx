import React, { createContext, useContext, useEffect, useMemo, useState } from 'react';
import { api } from '../api/client';
import type { FeatureFlagValue, ServerConfig, UserInfo } from '../types';

interface ConfigState {
  config?: ServerConfig;
  user?: UserInfo;
  loading: boolean;
  featureEnabled: (name: string) => boolean;
  featureForced: (name: string) => boolean;
  refreshUser: () => Promise<void>;
  refreshConfig: () => Promise<void>;
  title: string;
}

const staticTitle = 'Plik';
const ConfigContext = createContext<ConfigState | undefined>(undefined);

export const ConfigProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [config, setConfig] = useState<ServerConfig>();
  const [user, setUser] = useState<UserInfo>();
  const [loading, setLoading] = useState(true);

  const featureEnabled = (name: string) => {
    const value = config?.[`feature_${name}` as keyof ServerConfig] as FeatureFlagValue | undefined;
    return Boolean(value && value !== 'disabled');
  };

  const featureForced = (name: string) => {
    const value = config?.[`feature_${name}` as keyof ServerConfig] as FeatureFlagValue | undefined;
    return value === 'forced';
  };

  const refreshConfig = async () => {
    const cfg = await api.getConfig();
    setConfig(cfg);
    if (cfg.title) document.title = cfg.title;
  };

  const refreshUser = async () => {
    try {
      const u = await api.getUser();
      setUser(u);
    } catch (err) {
      setUser(undefined);
    }
  };

  useEffect(() => {
    const bootstrap = async () => {
      try {
        await refreshConfig();
      } finally {
        setLoading(false);
      }
      await refreshUser();
    };
    bootstrap();
  }, []);

  const value = useMemo<ConfigState>(
    () => ({
      config,
      user,
      loading,
      featureEnabled,
      featureForced,
      refreshUser,
      refreshConfig,
      title: config?.title || staticTitle,
    }),
    [config, user, loading]
  );

  return <ConfigContext.Provider value={value}>{children}</ConfigContext.Provider>;
};

export const useConfig = () => {
  const ctx = useContext(ConfigContext);
  if (!ctx) {
    throw new Error('useConfig must be used within ConfigProvider');
  }
  return ctx;
};
