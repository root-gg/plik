import React, { useEffect, useState } from 'react';
import { Button, Paper, Stack, Typography, Collapse, Divider } from '@mui/material';
import { ExpandLess, ExpandMore } from '@mui/icons-material';
import { api } from '../api/client';
import type { ClientBinary } from '../types';

export const ClientsPage: React.FC = () => {
  const [clients, setClients] = useState<ClientBinary[]>([]);
  const [error, setError] = useState<string>('');

  useEffect(() => {
    api
      .getVersion()
      .then((info) => setClients(info.clients || []))
      .catch((err) => setError(err instanceof Error ? err.message : 'Unable to load clients'));
  }, []);

  if (error) return <Paper sx={{ p: 3 }}>{error}</Paper>;

  return (
    <Paper sx={{ p: 3 }} elevation={0}>
      <Stack spacing={1.5} divider={<Divider flexItem />}>
        {clients.map((client) => (
          <Stack key={client.name} spacing={1}>
            <Stack direction="row" alignItems="center" justifyContent="space-between">
              <Stack direction="row" spacing={1} alignItems="center">
                <Button
                  startIcon={client.showDetails ? <ExpandLess /> : <ExpandMore />}
                  onClick={() =>
                    setClients((prev) =>
                      prev.map((c) => (c.name === client.name ? { ...c, showDetails: !c.showDetails } : c)),
                    )
                  }
                >
                  {client.name}
                </Button>
              </Stack>
              <Button variant="contained" component="a" href={api.base + client.path}>
                Download
              </Button>
            </Stack>
            <Collapse in={client.showDetails}>
              {client.md5 && (
                <Typography variant="body2" color="text.secondary">
                  md5: {client.md5}
                </Typography>
              )}
            </Collapse>
          </Stack>
        ))}
      </Stack>
    </Paper>
  );
};
