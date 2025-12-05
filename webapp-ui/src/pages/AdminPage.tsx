import React, { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Paper, Typography } from '@mui/material';
import { useConfig } from '../context/ConfigContext';

export const AdminPage: React.FC = () => {
  const { user } = useConfig();
  const navigate = useNavigate();

  useEffect(() => {
    if (!user?.admin) {
      navigate('/');
    }
  }, [user, navigate]);

  if (!user?.admin) return null;

  return (
    <Paper sx={{ p: 3 }} elevation={0}>
      <Typography variant="h5" gutterBottom>
        Admin
      </Typography>
      <Typography color="text.secondary">
        Admin access is restricted to administrators. Use the classic admin endpoints to manage server data.
      </Typography>
    </Paper>
  );
};
