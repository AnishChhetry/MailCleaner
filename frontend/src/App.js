import React from 'react';
import {
  AppBar,
  Toolbar,
  Typography,
  Button,
  Container,
  Box,
  Card,
  CardContent,
  Grid,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  IconButton,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Chip,
  Alert,
  CircularProgress,
  Switch,
  FormControlLabel,
  Tabs,
  Tab,
  Fab,
  Snackbar,
  Badge,
  Avatar,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
  Divider,
  Pagination,
  Tooltip,
  LinearProgress,
  CardHeader,
  CardActions,
  Checkbox
} from '@mui/material';
import {
  Dashboard as DashboardIcon,
  Inbox as InboxIcon,
  Rule as RuleIcon,
  CleaningServices as CleanIcon,
  Delete as TrashIcon,
  Archive as ArchiveIcon,
  Settings as SettingsIcon,
  Logout as LogoutIcon,
  Add as AddIcon,
  Edit as EditIcon,
  Restore as RestoreIcon,
  DeleteForever as DeleteForeverIcon,
  Sync as SyncIcon,
  Preview as PreviewIcon,
  Google as GoogleIcon,
  Mail as MailIcon,
  TrendingUp as TrendingUpIcon,
  Filter1 as FilterIcon,
  Close as CloseIcon,
  Warning as WarningIcon,
  CheckCircle as CheckIcon,
  DarkMode,
  LightMode,
  Undo as UndoIcon,
  Security as SecurityIcon,
  MarkEmailRead as MarkReadIcon,
  MarkEmailUnread as MarkUnreadIcon,
  Unsubscribe as UnsubscribeIcon
} from '@mui/icons-material';
import { CssBaseline } from '@mui/material';
import { ThemeProvider, createTheme } from '@mui/material/styles';
import * as api from './api';


// Custom theme

// Utility functions
const getHeader = (headers, name) => headers?.find(h => h.name?.toLowerCase() === name.toLowerCase())?.value || '';
const isUnreadInDb = (email) => !email.read;
const isUnreadInGmail = (email) => email.labelIds && email.labelIds.includes('UNREAD');

function Login() {
  return (
    <Container maxWidth="md" sx={{ mt: 8 }}>
      <Card elevation={3} sx={{ p: 4, textAlign: 'center' }}>
        <CardContent>
          <Box sx={{ mb: 4 }}>
            <Typography variant="h2" component="h1" gutterBottom sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 2 }}>
              🧹 MailCleaner
            </Typography>
            <Typography variant="h5" color="text.secondary" gutterBottom>
              Clean Your Inbox, Reclaim Your Focus
            </Typography>
          </Box>

          <Typography variant="body1" paragraph sx={{ mb: 4 }}>
            MailCleaner helps you automate your inbox management. Connect your Google account to apply powerful, 
            custom rules that keep your digital space tidy by archiving or deleting emails automatically.
          </Typography>

          <Card variant="outlined" sx={{ mb: 4, textAlign: 'left' }}>
            <CardHeader title="How It Works" />
            <CardContent>
              <List>
                <ListItem>
                  <ListItemIcon><SecurityIcon color="primary" /></ListItemIcon>
                  <ListItemText 
                    primary="Connect Securely" 
                    secondary="Sign in with your Google account using secure OAuth2. We never see your password." 
                  />
                </ListItem>
                <Divider />
                <ListItem>
                  <ListItemIcon><RuleIcon color="primary" /></ListItemIcon>
                  <ListItemText 
                    primary="Create Rules" 
                    secondary="Define powerful rules based on sender, subject, keywords, or the age of the email." 
                  />
                </ListItem>
                <Divider />
                <ListItem>
                  <ListItemIcon><CleanIcon color="primary" /></ListItemIcon>
                  <ListItemText 
                    primary="Clean Automatically" 
                    secondary="Enable automation to let MailCleaner manage your inbox 24/7, or run a clean manually." 
                  />
                </ListItem>
              </List>
            </CardContent>
          </Card>

          <Button
            variant="contained"
            size="large"
            startIcon={<GoogleIcon />}
            href={api.loginUrl}
            sx={{ py: 1.5, px: 4 }}
          >
            Sign in with Google
          </Button>
        </CardContent>
      </Card>
    </Container>
  );
}

function useAuth() {
  const [authenticated, setAuthenticated] = React.useState(null);
  React.useEffect(() => {
    api.healthz()
      .then(() => setAuthenticated(true))
      .catch(() => setAuthenticated(false));
  }, []);
  return [authenticated, setAuthenticated];
}

