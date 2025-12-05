import React from 'react';
import {
  AppBar,
  Box,
  Chip,
  Divider,
  Drawer,
  IconButton,
  List,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Toolbar,
  Typography,
} from '@mui/material';
import { AdminPanelSettings, CloudUpload, GitHub, Home, Keyboard, Lock } from '@mui/icons-material';
import { Link as RouterLink, useLocation } from 'react-router-dom';
import { useConfig } from '../context/ConfigContext';

const drawerWidth = 260;

export const Layout: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { user, config, featureEnabled, title } = useConfig();
  const location = useLocation();

  const githubEnabled = featureEnabled('github');
  const clientsEnabled = featureEnabled('clients');
  const authenticationEnabled = featureEnabled('authentication');

  const menuLinks = [
    { path: '/', label: 'Upload', icon: <CloudUpload />, show: true },
    { path: '/home', label: 'Home', icon: <Home />, show: true },
    { path: '/clients', label: 'Command line client', icon: <Keyboard />, show: clientsEnabled },
    {
      path: '/admin',
      label: 'Admin',
      icon: <AdminPanelSettings />,
      show: authenticationEnabled && !!user?.admin,
    },
    {
      path: '/login',
      label: user ? user.login : 'Authentication',
      icon: <Lock />,
      show: authenticationEnabled,
    },
  ];

  return (
    <Box sx={{ display: 'flex', minHeight: '100vh', bgcolor: 'background.default' }}>
      <AppBar
        position="fixed"
        color="default"
        sx={{ zIndex: (theme) => theme.zIndex.drawer + 1, borderBottom: 1, borderColor: 'divider' }}
      >
        <Toolbar sx={{ gap: 2 }}>
          <Typography
            variant="h6"
            noWrap
            component={RouterLink}
            to="/"
            color="inherit"
            sx={{ textDecoration: 'none', fontWeight: 700 }}
          >
            {title}
          </Typography>
          <Box sx={{ flexGrow: 1 }} />
          {githubEnabled && (
            <IconButton
              color="inherit"
              component="a"
              href="https://github.com/root-gg/plik/tree/master"
              target="_blank"
              rel="noreferrer"
            >
              <GitHub />
            </IconButton>
          )}
          {user && <Chip icon={<Lock />} label={user.login} variant="outlined" />}
        </Toolbar>
      </AppBar>

      <Drawer
        variant="permanent"
        sx={{
          width: drawerWidth,
          flexShrink: 0,
          [`& .MuiDrawer-paper`]: {
            width: drawerWidth,
            boxSizing: 'border-box',
            borderRight: 1,
            borderColor: 'divider',
          },
        }}
      >
        <Toolbar />
        <Box sx={{ overflow: 'auto', py: 2 }}>
          <Typography variant="subtitle2" sx={{ px: 2, pb: 1, textTransform: 'uppercase', color: 'text.secondary' }}>
            Navigation
          </Typography>
          <Divider />
          <List>
            {menuLinks
              .filter((link) => link.show)
              .map((link) => (
                <ListItemButton
                  key={link.path}
                  component={RouterLink}
                  to={link.path}
                  selected={location.pathname === link.path}
                >
                  <ListItemIcon>{link.icon}</ListItemIcon>
                  <ListItemText primary={link.label} />
                </ListItemButton>
              ))}
          </List>
          {config?.abuseContact && (
            <Box sx={{ px: 3, pt: 2 }}>
              <Divider sx={{ mb: 1 }} />
              <Typography variant="body2" color="text.secondary">
                Abuse contact: {config.abuseContact}
              </Typography>
            </Box>
          )}
        </Box>
      </Drawer>

      <Box component="main" sx={{ flexGrow: 1, p: 3 }}>
        <Toolbar />
        {children}
      </Box>
    </Box>
  );
};
