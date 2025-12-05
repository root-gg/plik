import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Alert,
  Box,
  Button,
  Paper,
  Stack,
  Tab,
  Tabs,
  Typography,
  List,
  ListItem,
  ListItemText,
} from '@mui/material';
import { api } from '../api/client';
import { useConfig } from '../context/ConfigContext';
import type { UploadDescription } from '../types';

export const HomePage: React.FC = () => {
  const navigate = useNavigate();
  const { config, user, refreshUser, loading } = useConfig();
  const [uploads, setUploads] = useState<UploadDescription[]>([]);
  const [tokens, setTokens] = useState<{ token: string; comment?: string }[]>([]);
  const [error, setError] = useState<string>('');
  const [display, setDisplay] = useState<'uploads' | 'tokens'>('uploads');

  useEffect(() => {
    if (config && !config.authentication) {
      navigate('/');
    }
  }, [config, navigate]);

  useEffect(() => {
    if (!user) {
      refreshUser();
    }
  }, [user, refreshUser]);

  useEffect(() => {
    if (!loading && config?.authentication && !user) {
      navigate('/login');
    }
  }, [config, loading, user, navigate]);

  useEffect(() => {
    if (!user) return;
    if (display === 'uploads') {
      api
        .getUserUploads()
        .then((result) => setUploads(result.results))
        .catch((err) => setError(err instanceof Error ? err.message : 'Unable to load uploads'));
    } else {
      api
        .getUserTokens()
        .then((result) => setTokens(result.results))
        .catch((err) => setError(err instanceof Error ? err.message : 'Unable to load tokens'));
    }
  }, [user, display]);

  const deleteUpload = async (upload: UploadDescription) => {
    try {
      await api.removeUpload(upload);
      setUploads((prev) => prev.filter((u) => u.id !== upload.id));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to delete upload');
    }
  };

  const deleteAllUploads = async () => {
    try {
      await api.deleteUploads();
      setUploads([]);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to delete uploads');
    }
  };

  const createToken = async () => {
    const comment = prompt('Token comment (optional)') || undefined;
    try {
      await api.createToken(comment);
      const result = await api.getUserTokens();
      setTokens(result.results);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to create token');
    }
  };

  const revokeToken = async (token: string) => {
    try {
      await api.revokeToken(token);
      setTokens((prev) => prev.filter((t) => t.token !== token));
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to revoke token');
    }
  };

  if (!user) return <Paper sx={{ p: 3 }}>Loading user...</Paper>;

  return (
    <Paper sx={{ p: 3 }} elevation={0}>
      <Stack spacing={2}>
        <Tabs value={display} onChange={(_, val) => setDisplay(val)}>
          <Tab label="My uploads" value="uploads" />
          <Tab label="Tokens" value="tokens" />
        </Tabs>

        {error && <Alert severity="error">{error}</Alert>}

        {display === 'uploads' && (
          <Stack spacing={2}>
            <Box display="flex" justifyContent="flex-end">
              <Button variant="outlined" onClick={deleteAllUploads}>
                Delete all uploads
              </Button>
            </Box>
            <List>
              {uploads.map((upload) => (
                <ListItem
                  key={upload.id}
                  secondaryAction={
                    <Stack direction="row" spacing={1}>
                      <Button size="small" onClick={() => navigate(`/?id=${upload.id}`)}>
                        Open
                      </Button>
                      <Button size="small" color="error" onClick={() => deleteUpload(upload)}>
                        Delete
                      </Button>
                    </Stack>
                  }
                >
                  <ListItemText
                    primary={<Typography fontWeight={600}>{upload.id}</Typography>}
                    secondary={upload.files ? `${upload.files.length} file(s)` : undefined}
                  />
                </ListItem>
              ))}
            </List>
          </Stack>
        )}

        {display === 'tokens' && (
          <Stack spacing={2}>
            <Box display="flex" justifyContent="flex-end">
              <Button variant="contained" onClick={createToken}>
                Create token
              </Button>
            </Box>
            <List>
              {tokens.map((token) => (
                <ListItem
                  key={token.token}
                  secondaryAction={
                    <Button color="error" size="small" onClick={() => revokeToken(token.token)}>
                      Revoke
                    </Button>
                  }
                >
                  <ListItemText
                    primary={token.token}
                    secondary={token.comment ? <Typography color="text.secondary">{token.comment}</Typography> : undefined}
                  />
                </ListItem>
              ))}
            </List>
          </Stack>
        )}
      </Stack>
    </Paper>
  );
};
