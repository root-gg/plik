import React, { useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Box,
  Button,
  Chip,
  Divider,
  FormControl,
  FormControlLabel,
  InputLabel,
  List,
  ListItem,
  ListItemText,
  MenuItem,
  Paper,
  Select,
  Stack,
  Switch,
  TextField,
  Tooltip,
  Typography,
  Link as MuiLink,
} from '@mui/material';
import { UploadFile } from '@mui/icons-material';
import { api } from '../api/client';
import { useConfig } from '../context/ConfigContext';
import type { UploadDescription } from '../types';

interface UploadToggleState {
  oneShot: boolean;
  removable: boolean;
  stream: boolean;
  passwordEnabled: boolean;
  extendTtl: boolean;
  enableComments: boolean;
}

const ttlUnits = [
  { key: 'minutes', label: 'minutes', multiplier: 60 },
  { key: 'hours', label: 'hours', multiplier: 3600 },
  { key: 'days', label: 'days', multiplier: 86400 },
  { key: 'unlimited', label: 'unlimited', multiplier: 0 },
];

export const MainPage: React.FC = () => {
  const { config, user, featureEnabled, featureForced } = useConfig();
  const [files, setFiles] = useState<File[]>([]);
  const [toggles, setToggles] = useState<UploadToggleState>({
    oneShot: false,
    removable: false,
    stream: false,
    passwordEnabled: false,
    extendTtl: false,
    enableComments: false,
  });
  const [password, setPassword] = useState('');
  const [comments, setComments] = useState('');
  const [ttlValue, setTtlValue] = useState(1);
  const [ttlUnit, setTtlUnit] = useState(ttlUnits[2]);
  const [status, setStatus] = useState<string>('');
  const [uploadResult, setUploadResult] = useState<UploadDescription | null>(null);

  const maxFileSize = useMemo(() => {
    if (user?.maxFileSize && user.maxFileSize > 0) return user.maxFileSize;
    return config?.maxFileSize;
  }, [user, config]);

  useEffect(() => {
    if (!config) return;
    const defaultToggle = (name: string) => {
      const value = config[`feature_${name}` as keyof typeof config];
      return value === 'default' || value === 'forced';
    };
    setToggles((prev) => ({
      ...prev,
      oneShot: defaultToggle('one_shot'),
      removable: defaultToggle('removable'),
      stream: defaultToggle('stream'),
      passwordEnabled: defaultToggle('password'),
      extendTtl: defaultToggle('extend_ttl'),
      enableComments: defaultToggle('comments'),
    }));
  }, [config]);

  const handleFiles = (selected: FileList | null) => {
    if (!selected) return;
    const next: File[] = [...files];
    Array.from(selected).forEach((file) => {
      if (maxFileSize && file.size > maxFileSize) {
        setStatus(`File ${file.name} is too big`);
        return;
      }
      if (!next.find((f) => f.name === file.name && f.size === file.size)) {
        (file as any).fileName = file.name;
        next.push(file);
      }
    });
    setFiles(next);
  };

  const computeTtl = () => {
    if (ttlUnit.key === 'unlimited') return 0;
    return Math.max(0, Math.floor(ttlValue * ttlUnit.multiplier));
  };

  const toggle = (key: keyof UploadToggleState) => {
    setToggles((prev) => ({ ...prev, [key]: !prev[key] }));
  };

  const resetState = () => {
    setFiles([]);
    setPassword('');
    setComments('');
  };

  const startUpload = async (createEmpty = false) => {
    setStatus('Preparing upload...');
    setUploadResult(null);
    try {
      const payload = {
        oneShot: toggles.oneShot,
        removable: toggles.removable,
        stream: toggles.stream,
        extend_ttl: toggles.extendTtl,
        password: toggles.passwordEnabled ? password : undefined,
        comments: toggles.enableComments ? comments : undefined,
        ttl: computeTtl(),
      };
      const upload = await api.createUpload(payload);
      if (!createEmpty && files.length > 0) {
        for (const file of files) {
          setStatus(`Uploading ${file.name}...`);
          await api.uploadFile(upload, file);
        }
      }
      setUploadResult(upload);
      setStatus('Upload created successfully');
      resetState();
    } catch (err) {
      setStatus(err instanceof Error ? err.message : 'Upload failed');
    }
  };

  const isUploadReady = files.length > 0 || featureEnabled('text');

  const downloadUrl = uploadResult ? `${api.base}/#/?id=${uploadResult.id}` : undefined;

  const featureKey = (key: keyof UploadToggleState) => {
    switch (key) {
      case 'oneShot':
        return 'one_shot';
      case 'removable':
        return 'removable';
      case 'stream':
        return 'stream';
      case 'passwordEnabled':
        return 'password';
      case 'extendTtl':
        return 'extend_ttl';
      case 'enableComments':
        return 'comments';
      default:
        return key;
    }
  };

  const toggleField = (
    label: string,
    key: keyof UploadToggleState,
    description: string,
    disabled?: boolean,
  ) => (
    <Tooltip title={description} placement="right">
      <FormControlLabel
        control={
          <Switch
            checked={toggles[key]}
            onChange={() => toggle(key)}
            disabled={disabled || featureForced(featureKey(key))}
          />
        }
        label={label}
      />
    </Tooltip>
  );

  return (
    <Box
      sx={{
        display: 'grid',
        gap: 3,
        gridTemplateColumns: { xs: '1fr', md: 'minmax(320px, 1fr) minmax(420px, 1.2fr)' },
        alignItems: 'flex-start',
      }}
    >
      <Paper sx={{ p: 3 }} elevation={0}>
        <Typography variant="h5" gutterBottom>
          Upload options
        </Typography>
        <Stack spacing={1.5}>
          {featureEnabled('one_shot') &&
            toggleField('One shot', 'oneShot', 'Download will be available only once.', featureForced('one_shot'))}
          {featureEnabled('stream') &&
            toggleField(
              'Streaming',
              'stream',
              'Files are streamed instead of stored. Upload starts when the receiver downloads.',
              featureForced('stream'),
            )}
          {featureEnabled('removable') &&
            toggleField(
              'Removable',
              'removable',
              'Allow manual removal of the uploaded files at any time.',
              featureForced('removable'),
            )}
          {featureEnabled('password') && (
            <>
              {toggleField(
                'Password',
                'passwordEnabled',
                'Protect your upload with credentials before upload and download.',
                featureForced('password'),
              )}
              {toggles.passwordEnabled && (
                <TextField
                  type="password"
                  label="Password"
                  size="small"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  fullWidth
                />
              )}
            </>
          )}
          {featureEnabled('extend_ttl') &&
            toggleField(
              'Extend expiration on access',
              'extendTtl',
              'Extend upload expiration date by TTL whenever accessed.',
              featureForced('extend_ttl'),
            )}
          {featureEnabled('comments') && (
            <>
              {toggleField(
                'Comments (Markdown)',
                'enableComments',
                'Add Markdown comments to the upload with live preview.',
                featureForced('comments'),
              )}
              {toggles.enableComments && (
                <TextField
                  label="Comment"
                  multiline
                  minRows={3}
                  value={comments}
                  onChange={(e) => setComments(e.target.value)}
                  fullWidth
                />
              )}
            </>
          )}
          <Divider />
          <Typography variant="body2" color="text.secondary">
            Files will be automatically removed in
          </Typography>
          <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1.5} alignItems="center">
            {ttlUnit.key !== 'unlimited' && (
              <TextField
                type="number"
                label="TTL"
                size="small"
                value={ttlValue}
                onChange={(e) => setTtlValue(Number(e.target.value))}
                disabled={!featureEnabled('set_ttl')}
                sx={{ maxWidth: 140 }}
              />
            )}
            <FormControl size="small" sx={{ minWidth: 160 }}>
              <InputLabel id="ttl-select">Unit</InputLabel>
              <Select
                labelId="ttl-select"
                label="Unit"
                value={ttlUnit.key}
                onChange={(e) => {
                  const unit = ttlUnits.find((u) => u.key === e.target.value) || ttlUnits[0];
                  setTtlUnit(unit);
                }}
                disabled={!featureEnabled('set_ttl')}
              >
                {ttlUnits.map((unit) => (
                  <MenuItem key={unit.key} value={unit.key}>
                    {unit.label}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          </Stack>
          <Button
            variant="outlined"
            onClick={() => startUpload(true)}
            disabled={!featureEnabled('stream') && !featureEnabled('text') && files.length === 0}
          >
            Create empty upload
          </Button>
        </Stack>
      </Paper>

      <Paper sx={{ p: 3 }} elevation={0}>
        <Typography variant="h5" gutterBottom>
          Add files
        </Typography>
        <Stack spacing={2}>
          <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1.5} alignItems="center">
            <Button variant="contained" startIcon={<UploadFile />} component="label">
              Select files
              <input multiple type="file" onChange={(e) => handleFiles(e.target.files)} />
            </Button>
            {files.length > 0 && (
              <Button variant="text" onClick={() => setFiles([])}>
                Clear selection
              </Button>
            )}
            {featureEnabled('text') && <Chip label="Text uploads enabled" color="success" variant="outlined" size="small" />}
          </Stack>
          {maxFileSize && (
            <Typography variant="body2" color="text.secondary">
              Max file size: {Math.round(maxFileSize / 1024 / 1024)} MB
            </Typography>
          )}
          {files.length > 0 && (
            <Paper variant="outlined">
              <List dense>
                {files.map((file) => (
                  <ListItem key={file.name}>
                    <ListItemText primary={file.name} secondary={`${Math.round(file.size / 1024)} KB`} />
                  </ListItem>
                ))}
              </List>
            </Paper>
          )}
          <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1.5} alignItems="center">
            <Button variant="contained" disabled={!isUploadReady} onClick={() => startUpload()}>
              Upload
            </Button>
          </Stack>
          {status && <Alert severity={uploadResult ? 'success' : 'info'}>{status}</Alert>}
          {uploadResult && downloadUrl && (
            <Alert severity="success">
              <Stack spacing={0.5}>
                <Typography>Upload ready.</Typography>
                <MuiLink href={downloadUrl}>{downloadUrl}</MuiLink>
              </Stack>
            </Alert>
          )}
        </Stack>
      </Paper>
    </Box>
  );
};