function Dashboard() {
  const [message, setMessage] = React.useState('');
  const [error, setError] = React.useState('');
  const [loading, setLoading] = React.useState(false);
  const [syncProgress, setSyncProgress] = React.useState('');
  const [analytics, setAnalytics] = React.useState([]);
  const [rules, setRules] = React.useState([]);
  const [stats, setStats] = React.useState({ synced: 0, trash: 0, archived: 0 });

  const fetchDashboardData = React.useCallback(() => {
    api.fetchStats()
      .then(setStats)
      .catch(err => console.error("Failed to fetch stats", err));

    api.fetchSenderAnalytics()
      .then(data => setAnalytics(data.analytics || []))
      .catch(err => console.error("Failed to fetch analytics", err));

    api.fetchRules()
      .then(data => setRules(data.rules || []))
      .catch(err => console.error("Failed to fetch rules", err));
  }, []);

  React.useEffect(() => {
    fetchDashboardData();
  }, [fetchDashboardData]);

  const handleFullSync = () => {
    setLoading(true);
    setMessage('');
    setError('');
    setSyncProgress('Starting full sync...');

    // Start polling for progress
    const progressInterval = setInterval(() => {
      api.getSyncProgress()
        .then(progress => {
          if (progress.in_progress) {
            setSyncProgress(`${progress.stage}: ${Math.round(progress.percentage)}%`);
          }
        })
        .catch(() => {}); // Ignore polling errors
    }, 500); // Poll every 500ms

    api.syncEmails()
      .then(response => {
        clearInterval(progressInterval);
        setMessage(response.message);
        setSyncProgress('');
        fetchDashboardData();
      })
      .catch(err => {
        clearInterval(progressInterval);
        setError(err.message || 'Failed to sync emails.');
        setSyncProgress('');
      })
      .finally(() => {
        clearInterval(progressInterval);
        setLoading(false);
      });
  };

  const handleQuickSync = () => {
    setLoading(true);
    setMessage('');
    setError('');
    setSyncProgress('Starting quick sync...');
    
    // Start polling for progress
    const progressInterval = setInterval(() => {
      api.getSyncProgress()
        .then(progress => {
          if (progress.in_progress) {
            setSyncProgress(progress.stage);
          }
        })
        .catch(() => {}); // Ignore polling errors
    }, 300); // Poll every 300ms for quick sync

    api.syncHistory()
      .then(response => {
        clearInterval(progressInterval);
        setMessage(response.message);
        setSyncProgress('');
        fetchDashboardData();
      })
      .catch(err => {
        clearInterval(progressInterval);
        setError(err.message || 'Failed to sync recent emails.');
        setSyncProgress('');
      })
      .finally(() => {
        clearInterval(progressInterval);
        setLoading(false);
      });
  };

  return (
    <Container maxWidth="xl" sx={{ mt: 4, mb: 6, display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
      {/* Header Section */}
      <Box sx={{ mb: 6, textAlign: 'center', width: '100%' }}>
        <Typography variant="h3" component="h1" gutterBottom sx={{ 
          display: 'flex', 
          alignItems: 'center', 
          justifyContent: 'center',
          gap: 2,
          fontWeight: 700,
          background: 'linear-gradient(45deg, #1976d2, #42a5f5)',
          backgroundClip: 'text',
          WebkitBackgroundClip: 'text',
          WebkitTextFillColor: 'transparent',
          fontSize: { xs: '2.5rem', md: '3.5rem' },
          mb: 2
        }}>
          <DashboardIcon sx={{ fontSize: { xs: 40, md: 50 } }} />
          Dashboard Overview
        </Typography>
        <Typography variant="h6" color="text.secondary" sx={{ 
          mb: 4, 
          fontSize: { xs: '1.1rem', md: '1.3rem' },
          maxWidth: '800px',
          mx: 'auto',
          lineHeight: 1.6
        }}>
          Welcome back! Here's what's happening with your email management.
        </Typography>
        
        {/* Quick Stats Summary */}
        <Card sx={{ 
          mb: 4, 
          background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
          color: 'white',
          borderRadius: 3,
          boxShadow: '0 8px 32px rgba(0,0,0,0.1)',
          border: 'none',
          maxWidth: '900px',
          mx: 'auto'
        }}>
          <CardContent sx={{ p: 4 }}>
            <Box sx={{ textAlign: 'center', mb: 3 }}>
              <Typography variant="h5" gutterBottom sx={{ fontWeight: 600, mb: 2 }}>
                📧 Email Management Summary
              </Typography>
              <Typography variant="body1" sx={{ opacity: 0.9, fontSize: '1.1rem', lineHeight: 1.6 }}>
                You have <strong>{stats.synced.toLocaleString()}</strong> emails in your inbox, <strong>{stats.trash.toLocaleString()}</strong> in trash, 
                and <strong>{stats.archived.toLocaleString()}</strong> archived. <strong>{rules.length}</strong> active rules are helping you stay organized.
              </Typography>
            </Box>
            <Box sx={{ textAlign: 'center', mt: 3 }}>
              <Typography variant="h3" color="inherit" sx={{ fontWeight: 700, mb: 1 }}>
                {((stats.synced + stats.trash + stats.archived) / 1000).toFixed(1)}K
              </Typography>
              <Typography variant="body1" sx={{ opacity: 0.8, fontSize: '1rem' }}>
                Total Emails Processed
              </Typography>
            </Box>
          </CardContent>
        </Card>
      </Box>

      {/* Stats Cards with Enhanced Design */}
      <Grid container spacing={4} sx={{ mb: 6, justifyContent: 'center' }}>
        <Grid item xs={12} sm={6} lg={3}>
          <Card sx={{ 
            background: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
            color: 'white',
            position: 'relative',
            overflow: 'hidden',
            '&::before': {
              content: '""',
              position: 'absolute',
              top: 0,
              right: 0,
              width: '100px',
              height: '100px',
              background: 'rgba(255,255,255,0.1)',
              borderRadius: '50%',
              transform: 'translate(30px, -30px)'
            }
          }}>
            <CardContent sx={{ position: 'relative', zIndex: 1 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
                <MailIcon sx={{ fontSize: 40, opacity: 0.9 }} />
                <Typography variant="h6" sx={{ opacity: 0.9 }}>
                  Inbox
                </Typography>
              </Box>
              <Typography variant="h3" component="h2" sx={{ fontWeight: 700, mb: 1 }}>
                {stats.synced.toLocaleString()}
              </Typography>
              <Typography variant="body2" sx={{ opacity: 0.9 }}>
                Synced Emails
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        
        <Grid item xs={12} sm={6} lg={3}>
          <Card sx={{ 
            background: 'linear-gradient(135deg, #f093fb 0%, #f5576c 100%)',
            color: 'white',
            position: 'relative',
            overflow: 'hidden',
            '&::before': {
              content: '""',
              position: 'absolute',
              top: 0,
              right: 0,
              width: '100px',
              height: '100px',
              background: 'rgba(255,255,255,0.1)',
              borderRadius: '50%',
              transform: 'translate(30px, -30px)'
            }
          }}>
            <CardContent sx={{ position: 'relative', zIndex: 1 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
                <TrashIcon sx={{ fontSize: 40, opacity: 0.9 }} />
                <Typography variant="h6" sx={{ opacity: 0.9 }}>
                  Trash
                </Typography>
              </Box>
              <Typography variant="h3" component="h2" sx={{ fontWeight: 700, mb: 1 }}>
                {stats.trash.toLocaleString()}
              </Typography>
              <Typography variant="body2" sx={{ opacity: 0.9 }}>
                Deleted Emails
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        
        <Grid item xs={12} sm={6} lg={3}>
          <Card sx={{ 
            background: 'linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)',
            color: 'white',
            position: 'relative',
            overflow: 'hidden',
            '&::before': {
              content: '""',
              position: 'absolute',
              top: 0,
              right: 0,
              width: '100px',
              height: '100px',
              background: 'rgba(255,255,255,0.1)',
              borderRadius: '50%',
              transform: 'translate(30px, -30px)'
            }
          }}>
            <CardContent sx={{ position: 'relative', zIndex: 1 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
                <ArchiveIcon sx={{ fontSize: 40, opacity: 0.9 }} />
                <Typography variant="h6" sx={{ opacity: 0.9 }}>
                  Archive
                </Typography>
              </Box>
              <Typography variant="h3" component="h2" sx={{ fontWeight: 700, mb: 1 }}>
                {stats.archived.toLocaleString()}
              </Typography>
              <Typography variant="body2" sx={{ opacity: 0.9 }}>
                Archived Emails
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        
        <Grid item xs={12} sm={6} lg={3}>
          <Card sx={{ 
            background: 'linear-gradient(135deg, #fa709a 0%, #fee140 100%)',
            color: 'white',
            position: 'relative',
            overflow: 'hidden',
            '&::before': {
              content: '""',
              position: 'absolute',
              top: 0,
              right: 0,
              width: '100px',
              height: '100px',
              background: 'rgba(255,255,255,0.1)',
              borderRadius: '50%',
              transform: 'translate(30px, -30px)'
            }
          }}>
            <CardContent sx={{ position: 'relative', zIndex: 1 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
                <TrendingUpIcon sx={{ fontSize: 40, opacity: 0.9 }} />
                <Typography variant="h6" sx={{ opacity: 0.9 }}>
                  Rules
                </Typography>
              </Box>
              <Typography variant="h3" component="h2" sx={{ fontWeight: 700, mb: 1 }}>
                {rules.length}
              </Typography>
              <Typography variant="body2" sx={{ opacity: 0.9 }}>
                Active Rules
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Quick Actions Section */}
      <Card sx={{ 
        mb: 6, 
        background: 'linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)', 
        color: 'white',
        borderRadius: 3,
        boxShadow: '0 8px 32px rgba(0,0,0,0.1)',
        maxWidth: '900px',
        mx: 'auto'
      }}>
        <CardContent sx={{ p: 4 }}>
          <Typography variant="h5" gutterBottom sx={{ 
            display: 'flex', 
            alignItems: 'center', 
            justifyContent: 'center',
            gap: 1, 
            mb: 4,
            textAlign: 'center',
            fontWeight: 600
          }}>
            <SyncIcon /> Quick Actions
          </Typography>
          <Box sx={{ textAlign: 'center' }}>
            <Typography variant="body1" sx={{ 
              mb: 4, 
              opacity: 0.9, 
              textAlign: 'center', 
              fontSize: '1.1rem', 
              lineHeight: 1.6,
              maxWidth: '600px',
              mx: 'auto'
            }}>
              Keep your email data synchronized and up-to-date with Gmail. Choose between a full sync or quick incremental update.
            </Typography>
            
            {loading && (
              <Box sx={{ mb: 3, maxWidth: '400px', mx: 'auto' }}>
                <LinearProgress 
                  sx={{ 
                    mb: 1, 
                    bgcolor: 'rgba(255,255,255,0.3)',
                    '& .MuiLinearProgress-bar': {
                      bgcolor: 'rgba(255,255,255,0.8)'
                    }
                  }} 
                />
                <Typography variant="body2" sx={{ opacity: 0.8, textAlign: 'center' }}>
                  {syncProgress || 'Processing...'}
                </Typography>
              </Box>
            )}
            
            {message && (
              <Alert severity="success" sx={{ 
                mb: 3, 
                bgcolor: 'rgba(76, 175, 80, 0.1)', 
                color: 'white',
                maxWidth: '500px',
                mx: 'auto'
              }} onClose={() => setMessage('')}>
                {message}
              </Alert>
            )}
            
            {error && (
              <Alert severity="error" sx={{ 
                mb: 3, 
                bgcolor: 'rgba(244, 67, 54, 0.1)', 
                color: 'white',
                maxWidth: '500px',
                mx: 'auto'
              }} onClose={() => setError('')}>
                {error}
              </Alert>
            )}
            
            {syncProgress && (
              <Alert severity="info" sx={{ 
                mb: 3, 
                bgcolor: 'rgba(33, 150, 243, 0.1)', 
                color: 'white',
                maxWidth: '500px',
                mx: 'auto'
              }}>
                {syncProgress}
              </Alert>
            )}

            <Box sx={{ 
              display: 'flex', 
              flexDirection: { xs: 'column', sm: 'row' },
              gap: 3, 
              alignItems: 'center',
              justifyContent: 'center',
              maxWidth: '500px',
              mx: 'auto'
            }}>
              <Button
                variant="contained"
                onClick={handleFullSync}
                disabled={loading}
                startIcon={loading ? <CircularProgress size={20} color="inherit" /> : <SyncIcon />}
                sx={{ 
                  bgcolor: 'rgba(255,255,255,0.2)', 
                  color: 'white',
                  '&:hover': { bgcolor: 'rgba(255,255,255,0.3)' },
                  py: 1.5,
                  px: 4,
                  minWidth: 180,
                  borderRadius: 2,
                  fontWeight: 600
                }}
              >
                {loading ? 'Syncing...' : 'Full Sync'}
              </Button>
              <Button
                variant="outlined"
                onClick={handleQuickSync}
                disabled={loading}
                startIcon={<SyncIcon />}
                sx={{ 
                  borderColor: 'rgba(255,255,255,0.5)',
                  color: 'white',
                  '&:hover': { 
                    borderColor: 'white',
                    bgcolor: 'rgba(255,255,255,0.1)'
                  },
                  py: 1.5,
                  px: 4,
                  minWidth: 180,
                  borderRadius: 2,
                  fontWeight: 600
                }}
              >
                Quick Sync
              </Button>
            </Box>
          </Box>
        </CardContent>
      </Card>

      {/* Analytics and Top Senders */}
      <Grid container spacing={4} direction="column" alignItems="center" sx={{ width: '100%' }}>
        <Grid item xs={12} lg={8}>
          <Card sx={{ height: '100%' }}>
            <CardHeader 
              title="Top Email Senders" 
              avatar={<TrendingUpIcon color="primary" />}
              action={
                <Chip 
                  label={`${analytics.length} senders`} 
                  color="primary" 
                  variant="outlined" 
                  size="small"
                />
              }
            />
            <CardContent>
              {analytics.length > 0 ? (
                <TableContainer>
                  <Table>
                    <TableHead>
                      <TableRow>
                        <TableCell><strong>Rank</strong></TableCell>
                        <TableCell><strong>Sender</strong></TableCell>
                        <TableCell align="right"><strong>Email Count</strong></TableCell>
                        <TableCell align="right"><strong>Percentage</strong></TableCell>
                        <TableCell align="right"><strong>Actions</strong></TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {analytics.slice(0, 8).map((item, index) => {
                        const percentage = ((item.count / stats.synced) * 100).toFixed(1);
                        const maxCount = analytics[0]?.count || 1;
                        const barWidth = (item.count / maxCount) * 100;
                        return (
                          <TableRow key={item.sender} hover>
                            <TableCell>
                              <Avatar sx={{ 
                                width: 32, 
                                height: 32, 
                                fontSize: 14,
                                bgcolor: index < 3 ? 'primary.main' : 'grey.300',
                                color: index < 3 ? 'white' : 'text.primary'
                              }}>
                                {index + 1}
                              </Avatar>
                            </TableCell>
                            <TableCell sx={{ minWidth: 200 }}>
                              <Box>
                                <Typography variant="body2" sx={{ fontWeight: 500, mb: 0.5 }}>
                                  {item.sender}
                                </Typography>
                                <Box sx={{ 
                                  width: '100%', 
                                  height: 4, 
                                  bgcolor: 'grey.200', 
                                  borderRadius: 2,
                                  overflow: 'hidden'
                                }}>
                                  <Box sx={{
                                    width: `${barWidth}%`,
                                    height: '100%',
                                    bgcolor: index < 3 ? 'primary.main' : 'grey.400',
                                    transition: 'width 0.3s ease'
                                  }} />
                                </Box>
                              </Box>
                            </TableCell>
                            <TableCell align="right">
                              <Chip 
                                label={item.count.toLocaleString()} 
                                color={index < 3 ? 'primary' : 'default'}
                                size="small" 
                              />
                            </TableCell>
                            <TableCell align="right">
                              <Typography variant="body2" color="text.secondary" sx={{ fontWeight: 500 }}>
                                {percentage}%
                              </Typography>
                            </TableCell>
                            <TableCell align="right">
                              <Tooltip title="Block this sender">
                                <IconButton 
                                  size="small"
                                  color="error"
                                  onClick={() => {
                                    if (window.confirm(`Block "${item.sender}"? Future emails from this sender will be automatically moved to trash.`)) {
                                      api.blockSender(item.sender)
                                        .then(() => {
                                          alert(`Blocked ${item.sender}. A rule has been created.`);
                                        })
                                        .catch(err => {
                                          alert(`Failed to block: ${err.message}`);
                                        });
                                    }
                                  }}
                                >
                                  <UnsubscribeIcon />
                                </IconButton>
                              </Tooltip>
                            </TableCell>
                          </TableRow>
                        );
                      })}
                    </TableBody>
                  </Table>
                </TableContainer>
              ) : (
                <Box sx={{ textAlign: 'center', py: 4 }}>
                  <MailIcon sx={{ fontSize: 64, color: 'grey.300', mb: 2 }} />
                  <Typography variant="h6" color="text.secondary" gutterBottom>
                    No Analytics Available
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    Sync some emails to see sender analytics
                  </Typography>
                </Box>
              )}
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} lg={4}>
          <Card sx={{ height: '100%' }}>
            <CardHeader 
              title="Email Management Tips" 
              avatar={<CheckIcon color="success" />}
            />
            <CardContent>
              <List>
                <ListItem sx={{ px: 0 }}>
                  <ListItemIcon>
                    <CheckIcon color="success" />
                  </ListItemIcon>
                  <ListItemText 
                    primary="Regular Sync" 
                    secondary="Keep your data fresh with regular incremental syncs"
                  />
                </ListItem>
                <Divider />
                <ListItem sx={{ px: 0 }}>
                  <ListItemIcon>
                    <RuleIcon color="primary" />
                  </ListItemIcon>
                  <ListItemText 
                    primary="Create Rules" 
                    secondary="Set up automated rules to manage emails efficiently"
                  />
                </ListItem>
                <Divider />
                <ListItem sx={{ px: 0 }}>
                  <ListItemIcon>
                    <CleanIcon color="warning" />
                  </ListItemIcon>
                  <ListItemText 
                    primary="Preview Before Clean" 
                    secondary="Always preview changes before running a clean operation"
                  />
                </ListItem>
                <Divider />
                <ListItem sx={{ px: 0 }}>
                  <ListItemIcon>
                    <ArchiveIcon color="info" />
                  </ListItemIcon>
                  <ListItemText 
                    primary="Archive vs Delete" 
                    secondary="Archive important emails instead of deleting them"
                  />
                </ListItem>
              </List>
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Container>
  );
}

function Rules() {
  const [rules, setRules] = React.useState([]);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState('');
  const [message, setMessage] = React.useState('');
  const [editingId, setEditingId] = React.useState(null);
  const [newRule, setNewRule] = React.useState({ type: 'sender', value: '', action: 'DELETE', age_days: 0 });
  const [dialogOpen, setDialogOpen] = React.useState(false);

  const fetchRules = React.useCallback(() => {
    setLoading(true);
    api.fetchRules()
      .then(data => setRules(data.rules || []))
      .catch(err => setError(err.message || 'Failed to load rules'))
      .finally(() => setLoading(false));
  }, []);

  React.useEffect(() => {
    fetchRules();
  }, [fetchRules]);

  const clearForm = () => {
    setEditingId(null);
    setNewRule({ type: 'sender', value: '', action: 'DELETE', age_days: 0 });
    setDialogOpen(false);
  };

  const handleSubmit = (e) => {
    e.preventDefault();
    setMessage('');
    setError('');
    const actionPromise = editingId
      ? api.updateRule(editingId, newRule)
      : api.createRule(newRule);

    actionPromise
      .then(data => {
        setMessage(data.message || (editingId ? 'Rule updated' : 'Rule created'));
        clearForm();
        fetchRules();
      })
      .catch(err => setError(err.message || 'Failed to save rule'));
  };

  const handleEdit = (rule) => {
    setEditingId(rule.id);
    setNewRule({ type: rule.type, value: rule.value, action: rule.action, age_days: rule.age_days });
    setDialogOpen(true);
  };

  const handleDeleteRule = (id) => {
    api.deleteRule(id)
      .then(() => {
        setMessage('Rule deleted');
        fetchRules();
      })
      .catch(err => setError(err.message || 'Failed to delete rule'));
  };

  const getActionChip = (action) => {
    const configs = {
      DELETE: { label: 'Trash', color: 'error', icon: <TrashIcon /> },
      ARCHIVE: { label: 'Archive', color: 'warning', icon: <ArchiveIcon /> },
      MARK_READ: { label: 'Mark Read', color: 'info', icon: <CheckIcon /> }
    };
    const config = configs[action] || configs.DELETE;
    return <Chip label={config.label} color={config.color} size="small" />;
  };

  if (loading) return (
    <Container maxWidth="lg" sx={{ mt: 2, display: 'flex', justifyContent: 'center' }}>
      <CircularProgress />
    </Container>
  );

  return (
    <Container maxWidth="lg" sx={{ mt: 4, mb: 6, display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
      <Box sx={{ 
        display: 'flex', 
        justifyContent: 'space-between', 
        alignItems: 'center', 
        mb: 4, 
        width: '100%',
        flexWrap: 'wrap',
        gap: 2
      }}>
        <Typography variant="h4" sx={{ 
          display: 'flex', 
          alignItems: 'center', 
          gap: 1,
          fontWeight: 700,
          background: 'linear-gradient(45deg, #1976d2, #42a5f5)',
          backgroundClip: 'text',
          WebkitBackgroundClip: 'text',
          WebkitTextFillColor: 'transparent'
        }}>
          <RuleIcon /> Rules Management
        </Typography>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={() => setDialogOpen(true)}
          sx={{
            borderRadius: 2,
            px: 3,
            py: 1.5,
            fontWeight: 600,
            boxShadow: '0 4px 16px rgba(25, 118, 210, 0.3)'
          }}
        >
          Add New Rule
        </Button>
      </Box>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError('')}>
          {error}
        </Alert>
      )}
      {message && (
        <Alert severity="success" sx={{ mb: 2 }} onClose={() => setMessage('')}>
          {message}
        </Alert>
      )}

      <Card sx={{ 
        width: '100%',
        borderRadius: 3,
        boxShadow: '0 8px 32px rgba(0,0,0,0.1)',
        overflow: 'hidden'
      }}>
        <TableContainer>
          <Table>
            <TableHead sx={{ 
              bgcolor: (theme) => theme.palette.mode === 'dark' ? 'grey.800' : 'grey.50'
            }}>
              <TableRow>
                <TableCell sx={{ 
                  fontWeight: 600, 
                  fontSize: '1rem',
                  color: (theme) => theme.palette.mode === 'dark' ? 'text.primary' : 'text.primary'
                }}>Type</TableCell>
                <TableCell sx={{ 
                  fontWeight: 600, 
                  fontSize: '1rem',
                  color: (theme) => theme.palette.mode === 'dark' ? 'text.primary' : 'text.primary'
                }}>Value</TableCell>
                <TableCell sx={{ 
                  fontWeight: 600, 
                  fontSize: '1rem',
                  color: (theme) => theme.palette.mode === 'dark' ? 'text.primary' : 'text.primary'
                }}>Action</TableCell>
                <TableCell sx={{ 
                  fontWeight: 600, 
                  fontSize: '1rem',
                  color: (theme) => theme.palette.mode === 'dark' ? 'text.primary' : 'text.primary'
                }}>Age (Days)</TableCell>
                <TableCell align="right" sx={{ 
                  fontWeight: 600, 
                  fontSize: '1rem',
                  color: (theme) => theme.palette.mode === 'dark' ? 'text.primary' : 'text.primary'
                }}>Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {rules.map((rule) => (
                <TableRow key={rule.id} hover>
                  <TableCell>
                    <Chip label={rule.type} variant="outlined" size="small" />
                  </TableCell>
                  <TableCell>{rule.value}</TableCell>
                  <TableCell>{getActionChip(rule.action)}</TableCell>
                  <TableCell>
                    {rule.age_days > 0 ? `> ${rule.age_days}` : 'Any'}
                  </TableCell>
                  <TableCell align="right">
                    <IconButton onClick={() => handleEdit(rule)} size="small">
                      <EditIcon />
                    </IconButton>
                    <IconButton 
                      onClick={() => handleDeleteRule(rule.id)} 
                      size="small" 
                      color="error"
                    >
                      <TrashIcon />
                    </IconButton>
                  </TableCell>
                </TableRow>
              ))}
              {rules.length === 0 && (
                <TableRow>
                  <TableCell colSpan={5} align="center">
                    <Typography color="text.secondary">
                      No rules created yet. Click "Add Rule" to get started.
                    </Typography>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </TableContainer>
      </Card>

      {/* Rule Dialog */}
      <Dialog open={dialogOpen} onClose={clearForm} maxWidth="sm" fullWidth>
        <form onSubmit={handleSubmit}>
          <DialogTitle>
            {editingId ? 'Edit Rule' : 'Add New Rule'}
          </DialogTitle>
          <DialogContent>
            <Grid container spacing={2} sx={{ mt: 1 }}>
              <Grid item xs={12}>
                <FormControl fullWidth>
                  <InputLabel>Type</InputLabel>
                  <Select
                    value={newRule.type}
                    label="Type"
                    onChange={(e) => setNewRule(prev => ({ ...prev, type: e.target.value }))}
                  >
                    <MenuItem value="sender">Sender</MenuItem>
                    <MenuItem value="subject">Subject</MenuItem>
                    <MenuItem value="keyword">Keyword</MenuItem>
                  </Select>
                </FormControl>
              </Grid>
              <Grid item xs={12}>
                <TextField
                  fullWidth
                  label="Rule Value"
                  value={newRule.value}
                  onChange={(e) => setNewRule(prev => ({ ...prev, value: e.target.value }))}
                  placeholder="e.g., 'newsletter@example.com'"
                  required
                />
              </Grid>
              <Grid item xs={12} sm={6}>
                <FormControl fullWidth>
                  <InputLabel>Action</InputLabel>
                  <Select
                    value={newRule.action}
                    label="Action"
                    onChange={(e) => setNewRule(prev => ({ ...prev, action: e.target.value }))}
                  >
                    <MenuItem value="DELETE">Move to Trash</MenuItem>
                    <MenuItem value="ARCHIVE">Archive</MenuItem>
                    <MenuItem value="MARK_READ">Mark as Read</MenuItem>
                  </Select>
                </FormControl>
              </Grid>
              <Grid item xs={12} sm={6}>
                <TextField
                  fullWidth
                  type="number"
                  label="Age (Days)"
                  value={newRule.age_days}
                  onChange={(e) => setNewRule(prev => ({ ...prev, age_days: parseInt(e.target.value) || 0 }))}
                  helperText="0 for any age"
                />
              </Grid>
            </Grid>
          </DialogContent>
          <DialogActions>
            <Button onClick={clearForm}>Cancel</Button>
            <Button type="submit" variant="contained">
              {editingId ? 'Update Rule' : 'Add Rule'}
            </Button>
          </DialogActions>
        </form>
      </Dialog>
    </Container>
  );
}

function Clean({ showUndoToast }) {
  const [message, setMessage] = React.useState('');
  const [error, setError] = React.useState('');
  const [loading, setLoading] = React.useState(false);
  const [previewData, setPreviewData] = React.useState(null);
  const [excludedIds, setExcludedIds] = React.useState(new Set());
  const [permanentDelete, setPermanentDelete] = React.useState(false);
  const [subscribedSenders, setSubscribedSenders] = React.useState([]);
  const [loadingSubscribed, setLoadingSubscribed] = React.useState(false);
  const [showSubscriptions, setShowSubscriptions] = React.useState(false);

  const fetchSubscribedSenders = React.useCallback(() => {
    setLoadingSubscribed(true);
    api.fetchSubscribedSenders()
      .then(data => {
        setSubscribedSenders(data.subscribed_senders || []);
      })
      .catch(err => {
        console.error('Failed to fetch subscribed senders:', err);
        setError('Failed to load subscriptions');
      })
      .finally(() => setLoadingSubscribed(false));
  }, []);

  const handleToggleSubscriptions = () => {
    if (!showSubscriptions && subscribedSenders.length === 0) {
      fetchSubscribedSenders();
    }
    setShowSubscriptions(!showSubscriptions);
  };

  const handleUnsubscribeFromList = (sender, unsubscribeHeader) => {
    if (!window.confirm(`Unsubscribe from "${sender}"? This will use the newsletter's official unsubscribe method.`)) {
      return;
    }
    setMessage('');
    setError('');
    api.unsubscribeFromNewsletter(unsubscribeHeader, sender)
      .then((response) => {
        setMessage(`Successfully unsubscribed from ${sender} via ${response.method}`);
        showUndoToast(`Successfully unsubscribed from ${sender}`);
        fetchSubscribedSenders();
      })
      .catch(err => {
        setError(`Failed to unsubscribe: ${err.message}`);
      });
  };

  const handlePreview = () => {
    setLoading(true);
    setMessage('');
    setError('');
    setPreviewData(null);
    api.previewClean()
      .then(data => {
        setPreviewData(data);
        setMessage(data.message);
        setExcludedIds(new Set());
      })
      .catch(err => setError(err.message || 'Failed to generate preview'))
      .finally(() => setLoading(false));
  };

  const handleClean = () => {
    setLoading(true);
    setMessage('');
    setError('');
    const allIds = (previewData?.affected || []).map(e => e.id);
    const idsToProcess = allIds.filter(id => !excludedIds.has(id));
    api.triggerClean({ ids: idsToProcess, permanentDelete })
      .then(data => {
        setMessage(`Cleaning complete! ${data.affected_count} emails processed.`);
        showUndoToast(`${data.affected_count} emails processed.`, null);
        setPreviewData(null);
      })
      .catch(err => setError(err.message || 'Failed to trigger cleaning'))
      .finally(() => setLoading(false));
  };

  return (
    <Container maxWidth="lg" sx={{ mt: 4, mb: 6, display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
      <Typography variant="h4" gutterBottom sx={{ 
        display: 'flex', 
        alignItems: 'center', 
        gap: 1,
        fontWeight: 700,
        background: 'linear-gradient(45deg, #1976d2, #42a5f5)',
        backgroundClip: 'text',
        WebkitBackgroundClip: 'text',
        WebkitTextFillColor: 'transparent',
        mb: 3
      }}>
        <CleanIcon /> Email Cleaning
      </Typography>

      {/* Success/Error Messages - Prominent Position */}
      {error && (
        <Alert severity="error" sx={{ mb: 3, width: '100%' }} onClose={() => setError('')}>
          {error}
        </Alert>
      )}
      {message && (
        <Alert severity="success" sx={{ mb: 3, width: '100%' }} onClose={() => setMessage('')}>
          {message}
        </Alert>
      )}

      {/* Newsletter Subscriptions Button */}
      <Box sx={{ mb: 4, width: '100%', display: 'flex', justifyContent: 'center' }}>
        <Button
          variant="outlined"
          size="large"
          color="info"
          startIcon={<MailIcon />}
          onClick={handleToggleSubscriptions}
          sx={{
            borderRadius: 2,
            px: 4,
            py: 1.5,
            fontWeight: 600,
            borderWidth: 2,
            '&:hover': {
              borderWidth: 2
            }
          }}
        >
          {showSubscriptions ? 'Hide' : 'Manage'} Newsletter Subscriptions
        </Button>
      </Box>

      {/* Subscribed Senders Section - Collapsible */}
      {showSubscriptions && (
        <Card sx={{ 
          mb: 4, 
          width: '100%',
          borderRadius: 3,
          boxShadow: '0 8px 32px rgba(0,0,0,0.1)',
          border: '2px solid',
          borderColor: 'info.main'
        }}>
          <CardHeader 
            title="Newsletter & Marketing Subscriptions" 
            avatar={<MailIcon color="info" />}
            titleTypographyProps={{ 
              variant: 'h6', 
              fontWeight: 600,
              color: 'info.main'
            }}
            subheader={subscribedSenders.length > 0 ? `${subscribedSenders.length} subscription${subscribedSenders.length !== 1 ? 's' : ''} found` : 'Scanning your inbox...'}
          />
          <CardContent sx={{ p: 3 }}>
            <Typography variant="body2" color="text.secondary" sx={{ mb: 3, textAlign: 'center' }}>
              These senders have a List-Unsubscribe header (newsletters, marketing emails). 
              Click unsubscribe to use their official unsubscribe method.
            </Typography>
            
            {loadingSubscribed ? (
              <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', py: 4 }}>
                <CircularProgress size={50} />
                <Typography variant="body2" color="text.secondary" sx={{ mt: 2 }}>
                  Scanning inbox for newsletters...
                </Typography>
              </Box>
            ) : subscribedSenders.length > 0 ? (
              <TableContainer>
                <Table>
                  <TableHead>
                    <TableRow>
                      <TableCell><strong>Sender</strong></TableCell>
                      <TableCell><strong>Sample Subject</strong></TableCell>
                      <TableCell align="right"><strong>Email Count</strong></TableCell>
                      <TableCell align="right"><strong>Action</strong></TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {subscribedSenders.slice(0, 20).map((sender) => (
                      <TableRow key={sender.sender} hover>
                        <TableCell>
                          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                            <MailIcon fontSize="small" color="action" />
                            <Typography variant="body2" sx={{ fontWeight: 500 }}>
                              {sender.sender}
                            </Typography>
                          </Box>
                        </TableCell>
                        <TableCell>
                          <Typography variant="body2" color="text.secondary" noWrap sx={{ maxWidth: 300 }}>
                            {sender.sample_subject || 'N/A'}
                          </Typography>
                        </TableCell>
                        <TableCell align="right">
                          <Chip 
                            label={sender.count} 
                            size="small" 
                            color="primary"
                            variant="outlined"
                          />
                        </TableCell>
                        <TableCell align="right">
                          <Button
                            variant="contained"
                            size="small"
                            color="success"
                            startIcon={<UnsubscribeIcon />}
                            onClick={() => handleUnsubscribeFromList(sender.sender, sender.unsubscribe_header)}
                          >
                            Unsubscribe
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            ) : (
              <Box sx={{ textAlign: 'center', py: 4 }}>
                <MailIcon sx={{ fontSize: 64, color: 'grey.300', mb: 2 }} />
                <Typography variant="h6" color="text.secondary" gutterBottom>
                  No Subscriptions Found
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  No emails with List-Unsubscribe headers detected in your inbox.
                </Typography>
              </Box>
            )}
          </CardContent>
        </Card>
      )}

      {/* Rule-Based Cleaning Section */}
      <Card sx={{ 
        mb: 4, 
        width: '100%',
        borderRadius: 3,
        boxShadow: '0 8px 32px rgba(0,0,0,0.1)'
      }}>
        <CardContent sx={{ p: 4 }}>
          <Typography variant="body1" paragraph sx={{ 
            textAlign: 'center', 
            fontSize: '1.1rem',
            lineHeight: 1.6,
            mb: 3
          }}>
            Generate a preview to see which emails will be affected by your rules. Actions will move emails 
            to trash, archive them, or mark them as read.
          </Typography>

          {loading && <LinearProgress sx={{ mb: 2 }} />}

          <Box sx={{ 
            display: 'flex', 
            gap: 3, 
            flexWrap: 'wrap', 
            justifyContent: 'center',
            mt: 3
          }}>
            <Button
              variant="outlined"
              onClick={handlePreview}
              disabled={loading}
              startIcon={loading ? <CircularProgress size={20} /> : <PreviewIcon />}
              sx={{
                borderRadius: 2,
                px: 4,
                py: 1.5,
                fontWeight: 600,
                minWidth: 200
              }}
            >
              {loading ? 'Generating...' : '1. Generate Preview'}
            </Button>
            {previewData?.affected?.some(e => e.action === 'DELETE') && (
              <FormControlLabel
                control={<Checkbox checked={permanentDelete} onChange={(e) => setPermanentDelete(e.target.checked)} />}
                label="Permanently delete items marked for Delete"
              />
            )}

            {previewData?.affected?.length > 0 && (
              <Button
                variant="contained"
                onClick={handleClean}
                disabled={loading}
                startIcon={loading ? <CircularProgress size={20} /> : <CleanIcon />}
                color="warning"
                sx={{
                  borderRadius: 2,
                  px: 4,
                  py: 1.5,
                  fontWeight: 600,
                  minWidth: 200,
                  boxShadow: '0 4px 16px rgba(255, 152, 0, 0.3)'
                }}
              >
                {loading ? 'Cleaning...' : `2. Run Clean on ${(previewData.affected || []).filter(e => !excludedIds.has(e.id)).length} Emails`}
              </Button>
            )}
          </Box>
        </CardContent>
      </Card>

      {previewData && (
        <Card>
          <CardHeader 
            title={`Preview: Emails to be Processed (${previewData?.affected?.length || 0})`} 
          />
          <CardContent>
            {previewData.affected?.length > 0 ? (
              <TableContainer>
                <Table>
                  <TableHead>
                    <TableRow>
                      <TableCell padding="checkbox">
                        <Checkbox
                          indeterminate={excludedIds.size > 0 && excludedIds.size < previewData.affected.length}
                          checked={excludedIds.size === 0}
                          onChange={(e) => {
                            if (e.target.checked) {
                              setExcludedIds(new Set());
                            } else {
                              setExcludedIds(new Set(previewData.affected.map(e => e.id)));
                            }
                          }}
                        />
                      </TableCell>
                      <TableCell>Action</TableCell>
                      <TableCell>Sender</TableCell>
                      <TableCell>Subject</TableCell>
                      <TableCell>Date</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {previewData.affected.map((email) => (
                      <TableRow key={email.id} hover>
                        <TableCell padding="checkbox">
                          <Checkbox
                            checked={!excludedIds.has(email.id)}
                            onChange={(e) => {
                              const next = new Set(excludedIds);
                              if (e.target.checked) {
                                next.delete(email.id);
                              } else {
                                next.add(email.id);
                              }
                              setExcludedIds(next);
                            }}
                          />
                        </TableCell>
                        <TableCell>
                          <Chip 
                            label={email.action} 
                            size="small"
                            color={email.action === 'DELETE' ? 'error' : email.action === 'ARCHIVE' ? 'warning' : 'info'}
                          />
                        </TableCell>
                        <TableCell>{email.sender}</TableCell>
                        <TableCell>{email.subject}</TableCell>
                        <TableCell>{new Date(email.date).toLocaleString()}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            ) : (
              <Alert severity="info">
                No emails matched your rules.
              </Alert>
            )}
          </CardContent>
        </Card>
      )}
    </Container>
  );
}

function EmailDetailModal({ emailId, onClose }) {
  const [email, setEmail] = React.useState(null);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState('');

  React.useEffect(() => {
    if (!emailId) return;
    setLoading(true);
    setError('');
    api.fetchEmailDetails(emailId)
      .then(data => setEmail(data.email))
      .catch(() => setError('Failed to load email content.'))
      .finally(() => setLoading(false));
  }, [emailId]);

  const getHeader = (name) => {
    if (!email?.payload?.headers) return '';
    const header = email.payload.headers.find(h => h.name.toLowerCase() === name.toLowerCase());
    return header ? header.value : '';
  };

  return (
    <Dialog open={!!emailId} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        Email Details
        <IconButton onClick={onClose} size="small">
          <CloseIcon />
        </IconButton>
      </DialogTitle>
      <DialogContent dividers>
        {loading && (
          <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
            <CircularProgress />
          </Box>
        )}
        {error && (
          <Alert severity="error">{error}</Alert>
        )}
        {email && (
          <Box>
            <Typography variant="h6" gutterBottom>
              {getHeader('Subject')}
            </Typography>
            <Card variant="outlined" sx={{ mb: 2 }}>
              <CardContent>
                <Grid container spacing={1}>
                  <Grid item xs={12}>
                    <Typography variant="body2" color="text.secondary">
                      <strong>From:</strong> {getHeader('From')}
                    </Typography>
                  </Grid>
                  <Grid item xs={12}>
                    <Typography variant="body2" color="text.secondary">
                      <strong>Date:</strong> {new Date(getHeader('Date')).toLocaleString()}
                    </Typography>
                  </Grid>
                </Grid>
              </CardContent>
            </Card>
            <Divider sx={{ my: 2 }} />
            {email.snippet && email.snippet.includes('<') ? (
              <Box 
                sx={{ 
                  '& *': { maxWidth: '100%' },
                  '& img': { maxWidth: '100%', height: 'auto' },
                  '& table': { maxWidth: '100%', overflow: 'auto' },
                  '& p': { margin: '8px 0' },
                  '& a': { color: 'primary.main', textDecoration: 'underline' }
                }}
                dangerouslySetInnerHTML={{ __html: email.snippet }}
              />
            ) : (
              <Typography variant="body2" sx={{ whiteSpace: 'pre-wrap' }}>
                {email.snippet}
              </Typography>
            )}
          </Box>
        )}
      </DialogContent>
    </Dialog>
  );
}

function PaginatedEmails({ page, setPage, showUndoToast, onViewEmail, triggerQuickSync }) {
  const [emails, setEmails] = React.useState([]);
  const [pageSize, setPageSize] = React.useState(20);
  const [totalEmails, setTotalEmails] = React.useState(0);
  const [filter, setFilter] = React.useState('');
  const [loading, setLoading] = React.useState(true);
  const [syncing, setSyncing] = React.useState(false);
  const [syncingProgress, setSyncingProgress] = React.useState('');
  const [selectedEmails, setSelectedEmails] = React.useState(new Set());
  const [bulkActionLoading, setBulkActionLoading] = React.useState(false);
  const [intervalVersion, setIntervalVersion] = React.useState(0);

  const fetchData = React.useCallback(() => {
    setLoading(true);
    api.fetchEmailsPaginated(page, pageSize, filter)
      .then(data => {
        setEmails(data.emails || []);
        setTotalEmails(data.total || 0);
      })
      .finally(() => setLoading(false));
  }, [page, pageSize, filter]);

  const handlePageSizeChange = (newPageSize) => {
    setPageSize(newPageSize);
    setPage(1); // Reset to first page when changing page size
  };

  React.useEffect(() => { fetchData() }, [fetchData]);

  // Real-time sync: run incremental sync using configured interval (default 30s)
  React.useEffect(() => {
    const getMs = () => {
      const v = parseInt(localStorage.getItem('quickSyncMs') || '30000', 10);
      return Number.isFinite(v) ? v : 30000;
    };

    const ms = getMs();
    if (!ms || ms <= 0) {
      return () => {};
    }

    const interval = setInterval(() => {
      api.syncHistory()
        .then(data => {
          if (data.message && !data.message.includes('No new history')) {
            console.log('Quick sync:', data.message);
            fetchData();
          }
        })
        .catch(err => console.warn('Quick sync failed:', err));
    }, ms);

    return () => clearInterval(interval);
  }, [fetchData, intervalVersion]);

  // Listen for interval changes from Settings
  React.useEffect(() => {
    const handler = () => setIntervalVersion(v => v + 1);
    window.addEventListener('quick-sync-interval-changed', handler);
    return () => window.removeEventListener('quick-sync-interval-changed', handler);
  }, []);

  const handleFullSync = () => {
    setSyncing(true);
    setSyncingProgress('Starting full sync...');
    
    const progressInterval = setInterval(() => {
      api.getSyncProgress()
        .then(progress => {
          if (progress.in_progress) {
            setSyncingProgress(`${progress.stage}: ${Math.round(progress.percentage)}%`);
          }
        })
        .catch(() => {});
    }, 500);

    api.syncEmails()
      .then(data => {
        clearInterval(progressInterval);
        showUndoToast(`Full sync completed. ${data.total} emails synced.`);
        fetchData();
      })
      .catch(err => {
        clearInterval(progressInterval);
        showUndoToast(`Sync failed: ${err.message}`);
      })
      .finally(() => {
        clearInterval(progressInterval);
        setSyncing(false);
        setSyncingProgress('');
      });
  };

  const handleIncrementalSync = React.useCallback(() => {
    setSyncing(true);
    setSyncingProgress('Starting quick sync...');
    
    const progressInterval = setInterval(() => {
      api.getSyncProgress()
        .then(progress => {
          if (progress.in_progress) {
            setSyncingProgress(progress.stage);
          }
        })
        .catch(() => {});
    }, 300);

    api.syncHistory()
      .then(data => {
        clearInterval(progressInterval);
        if (data.message) {
          showUndoToast(data.message);
          if (!data.message.includes('No new history')) {
            fetchData();
          }
        }
      })
      .catch(err => {
        clearInterval(progressInterval);
        showUndoToast(`Quick sync failed: ${err.message}`);
      })
      .finally(() => {
        clearInterval(progressInterval);
        setSyncing(false);
        setSyncingProgress('');
      });
  }, [fetchData, showUndoToast]);

  // Expose quick sync to parent component via triggerQuickSync prop
  React.useEffect(() => {
    if (triggerQuickSync) {
      triggerQuickSync.current = handleIncrementalSync;
    }
  }, [handleIncrementalSync, triggerQuickSync]);

  // Selection management
  const handleSelectAll = () => {
    if (selectedEmails.size === emails.length) {
      setSelectedEmails(new Set());
    } else {
      setSelectedEmails(new Set(emails.map(email => email.id)));
    }
  };

  const handleSelectEmail = (emailId) => {
    const newSelected = new Set(selectedEmails);
    if (newSelected.has(emailId)) {
      newSelected.delete(emailId);
    } else {
      newSelected.add(emailId);
    }
    setSelectedEmails(newSelected);
  };

  // Bulk actions
  const handleBulkMarkRead = () => {
    if (selectedEmails.size === 0) return;
    
    setBulkActionLoading(true);
    api.bulkMarkRead(Array.from(selectedEmails))
      .then(data => {
        showUndoToast(`Marked ${data.successCount} emails as read`);
        setSelectedEmails(new Set());
        fetchData();
      })
      .catch(err => {
        showUndoToast(`Bulk mark as read failed: ${err.message}`);
      })
      .finally(() => setBulkActionLoading(false));
  };

  const handleBulkMarkUnread = () => {
    if (selectedEmails.size === 0) return;
    
    setBulkActionLoading(true);
    api.bulkMarkUnread(Array.from(selectedEmails))
      .then(data => {
        showUndoToast(`Marked ${data.successCount} emails as unread`);
        setSelectedEmails(new Set());
        fetchData();
      })
      .catch(err => {
        showUndoToast(`Bulk mark as unread failed: ${err.message}`);
      })
      .finally(() => setBulkActionLoading(false));
  };

  const handleBulkDelete = () => {
    if (selectedEmails.size === 0) return;
    
    if (!window.confirm(`Are you sure you want to move ${selectedEmails.size} emails to trash?`)) {
      return;
    }
    
    setBulkActionLoading(true);
    api.bulkDelete(Array.from(selectedEmails))
      .then(data => {
        showUndoToast(`Moved ${data.successCount} emails to trash`);
        setSelectedEmails(new Set());
        fetchData();
      })
      .catch(err => {
        showUndoToast(`Bulk delete failed: ${err.message}`);
      })
      .finally(() => setBulkActionLoading(false));
  };

  const handleBulkArchive = () => {
    if (selectedEmails.size === 0) return;
    
    setBulkActionLoading(true);
    const ids = Array.from(selectedEmails);
    // Optimistically remove from inbox list, then refresh
    setEmails(prev => prev.filter(e => !ids.includes(e.id)));
    api.bulkArchive(ids)
      .then(data => {
        showUndoToast(`Archived ${data.successCount} emails`);
        setSelectedEmails(new Set());
        fetchData();
      })
      .catch(err => {
        showUndoToast(`Bulk archive failed: ${err.message}`);
      })
      .finally(() => setBulkActionLoading(false));
  };

  const handleDelete = (e, id) => {
    e.stopPropagation();
    api.deleteEmail(id).then(res => {
      showUndoToast(`Email moved to trash.`, res.id);
      fetchData();
    });
  };

  const handleArchive = (e, id) => {
    e.stopPropagation();
    // Optimistically remove from current inbox list, then refresh
    setEmails(prev => prev.filter(e => e.id !== id));
    api.archiveEmail(id).then(() => fetchData());
  };

  const handleBlockSender = (e, email) => {
    e.stopPropagation();
    if (!window.confirm(`Block "${email.sender}"? Future emails from this sender will be automatically moved to trash.`)) {
      return;
    }
    api.blockSender(email.sender)
      .then(() => {
        showUndoToast(`Blocked ${email.sender}. A rule has been created.`);
      })
      .catch(err => {
        showUndoToast(`Failed to block: ${err.message}`);
      });
  };

  const totalPages = Math.ceil(totalEmails / pageSize);

  return (
    <Container maxWidth="lg" sx={{ mt: 4, mb: 6, display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
      <Box sx={{ 
        display: 'flex', 
        justifyContent: 'space-between', 
        alignItems: 'center', 
        mb: 4, 
        width: '100%',
        flexWrap: 'wrap',
        gap: 2
      }}>
        <Typography variant="h4" sx={{ 
          display: 'flex', 
          alignItems: 'center', 
          gap: 1,
          fontWeight: 700,
          background: 'linear-gradient(45deg, #1976d2, #42a5f5)',
          backgroundClip: 'text',
          WebkitBackgroundClip: 'text',
          WebkitTextFillColor: 'transparent'
        }}>
          <InboxIcon /> Email Inbox
        </Typography>
        <Box sx={{ display: 'flex', gap: 1, alignItems: 'center', flexWrap: 'wrap' }}>
          {selectedEmails.size > 0 && (
            <Box sx={{ display: 'flex', gap: 1, alignItems: 'center', mr: 2 }}>
              <Typography variant="body2" color="text.secondary">
                {selectedEmails.size} selected
              </Typography>
              <Button
                variant="outlined"
                size="small"
                onClick={handleBulkMarkRead}
                disabled={bulkActionLoading}
                startIcon={<MarkReadIcon />}
              >
                Mark Read
              </Button>
              <Button
                variant="outlined"
                size="small"
                onClick={handleBulkMarkUnread}
                disabled={bulkActionLoading}
                startIcon={<MarkUnreadIcon />}
              >
                Mark Unread
              </Button>
              <Button
                variant="outlined"
                size="small"
                onClick={handleBulkDelete}
                disabled={bulkActionLoading}
                startIcon={<TrashIcon />}
                color="error"
              >
                Delete
              </Button>
              <Button
                variant="outlined"
                size="small"
                onClick={handleBulkArchive}
                disabled={bulkActionLoading}
                startIcon={<ArchiveIcon />}
                color="success"
              >
                Archive
              </Button>
            </Box>
          )}
          
          <FormControl size="small" sx={{ minWidth: 120 }}>
            <InputLabel>Show</InputLabel>
            <Select
              value={pageSize}
              label="Show"
              onChange={(e) => handlePageSizeChange(e.target.value)}
            >
              <MenuItem value={20}>20 per page</MenuItem>
              <MenuItem value={50}>50 per page</MenuItem>
              <MenuItem value={100}>100 per page</MenuItem>
              <MenuItem value={totalEmails}>Show All</MenuItem>
            </Select>
          </FormControl>
          
          <Button
            variant="outlined"
            size="small"
            onClick={handleIncrementalSync}
            disabled={syncing}
            startIcon={<SyncIcon />}
          >
            {syncing && syncingProgress ? syncingProgress : 'Quick Sync'}
          </Button>
          <Button
            variant="contained"
            size="small"
            onClick={handleFullSync}
            disabled={syncing}
            startIcon={syncing ? <CircularProgress size={16} /> : <SyncIcon />}
          >
            {syncing && syncingProgress ? syncingProgress : (syncing ? 'Syncing...' : 'Full Sync')}
          </Button>
          <TextField
            size="small"
            placeholder="Filter by sender or subject..."
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            InputProps={{
              startAdornment: <FilterIcon color="action" sx={{ mr: 1 }} />
            }}
          />
        </Box>
      </Box>

      <Card sx={{ 
        width: '100%',
        borderRadius: 3,
        boxShadow: '0 8px 32px rgba(0,0,0,0.1)',
        overflow: 'hidden'
      }}>
        {loading && <LinearProgress />}
        <TableContainer>
          <Table>
            <TableHead sx={{ 
              bgcolor: (theme) => theme.palette.mode === 'dark' ? 'grey.800' : 'grey.50'
            }}>
              <TableRow>
                <TableCell padding="checkbox" sx={{ 
                  fontWeight: 600, 
                  fontSize: '1rem',
                  color: (theme) => theme.palette.mode === 'dark' ? 'text.primary' : 'text.primary'
                }}>
                  <Checkbox
                    indeterminate={selectedEmails.size > 0 && selectedEmails.size < emails.length}
                    checked={emails.length > 0 && selectedEmails.size === emails.length}
                    onChange={handleSelectAll}
                  />
                </TableCell>
                <TableCell width="50" sx={{ 
                  fontWeight: 600, 
                  fontSize: '1rem',
                  color: (theme) => theme.palette.mode === 'dark' ? 'text.primary' : 'text.primary'
                }}></TableCell>
                <TableCell sx={{ 
                  fontWeight: 600, 
                  fontSize: '1rem',
                  color: (theme) => theme.palette.mode === 'dark' ? 'text.primary' : 'text.primary'
                }}>Sender</TableCell>
                <TableCell sx={{ 
                  fontWeight: 600, 
                  fontSize: '1rem',
                  color: (theme) => theme.palette.mode === 'dark' ? 'text.primary' : 'text.primary'
                }}>Subject</TableCell>
                <TableCell sx={{ 
                  fontWeight: 600, 
                  fontSize: '1rem',
                  color: (theme) => theme.palette.mode === 'dark' ? 'text.primary' : 'text.primary'
                }}>Date/Time</TableCell>
                <TableCell align="right" sx={{ 
                  fontWeight: 600, 
                  fontSize: '1rem',
                  color: (theme) => theme.palette.mode === 'dark' ? 'text.primary' : 'text.primary'
                }}>Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {emails.map((email) => (
                <TableRow 
                  key={email.id} 
                  hover 
                  onClick={() => onViewEmail(email.id)}
                  sx={{ cursor: 'pointer', fontWeight: !email.read ? 'bold' : 'normal' }}
                >
                  <TableCell padding="checkbox" onClick={e => e.stopPropagation()}>
                    <Checkbox
                      checked={selectedEmails.has(email.id)}
                      onChange={() => handleSelectEmail(email.id)}
                    />
                  </TableCell>
                  <TableCell>
                    {!email.read && (
                      <Badge color="primary" variant="dot" />
                    )}
                  </TableCell>
                  <TableCell sx={{ fontWeight: !email.read ? 'bold' : 'normal' }}>
                    {email.sender}
                  </TableCell>
                  <TableCell sx={{ fontWeight: !email.read ? 'bold' : 'normal' }}>
                    {email.subject}
                  </TableCell>
                  <TableCell sx={{ fontWeight: !email.read ? 'bold' : 'normal' }}>
                    {new Date(email.date).toLocaleString()}
                  </TableCell>
                  <TableCell align="right" onClick={e => e.stopPropagation()}>
                    {!email.read ? (
                      <Tooltip title="Mark as read">
                        <IconButton onClick={(e) => {
                          e.stopPropagation();
                          api.markEmailRead(email.id).then(fetchData);
                        }} size="small">
                          <MarkReadIcon />
                        </IconButton>
                      </Tooltip>
                    ) : (
                      <Tooltip title="Mark as unread">
                        <IconButton onClick={(e) => {
                          e.stopPropagation();
                          api.markEmailUnread(email.id).then(fetchData);
                        }} size="small">
                          <MarkUnreadIcon />
                        </IconButton>
                      </Tooltip>
                    )}
                    <Tooltip title="Move to trash">
                      <IconButton onClick={(e) => handleDelete(e, email.id)} size="small" color="error">
                        <TrashIcon />
                      </IconButton>
                    </Tooltip>
                    <Tooltip title="Archive">
                      <IconButton onClick={(e) => handleArchive(e, email.id)} size="small" color="success">
                        <ArchiveIcon />
                      </IconButton>
                    </Tooltip>
                    <Tooltip title="Block Sender">
                      <IconButton onClick={(e) => handleBlockSender(e, email)} size="small" color="error">
                        <UnsubscribeIcon />
                      </IconButton>
                    </Tooltip>
                  </TableCell>
                </TableRow>
              ))}
              {emails.length === 0 && !loading && (
                <TableRow>
                  <TableCell colSpan={6} align="center">
                    <Typography color="text.secondary" sx={{ py: 4 }}>
                      No emails found
                    </Typography>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </TableContainer>
      </Card>

      {totalPages > 1 && (
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mt: 3 }}>
          <Typography variant="body2" color="text.secondary">
            Showing {((page - 1) * pageSize) + 1} to {Math.min(page * pageSize, totalEmails)} of {totalEmails} emails
          </Typography>
          <Pagination
            count={totalPages}
            page={page}
            onChange={(event, value) => setPage(value)}
            color="primary"
            size="large"
          />
        </Box>
      )}
    </Container>
  );
}

function PaginatedEmailList({ title, fetcher, actions, onViewEmail, isGmailSource = false, icon, showMarkReadUnread = false, context = '' }) {
  const [emails, setEmails] = React.useState([]);
  const [loading, setLoading] = React.useState(true);
  const [page, setPage] = React.useState(1);
  const [pageSize, setPageSize] = React.useState(20);
  const [totalEmails, setTotalEmails] = React.useState(0);
  const [filter, setFilter] = React.useState('');
  const [selectedEmails, setSelectedEmails] = React.useState(new Set());
  const [bulkActionLoading, setBulkActionLoading] = React.useState(false);

  const fetchData = React.useCallback(() => {
    setLoading(true);
    fetcher(page, pageSize, filter)
      .then(data => {
        const items = data.emails || [];
        const getEmailDate = (email) => {
          try {
            return new Date(isGmailSource ? getHeader(email.payload?.headers, 'Date') : email.date).getTime() || 0;
          } catch (e) {
            return 0;
          }
        };
        const sorted = [...items].sort((a, b) => getEmailDate(b) - getEmailDate(a));
        setEmails(sorted);
        setTotalEmails(data.total || 0);
      })
      .finally(() => setLoading(false));
  }, [fetcher, page, pageSize, filter, isGmailSource]);

  const handlePageSizeChange = (newPageSize) => {
    setPageSize(newPageSize);
    setPage(1); // Reset to first page when changing page size
  };

  React.useEffect(() => { fetchData() }, [fetchData]);

  // Selection management
  const handleSelectAll = () => {
    if (selectedEmails.size === emails.length) {
      setSelectedEmails(new Set());
    } else {
      setSelectedEmails(new Set(emails.map(email => email.id)));
    }
  };

  const handleSelectEmail = (emailId) => {
    const newSelected = new Set(selectedEmails);
    if (newSelected.has(emailId)) {
      newSelected.delete(emailId);
    } else {
      newSelected.add(emailId);
    }
    setSelectedEmails(newSelected);
  };

  // Bulk actions
  const handleBulkMarkRead = () => {
    if (selectedEmails.size === 0) return;
    
    setBulkActionLoading(true);
    api.bulkMarkRead(Array.from(selectedEmails))
      .then(data => {
        setSelectedEmails(new Set());
        fetchData();
      })
      .finally(() => setBulkActionLoading(false));
  };

  const handleBulkMarkUnread = () => {
    if (selectedEmails.size === 0) return;
    
    setBulkActionLoading(true);
    api.bulkMarkUnread(Array.from(selectedEmails))
      .then(data => {
        setSelectedEmails(new Set());
        fetchData();
      })
      .finally(() => setBulkActionLoading(false));
  };

  const handleBulkDelete = () => {
    if (selectedEmails.size === 0) return;
    
    if (!window.confirm(`Are you sure you want to move ${selectedEmails.size} emails to trash?`)) {
      return;
    }
    
    setBulkActionLoading(true);
    api.bulkDelete(Array.from(selectedEmails))
      .then(data => {
        setSelectedEmails(new Set());
        fetchData();
      })
      .finally(() => setBulkActionLoading(false));
  };

  const handleBulkUntrash = () => {
    if (selectedEmails.size === 0) return;
    setBulkActionLoading(true);
    const ids = Array.from(selectedEmails);
    Promise.all(ids.map((id) => api.untrashEmail(id)))
      .then(() => {
        setSelectedEmails(new Set());
        fetchData();
      })
      .finally(() => setBulkActionLoading(false));
  };

  const handleBulkPermanentDelete = () => {
    if (selectedEmails.size === 0) return;
    if (!window.confirm(`Permanently delete ${selectedEmails.size} email(s)? This cannot be undone.`)) return;
    setBulkActionLoading(true);
    const ids = Array.from(selectedEmails);
    Promise.all(ids.map((id) => api.deleteEmailPermanently(id)))
      .then(() => {
        setSelectedEmails(new Set());
        fetchData();
      })
      .finally(() => setBulkActionLoading(false));
  };

  const handleBulkUnarchive = () => {
    if (selectedEmails.size === 0) return;
    setBulkActionLoading(true);
    const ids = Array.from(selectedEmails);
    Promise.all(ids.map((id) => api.unarchiveEmail(id)))
      .then(() => {
        setSelectedEmails(new Set());
        fetchData();
      })
      .finally(() => setBulkActionLoading(false));
  };

  const totalPages = Math.ceil(totalEmails / pageSize);

  if (loading) {
    return (
      <Container maxWidth="lg" sx={{ mt: 2, display: 'flex', justifyContent: 'center' }}>
        <CircularProgress />
      </Container>
    );
  }

  return (
    <Container maxWidth="lg" sx={{ mt: 2, mb: 4 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h4" sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          {icon} {title}
        </Typography>
        <Box sx={{ display: 'flex', gap: 1, alignItems: 'center', flexWrap: 'wrap' }}>
          {selectedEmails.size > 0 && (
            <Box sx={{ display: 'flex', gap: 1, alignItems: 'center', mr: 2 }}>
              <Typography variant="body2" color="text.secondary">
                {selectedEmails.size} selected
              </Typography>
              {showMarkReadUnread && (
                <>
                <Button
                variant="outlined"
                size="small"
                onClick={handleBulkMarkRead}
                disabled={bulkActionLoading}
                startIcon={<MarkReadIcon />}
              >
                Mark Read
              </Button>
              <Button
                variant="outlined"
                size="small"
                onClick={handleBulkMarkUnread}
                disabled={bulkActionLoading}
                startIcon={<MarkUnreadIcon />}
              >
                Mark Unread
              </Button>
              <Button
                variant="outlined"
                size="small"
                onClick={handleBulkDelete}
                disabled={bulkActionLoading}
                startIcon={<TrashIcon />}
                color="error"
              >
                Delete
              </Button>
               </>
              )}
              {context === 'trash' && (
                <>
                  <Button
                    variant="contained"
                    size="small"
                    onClick={handleBulkUntrash}
                    disabled={bulkActionLoading}
                    startIcon={<RestoreIcon />}
                  >
                    Restore
                  </Button>
                  <Button
                    variant="outlined"
                    size="small"
                    onClick={handleBulkPermanentDelete}
                    disabled={bulkActionLoading}
                    startIcon={<DeleteForeverIcon />}
                    color="error"
                  >
                    Delete Forever
                  </Button>
                </>
              )}
              {context === 'archived' && (
                <Button
                  variant="contained"
                  size="small"
                  onClick={handleBulkUnarchive}
                  disabled={bulkActionLoading}
                  startIcon={<InboxIcon />}
                >
                  Move to Inbox
                </Button>
              )}
            </Box>
          )}
          
          <FormControl size="small" sx={{ minWidth: 120 }}>
            <InputLabel>Show</InputLabel>
            <Select
              value={pageSize}
              label="Show"
              onChange={(e) => handlePageSizeChange(e.target.value)}
            >
              <MenuItem value={20}>20 per page</MenuItem>
              <MenuItem value={50}>50 per page</MenuItem>
              <MenuItem value={100}>100 per page</MenuItem>
              <MenuItem value={totalEmails}>Show All</MenuItem>
            </Select>
          </FormControl>
          
          <TextField
            size="small"
            placeholder="Filter by sender or subject..."
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            InputProps={{
              startAdornment: <FilterIcon color="action" sx={{ mr: 1 }} />
            }}
          />
        </Box>
      </Box>
      
      <Card>
        {loading && <LinearProgress />}
        <TableContainer>
          <Table>
            <TableHead sx={{ 
              bgcolor: (theme) => theme.palette.mode === 'dark' ? 'grey.800' : 'grey.50'
            }}>
              <TableRow>
                <TableCell padding="checkbox" sx={{ 
                  fontWeight: 600, 
                  fontSize: '1rem',
                  color: (theme) => theme.palette.mode === 'dark' ? 'text.primary' : 'text.primary'
                }}>
                  <Checkbox
                    indeterminate={selectedEmails.size > 0 && selectedEmails.size < emails.length}
                    checked={emails.length > 0 && selectedEmails.size === emails.length}
                    onChange={handleSelectAll}
                  />
                </TableCell>
                <TableCell width="50" sx={{ 
                  fontWeight: 600, 
                  fontSize: '1rem',
                  color: (theme) => theme.palette.mode === 'dark' ? 'text.primary' : 'text.primary'
                }}></TableCell>
                <TableCell sx={{ 
                  fontWeight: 600, 
                  fontSize: '1rem',
                  color: (theme) => theme.palette.mode === 'dark' ? 'text.primary' : 'text.primary'
                }}>From/Sender</TableCell>
                <TableCell sx={{ 
                  fontWeight: 600, 
                  fontSize: '1rem',
                  color: (theme) => theme.palette.mode === 'dark' ? 'text.primary' : 'text.primary'
                }}>Subject</TableCell>
                <TableCell sx={{ 
                  fontWeight: 600, 
                  fontSize: '1rem',
                  color: (theme) => theme.palette.mode === 'dark' ? 'text.primary' : 'text.primary'
                }}>Date/Time</TableCell>
                <TableCell align="right" sx={{ 
                  fontWeight: 600, 
                  fontSize: '1rem',
                  color: (theme) => theme.palette.mode === 'dark' ? 'text.primary' : 'text.primary'
                }}>Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {emails.map((email) => {
                const unread = isGmailSource ? isUnreadInGmail(email) : isUnreadInDb(email);
                const from = isGmailSource ? getHeader(email.payload?.headers, 'From') : email.sender;
                const subject = isGmailSource ? getHeader(email.payload?.headers, 'Subject') : email.subject;

                return (
                  <TableRow 
                    key={email.id} 
                    hover 
                    onClick={() => onViewEmail(email.id)}
                    sx={{ cursor: 'pointer', fontWeight: unread ? 'bold' : 'normal' }}
                  >
                    <TableCell padding="checkbox" onClick={e => e.stopPropagation()}>
                      <Checkbox
                        checked={selectedEmails.has(email.id)}
                        onChange={() => handleSelectEmail(email.id)}
                      />
                    </TableCell>
                    <TableCell>
                      {unread && <Badge color="primary" variant="dot" />}
                    </TableCell>
                    <TableCell sx={{ fontWeight: unread ? 'bold' : 'normal' }}>
                      {from}
                    </TableCell>
                    <TableCell sx={{ fontWeight: unread ? 'bold' : 'normal' }}>
                      {subject}
                    </TableCell>
                    <TableCell sx={{ fontWeight: unread ? 'bold' : 'normal' }}>
                      {isGmailSource ? 
                        new Date(getHeader(email.payload?.headers, 'Date')).toLocaleString() : 
                        new Date(email.date).toLocaleString()
                      }
                    </TableCell>
                    <TableCell align="right" onClick={e => e.stopPropagation()}>
                      {showMarkReadUnread && (
                        <>
                          {unread ? (
                            <Tooltip title="Mark as read">
                              <IconButton onClick={(e) => {
                                e.stopPropagation();
                                api.markEmailRead(email.id).then(fetchData);
                              }} size="small">
                                <MarkReadIcon />
                              </IconButton>
                            </Tooltip>
                          ) : (
                            <Tooltip title="Mark as unread">
                              <IconButton onClick={(e) => {
                                e.stopPropagation();
                                api.markEmailUnread(email.id).then(fetchData);
                              }} size="small">
                                <MarkUnreadIcon />
                              </IconButton>
                            </Tooltip>
                          )}
                        </>
                      )}
                      {actions(email.id, fetchData)(email)}
                    </TableCell>
                  </TableRow>
                );
              })}
              {emails.length === 0 && (
                <TableRow>
                  <TableCell colSpan={showMarkReadUnread ? 6 : 5} align="center">
                    <Typography color="text.secondary" sx={{ py: 4 }}>
                      No emails found
                    </Typography>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </TableContainer>
      </Card>

      {totalPages > 1 && (
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mt: 3 }}>
          <Typography variant="body2" color="text.secondary">
            Showing {((page - 1) * pageSize) + 1} to {Math.min(page * pageSize, totalEmails)} of {totalEmails} emails
          </Typography>
          <Pagination
            count={totalPages}
            page={page}
            onChange={(event, value) => setPage(value)}
            color="primary"
            size="large"
          />
        </Box>
      )}
    </Container>
  );
}


function Settings() {
  const [settings, setSettings] = React.useState({ automation_enabled: false, automation_frequency: 'daily', automation_time: '09:00', automation_day_of_week: 'sunday' });
  const [quickSyncMs, setQuickSyncMs] = React.useState(() => {
    const v = parseInt(localStorage.getItem('quickSyncMs') || '30000', 10);
    return Number.isFinite(v) ? v : 30000;
  });
  const [loading, setLoading] = React.useState(true);
  const [message, setMessage] = React.useState('');
  const [error, setError] = React.useState('');
  const [confirmReset, setConfirmReset] = React.useState(false);
  const [unsubscribedSenders, setUnsubscribedSenders] = React.useState([]);
  const [loadingUnsubscribed, setLoadingUnsubscribed] = React.useState(false);

  const fetchUnsubscribedSenders = React.useCallback(() => {
    setLoadingUnsubscribed(true);
    api.fetchRules()
      .then(data => {
        // Filter for sender-type DELETE rules (unsubscribe rules)
        const unsubscribed = (data.rules || []).filter(
          rule => rule.type === 'sender' && rule.action === 'DELETE'
        );
        setUnsubscribedSenders(unsubscribed);
      })
      .catch(err => console.error('Failed to fetch unsubscribed senders:', err))
      .finally(() => setLoadingUnsubscribed(false));
  }, []);

  React.useEffect(() => {
    api.fetchSettings()
      .then(setSettings)
      .finally(() => setLoading(false));
    fetchUnsubscribedSenders();
  }, [fetchUnsubscribedSenders]);

  const handleSave = () => {
    setMessage('');
    setError('');
    const payload = {
      automation_enabled: settings.automation_enabled,
      automation_frequency: settings.automation_frequency,
      automation_time: settings.automation_time,
      automation_day_of_week: settings.automation_day_of_week,
    };
    api.saveSettings(payload).then(newSettings => {
      setSettings(newSettings);
      setMessage('Settings saved successfully!');
    }).catch(err => {
      setError(err.message || 'Failed to save settings');
    });
  };

  const handleSaveQuickSync = () => {
    const safe = Math.max(5000, Math.min(10 * 60 * 1000, Number(quickSyncMs) || 30000));
    localStorage.setItem('quickSyncMs', String(safe));
    setQuickSyncMs(safe);
    window.dispatchEvent(new Event('quick-sync-interval-changed'));
    setMessage(`Quick Sync interval updated to ${Math.round(safe / 1000)}s`);
  };

  const handleResetDatabase = () => {
    setMessage('');
    setError('');
    api.resetDatabase()
      .then(res => {
        setMessage(res.message);
        setConfirmReset(false);
      })
      .catch(err => {
        setError(err.message || 'Failed to reset database.');
        setConfirmReset(false);
      });
  };

  const handleUnblockSender = (ruleId, sender) => {
    if (!window.confirm(`Unblock "${sender}"? The auto-delete rule will be removed.`)) {
      return;
    }
    api.deleteRule(ruleId)
      .then(() => {
        setMessage(`Unblocked ${sender}. The rule has been removed.`);
        fetchUnsubscribedSenders();
      })
      .catch(err => {
        setError(`Failed to unblock: ${err.message}`);
      });
  };

  if (loading) {
    return (
      <Container maxWidth="lg" sx={{ mt: 2, display: 'flex', justifyContent: 'center' }}>
        <CircularProgress />
      </Container>
    );
  }

  return (
    <Container maxWidth="lg" sx={{ mt: 4, mb: 6, display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
      <Typography variant="h4" gutterBottom sx={{ 
        display: 'flex', 
        alignItems: 'center', 
        gap: 1,
        fontWeight: 700,
        background: 'linear-gradient(45deg, #1976d2, #42a5f5)',
        backgroundClip: 'text',
        WebkitBackgroundClip: 'text',
        WebkitTextFillColor: 'transparent',
        mb: 4
      }}>
        <SettingsIcon /> Settings & Configuration
      </Typography>

      {message && (
        <Alert severity="success" sx={{ mb: 2 }} onClose={() => setMessage('')}>
          {message}
        </Alert>
      )}
      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError('')}>
          {error}
        </Alert>
      )}

      <Grid container spacing={4} sx={{ justifyContent: 'center' }}>
        <Grid item xs={12} sx={{ width: '100%', maxWidth: 600 }}>
          <Card sx={{ 
            width: '100%',
            borderRadius: 3,
            boxShadow: '0 8px 32px rgba(0,0,0,0.1)'
          }}>
            <CardHeader 
              title="Automation Settings" 
              sx={{ textAlign: 'center' }}
              titleTypographyProps={{ 
                variant: 'h6', 
                fontWeight: 600,
                color: 'primary.main'
              }}
            />
            <CardContent sx={{ p: 4, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 3 }}>
              <FormControlLabel
                control={
                  <Switch
                    checked={settings.automation_enabled}
                    onChange={(e) => setSettings(s => ({ ...s, automation_enabled: e.target.checked }))}
                    sx={{ transform: 'scale(1.2)' }}
                  />
                }
                label={
                  <Typography variant="h6" sx={{ fontWeight: 500, textAlign: 'center' }}>
                    Enable Automated Background Cleaning
                  </Typography>
                }
                sx={{ mb: 1, alignSelf: 'center' }}
              />
              <Grid
                container
                spacing={2}
                sx={{ mt: 1, width: '100%', maxWidth: 400, justifyContent: 'center' }}
              >
                <Grid item xs={12}>
                  <FormControl fullWidth>
                    <InputLabel>Frequency</InputLabel>
                    <Select
                      label="Frequency"
                      value={settings.automation_frequency}
                      onChange={(e) => setSettings(s => ({ ...s, automation_frequency: e.target.value }))}
                      disabled={!settings.automation_enabled}
                    >
                      <MenuItem value="daily">Daily</MenuItem>
                      <MenuItem value="weekly">Weekly</MenuItem>
                    </Select>
                  </FormControl>
                </Grid>
                {settings.automation_frequency === 'weekly' && (
                  <Grid item xs={12}>
                    <FormControl fullWidth>
                      <InputLabel>Day of the Week</InputLabel>
                      <Select
                        label="Day of the Week"
                        value={settings.automation_day_of_week}
                        onChange={(e) => setSettings(s => ({ ...s, automation_day_of_week: e.target.value }))}
                        disabled={!settings.automation_enabled}
                        sx={{ minHeight: 56, minWidth: 160 }}
                        MenuProps={{
                          PaperProps: {
                            sx: {
                              maxHeight: 300,
                              '& .MuiMenuItem-root': {
                                fontSize: '1rem',
                                padding: '12px 16px'
                              }
                            }
                          }
                        }}
                      >
                        <MenuItem value="sunday">Sunday</MenuItem>
                        <MenuItem value="monday">Monday</MenuItem>
                        <MenuItem value="tuesday">Tuesday</MenuItem>
                        <MenuItem value="wednesday">Wednesday</MenuItem>
                        <MenuItem value="thursday">Thursday</MenuItem>
                        <MenuItem value="friday">Friday</MenuItem>
                        <MenuItem value="saturday">Saturday</MenuItem>
                      </Select>
                    </FormControl>
                  </Grid>
                )}
                <Grid item xs={12}>
                  <TextField
                    fullWidth
                    label="Time of Day"
                    type="time"
                    value={settings.automation_time || '09:00'}
                    onChange={(e) => setSettings(s => ({ ...s, automation_time: e.target.value }))}
                    disabled={!settings.automation_enabled}
                    InputLabelProps={{ shrink: true }}
                  />
                </Grid>
              </Grid>
            </CardContent>
            <CardActions sx={{ justifyContent: 'center', p: 3 }}>
              <Button 
                variant="contained" 
                onClick={handleSave}
                sx={{
                  borderRadius: 2,
                  px: 4,
                  py: 1.5,
                  fontWeight: 600,
                  boxShadow: '0 4px 16px rgba(25, 118, 210, 0.3)'
                }}
              >
                Save Settings
              </Button>
            </CardActions>
          </Card>
        </Grid>

        <Grid item xs={12} sx={{ width: '100%', maxWidth: 600 }}>
          <Card sx={{ width: '100%', borderRadius: 3, boxShadow: '0 8px 32px rgba(0,0,0,0.1)' }}>
            <CardHeader 
              title="Quick Sync Interval" 
              sx={{ textAlign: 'center' }}
            />
            <CardContent sx={{ p: 3, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 2, textAlign: 'center' }}>
              <Typography variant="body2" color="text.secondary">
                Choose how often the app runs Quick Sync in the background while you're on the Inbox.
              </Typography>
              <FormControl fullWidth sx={{ width: '100%', maxWidth: 300 }}>
                <InputLabel>Interval</InputLabel>
                <Select
                  label="Interval"
                  value={quickSyncMs}
                  onChange={(e) => setQuickSyncMs(Number(e.target.value))}
                >
                  <MenuItem value={5000}>5 seconds</MenuItem>
                  <MenuItem value={10000}>10 seconds</MenuItem>
                  <MenuItem value={30000}>30 seconds</MenuItem>
                  <MenuItem value={60000}>1 minute</MenuItem>
                  <MenuItem value={120000}>2 minutes</MenuItem>
                  <MenuItem value={300000}>5 minutes</MenuItem>
                </Select>
              </FormControl>
            </CardContent>
            <CardActions sx={{ justifyContent: 'center', p: 2 }}>
              <Button variant="outlined" onClick={handleSaveQuickSync}>
                Update Interval
              </Button>
            </CardActions>
          </Card>
        </Grid>

        <Grid item xs={12} sx={{ width: '100%', maxWidth: 600 }}>
          <Card sx={{ 
            width: '100%', 
            borderRadius: 3, 
            boxShadow: '0 8px 32px rgba(0,0,0,0.1)',
            border: '2px solid',
            borderColor: 'warning.main'
          }}>
            <CardHeader 
              title="Manage Blocked Senders" 
              avatar={<UnsubscribeIcon color="error" />}
              sx={{ textAlign: 'center' }}
              titleTypographyProps={{ 
                variant: 'h6', 
                fontWeight: 600,
                color: 'error.main'
              }}
              subheader={`${unsubscribedSenders.length} sender${unsubscribedSenders.length !== 1 ? 's' : ''} blocked`}
            />
            <CardContent sx={{ p: 3 }}>
              {loadingUnsubscribed ? (
                <Box sx={{ display: 'flex', justifyContent: 'center', py: 3 }}>
                  <CircularProgress size={40} />
                </Box>
              ) : unsubscribedSenders.length > 0 ? (
                <List sx={{ width: '100%' }}>
                  {unsubscribedSenders.map((rule, index) => (
                    <React.Fragment key={rule.id}>
                      {index > 0 && <Divider />}
                      <ListItem
                        secondaryAction={
                          <Button
                            variant="outlined"
                            size="small"
                            color="success"
                            startIcon={<AddIcon />}
                            onClick={() => handleUnblockSender(rule.id, rule.value)}
                          >
                            Unblock
                          </Button>
                        }
                        sx={{ py: 2 }}
                      >
                        <ListItemIcon>
                          <MailIcon color="warning" />
                        </ListItemIcon>
                        <ListItemText
                          primary={
                            <Typography variant="body1" sx={{ fontWeight: 500 }}>
                              {rule.value}
                            </Typography>
                          }
                          secondary={
                            <Typography variant="caption" color="text.secondary">
                              Emails automatically moved to trash
                              {rule.age_days > 0 && ` • Older than ${rule.age_days} days`}
                            </Typography>
                          }
                        />
                      </ListItem>
                    </React.Fragment>
                  ))}
                </List>
              ) : (
                <Box sx={{ textAlign: 'center', py: 4 }}>
                  <UnsubscribeIcon sx={{ fontSize: 64, color: 'grey.300', mb: 2 }} />
                  <Typography variant="h6" color="text.secondary" gutterBottom>
                    No Blocked Senders
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    You haven't blocked any senders yet.
                  </Typography>
                </Box>
              )}
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sx={{ width: '100%', maxWidth: 600 }}>
          <Card sx={{ 
            width: '100%',
            border: '2px solid', 
            borderColor: 'error.main',
            borderRadius: 3,
            boxShadow: '0 8px 32px rgba(244, 67, 54, 0.1)'
          }}>
            <Box sx={{ display: 'flex', justifyContent: 'center', pt: 3 }}>
              <WarningIcon color="error" sx={{ fontSize: 32 }} />
            </Box>
            <CardHeader
              title="Danger Zone" 
              titleTypographyProps={{ 
                color: 'error.main',
                variant: 'h6',
                fontWeight: 600
              }}
              sx={{ textAlign: 'center' }}
            />
            <CardContent sx={{ p: 3, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 2, textAlign: 'center' }}>
              <Typography variant="body1" color="text.secondary" paragraph sx={{ fontSize: '1rem', lineHeight: 1.6 }}>
                Resetting the database will clear all your synced emails, rules, and settings. 
                This cannot be undone.
              </Typography>
            </CardContent>
            <CardActions sx={{ justifyContent: 'center', p: 3 }}>
              <Button 
                variant="outlined" 
                color="error" 
                onClick={() => setConfirmReset(true)}
                sx={{
                  borderRadius: 2,
                  px: 3,
                  py: 1.5,
                  fontWeight: 600
                }}
              >
                Reset Database
              </Button>
            </CardActions>
          </Card>
        </Grid>
      </Grid>

      {/* Reset Confirmation Dialog */}
      <Dialog open={confirmReset} onClose={() => setConfirmReset(false)}>
        <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <WarningIcon color="error" />
          Confirm Database Reset
        </DialogTitle>
        <DialogContent>
          <Typography>
            Are you sure you want to reset the database? All emails, rules, and settings will be 
            permanently deleted. This action cannot be undone.
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setConfirmReset(false)}>Cancel</Button>
          <Button onClick={handleResetDatabase} color="error" variant="contained">
            Reset Database
          </Button>
        </DialogActions>
      </Dialog>
    </Container>
  );
}

function Navbar({ tab, setTab, onLogout, theme, setTheme, onInboxClick }) {
  const tabs = [
    { id: 'dashboard', label: 'Dashboard', icon: <DashboardIcon /> },
    { id: 'paginated', label: 'Inbox', icon: <InboxIcon /> },
    { id: 'rules', label: 'Rules', icon: <RuleIcon /> },
    { id: 'clean', label: 'Clean', icon: <CleanIcon /> },
    { id: 'trash', label: 'Trash', icon: <TrashIcon /> },
    { id: 'archived', label: 'Archived', icon: <ArchiveIcon /> },
    { id: 'settings', label: 'Settings', icon: <SettingsIcon /> }
  ];

  const handleTabChange = (e, newValue) => {
    setTab(newValue);
    // Trigger quick sync when Inbox tab is clicked
    if (newValue === 'paginated' && onInboxClick) {
      // Small delay to ensure component is mounted
      setTimeout(() => onInboxClick(), 100);
    }
  };

  return (
    <AppBar position="sticky" elevation={1}>
      <Container maxWidth="xl">
        <Toolbar sx={{ justifyContent: 'space-between' }}>
          <Typography variant="h6" component="div" sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            🧹 MailCleaner
          </Typography>
          
          <Box sx={{ display: { xs: 'none', md: 'flex' } }}>
            <Tabs 
              value={tab} 
              onChange={handleTabChange}
              textColor="inherit"
              indicatorColor="secondary"
            >
              {tabs.map((tabItem) => (
                <Tab
                  key={tabItem.id}
                  value={tabItem.id}
                  label={tabItem.label}
                  icon={tabItem.icon}
                  iconPosition="start"
                  sx={{ minHeight: 64 }}
                />
              ))}
            </Tabs>
          </Box>

          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <IconButton 
              color="inherit" 
              onClick={() => setTheme(theme === 'light' ? 'dark' : 'light')}
            >
              {theme === 'dark' ? <LightMode /> : <DarkMode />}
            </IconButton>
            <Button
              color="inherit"
              startIcon={<LogoutIcon />}
              onClick={onLogout}
            >
              Logout
            </Button>
          </Box>
        </Toolbar>
      </Container>
    </AppBar>
  );
}

function App() {
  // 🌙 Dark mode state with persistence
  const [tab, setTab] = React.useState('dashboard');
  const [viewingEmailId, setViewingEmailId] = React.useState(null);
  const [authenticated, setAuthenticated] = useAuth();
  const [theme, setTheme] = React.useState(() => localStorage.getItem('theme') || 'dark');
  const [undoInfo, setUndoInfo] = React.useState({ show: false, message: '', id: null });
  const [page, setPage] = React.useState(1);
  const quickSyncRef = React.useRef(null);

  // Create MUI theme object based on state
  const muiTheme = React.useMemo(() => createTheme({
    palette: {
      mode: theme,
      ...(theme === 'dark'
        ? {
            background: {
              default: '#181a20',
              paper: '#23272f'
            }
          }
        : {
            background: {
              default: '#f5f6fa',
              paper: '#fff'
            }
          })
    }
  }), [theme]);

  const handleLogout = async () => {
    try {
      await api.logout();
    } catch (e) {
      console.error("Logout failed:", e);
    }
    setAuthenticated(false);
  };

  React.useEffect(() => {
    localStorage.setItem('theme', theme);
  }, [theme]);

  const showUndoToast = (message, id) => {
    setUndoInfo({ show: true, message, id });
    setTimeout(() => setUndoInfo({ show: false, message: '', id: null }), 5000);
  };

  const handleUndo = async () => {
    if (undoInfo.id) {
      await api.untrashEmail(undoInfo.id);
      setUndoInfo({ show: false, message: '', id: null });
    } else {
      setUndoInfo({ show: false, message: '', id: null });
    }
  };

  const handleInboxClick = () => {
    if (quickSyncRef.current) {
      quickSyncRef.current();
    }
  };

  const renderContent = () => {
    switch (tab) {
      case 'dashboard': 
        return <Dashboard />;
      case 'rules': 
        return <Rules />;
      case 'clean': 
        return <Clean showUndoToast={showUndoToast} />;
      case 'paginated': 
        return <PaginatedEmails page={page} setPage={setPage} showUndoToast={showUndoToast} onViewEmail={setViewingEmailId} triggerQuickSync={quickSyncRef} />;
      case 'trash':
        return <PaginatedEmailList
          title="Recycle Bin (Trash)"
          fetcher={api.fetchTrashEmails}
          onViewEmail={setViewingEmailId}
          isGmailSource={true}
          icon={<TrashIcon />}
          showMarkReadUnread={false}
          context="trash"
          actions={(id, fetchData) => (email) => {
            const from = getHeader(email.payload?.headers, 'From');
            return (
              <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-end' }}>
                <Tooltip title="Restore">
                  <IconButton onClick={(e) => {
                    e.stopPropagation();
                    api.untrashEmail(id).then(fetchData);
                  }} size="small" color="primary">
                    <RestoreIcon />
                  </IconButton>
                </Tooltip>
                <Tooltip title="Delete forever">
                  <IconButton 
                    onClick={(e) => {
                      e.stopPropagation();
                      if (window.confirm('Delete forever?')) api.deleteEmailPermanently(id).then(fetchData);
                    }} 
                    size="small" 
                    color="error"
                  >
                    <DeleteForeverIcon />
                  </IconButton>
                </Tooltip>
                <Tooltip title="Block Sender">
                  <IconButton onClick={(e) => {
                    e.stopPropagation();
                    if (window.confirm(`Block "${from}"? Future emails from this sender will be automatically moved to trash.`)) {
                      api.blockSender(from)
                        .then(() => showUndoToast(`Blocked ${from}. A rule has been created.`))
                        .catch(err => showUndoToast(`Failed to block: ${err.message}`));
                    }
                  }} size="small" color="error">
                    <UnsubscribeIcon />
                  </IconButton>
                </Tooltip>
              </Box>
            );
          }}
        />;
      case 'archived':
        return <PaginatedEmailList
          title="Archived Mail"
          fetcher={api.fetchArchivedEmails}
          onViewEmail={setViewingEmailId}
          isGmailSource={true}
          icon={<ArchiveIcon />}
          showMarkReadUnread={true}
          context="archived"
          actions={(id, fetchData) => (email) => {
            const from = getHeader(email.payload?.headers, 'From');
            return (
              <>
                <Tooltip title="Move to Trash">
                  <IconButton onClick={(e) => {
                    e.stopPropagation();
                    api.deleteEmail(id).then(fetchData);
                  }} size="small" color="error">
                    <TrashIcon />
                  </IconButton>
                </Tooltip>
                <Tooltip title="Move to Inbox">
                  <IconButton onClick={(e) => {
                    e.stopPropagation();
                    api.unarchiveEmail(id).then(fetchData);
                  }} size="small" color="info">
                    <InboxIcon />
                  </IconButton>
                </Tooltip>
                <Tooltip title="Block Sender">
                  <IconButton onClick={(e) => {
                    e.stopPropagation();
                    if (window.confirm(`Block "${from}"? Future emails from this sender will be automatically moved to trash.`)) {
                      api.blockSender(from)
                        .then(() => showUndoToast(`Blocked ${from}. A rule has been created.`))
                        .catch(err => showUndoToast(`Failed to block: ${err.message}`));
                    }
                  }} size="small" color="error">
                    <UnsubscribeIcon />
                  </IconButton>
                </Tooltip>
              </>
            );
          }}
        />;
      case 'settings': 
        return <Settings />;
      default: 
        return <Dashboard />;
    }
  };

  if (authenticated === null) {
    return (
      <ThemeProvider theme={muiTheme}>
        <CssBaseline />
        <Container maxWidth="sm" sx={{ mt: 8, display: 'flex', justifyContent: 'center' }}>
          <CircularProgress size={60} />
        </Container>
      </ThemeProvider>
    );
  }

  if (!authenticated) {
    return (
      <ThemeProvider theme={muiTheme}>
        <CssBaseline />
        <Login />
      </ThemeProvider>
    );
  }

  return (
    <ThemeProvider theme={muiTheme}>
      <CssBaseline />
      <Box sx={{ flexGrow: 1, minHeight: '100vh', bgcolor: 'background.default' }}>
        <Navbar tab={tab} setTab={setTab} onLogout={handleLogout} theme={theme} setTheme={setTheme} onInboxClick={handleInboxClick} />
        
        <Box component="main" sx={{ pb: 4 }}>
          {renderContent()}
        </Box>

        {/* Email Detail Modal */}
        <EmailDetailModal 
          emailId={viewingEmailId} 
          onClose={() => setViewingEmailId(null)} 
        />

        {/* Undo Snackbar */}
        <Snackbar
          open={undoInfo.show}
          autoHideDuration={5000}
          onClose={() => setUndoInfo({ show: false, message: '', id: null })}
          message={undoInfo.message}
          action={
            undoInfo.id && (
              <Button color="inherit" size="small" onClick={handleUndo} startIcon={<UndoIcon />}>
                Undo
              </Button>
            )
          }
        />

        {/* Mobile FAB for adding rules */}
        {tab === 'rules' && (
          <Fab
            color="primary"
            aria-label="add rule"
            sx={{
              position: 'fixed',
              bottom: 16,
              right: 16,
              display: { xs: 'flex', md: 'none' }
            }}
            onClick={() => {/* This would need to trigger rule dialog */}}
          >
            <AddIcon />
          </Fab>
        )}
        
      </Box>
    </ThemeProvider>
  );
}

export default App;