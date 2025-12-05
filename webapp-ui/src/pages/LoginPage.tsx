import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Alert, Button, Paper, Stack, TextField, Typography } from '@mui/material';
import { api } from '../api/client';
import { useConfig } from '../context/ConfigContext';

export const LoginPage: React.FC = () => {
  const navigate = useNavigate();
  const { config, user, refreshUser, featureEnabled } = useConfig();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string>('');

  useEffect(() => {
    if (config && !config.authentication) {
      navigate('/');
    }
  }, [config, navigate]);

  useEffect(() => {
    if (user) {
      navigate('/home');
    }
  }, [user, navigate]);

  const handleLogin = async () => {
    try {
      await api.login('local', username, password);
      await refreshUser();
      navigate('/home');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to login');
    }
  };

  const handleProvider = async (provider: 'google' | 'ovh') => {
    try {
      const url = (await api.login(provider)) as string;
      window.location.replace(url);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to login');
    }
  };

  return (
    <Paper sx={{ p: 3, maxWidth: 520 }} elevation={0}>
      <Stack spacing={2}>
        <Typography variant="h5">Authentication</Typography>
        {error && <Alert severity="error">{error}</Alert>}
        <TextField label="Login" value={username} onChange={(e) => setUsername(e.target.value)} fullWidth />
        <TextField
          label="Password"
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          fullWidth
        />
        <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1}>
          <Button variant="contained" onClick={handleLogin}>
            Login
          </Button>
          <Button variant="outlined" onClick={() => navigate('/')}>Cancel</Button>
        </Stack>
        <Stack spacing={1}>
          {featureEnabled('authentication') && config?.googleAuthentication && (
            <Button variant="outlined" onClick={() => handleProvider('google')}>
              Login with Google
            </Button>
          )}
          {featureEnabled('authentication') && config?.ovhAuthentication && (
            <Button variant="outlined" onClick={() => handleProvider('ovh')}>
              Login with OVH
            </Button>
          )}
        </Stack>
      </Stack>
    </Paper>
  );
};
