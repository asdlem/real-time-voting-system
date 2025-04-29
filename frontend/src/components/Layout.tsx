import React from 'react';
import { Outlet, Link as RouterLink } from 'react-router-dom';
import { 
  AppBar, 
  Box, 
  Toolbar, 
  Typography, 
  Container, 
  Button,
  Link,
  IconButton,
  Drawer,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  useTheme,
  useMediaQuery
} from '@mui/material';
import { 
  Menu as MenuIcon, 
  Home as HomeIcon, 
  Add as AddIcon, 
  Speed as SpeedIcon,
  Close as CloseIcon
} from '@mui/icons-material';

const Layout: React.FC = () => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('md'));
  const [drawerOpen, setDrawerOpen] = React.useState(false);
  
  const toggleDrawer = () => {
    setDrawerOpen(!drawerOpen);
  };
  
  const menuItems = [
    { text: '投票列表', icon: <HomeIcon />, path: '/' },
    { text: '创建投票', icon: <AddIcon />, path: '/polls/create' },
    { text: '性能测试', icon: <SpeedIcon />, path: '/performance-test' }
  ];

  const drawer = (
    <Box sx={{ width: 250 }} role="presentation">
      <Box sx={{ 
        display: 'flex', 
        justifyContent: 'space-between', 
        alignItems: 'center',
        p: 2
      }}>
        <Typography variant="h6" component="div">
          投票应用
        </Typography>
        <IconButton onClick={toggleDrawer}>
          <CloseIcon />
        </IconButton>
      </Box>
      
      <List>
        {menuItems.map((item) => (
          <ListItem 
            key={item.text} 
            component={RouterLink} 
            to={item.path}
            onClick={toggleDrawer}
            sx={{ py: 1.5 }}
          >
            <ListItemIcon>{item.icon}</ListItemIcon>
            <ListItemText primary={item.text} />
          </ListItem>
        ))}
      </List>
    </Box>
  );

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', minHeight: '100vh' }}>
      <AppBar 
        position="static" 
        color="default" 
        elevation={0}
        sx={{ 
          borderBottom: '1px solid',
          borderColor: 'divider',
          bgcolor: 'rgba(255, 255, 255, 0.9)',
          backdropFilter: 'blur(12px)',
          boxShadow: '0 1px 10px rgba(0,0,0,0.06)'
        }}
      >
        <Toolbar>
          {isMobile ? (
            <>
              <IconButton
                color="inherit"
                aria-label="open drawer"
                edge="start"
                onClick={toggleDrawer}
                sx={{ mr: 2 }}
              >
                <MenuIcon color="primary" />
              </IconButton>
              <Typography 
                variant="h6" 
                component="div" 
                sx={{ 
                  flexGrow: 1,
                  fontWeight: 600,
                  backgroundImage: 'linear-gradient(45deg, #3367D6, #4285F4)',
                  backgroundClip: 'text',
                  WebkitBackgroundClip: 'text',
                  color: 'transparent',
                }}
              >
                投票应用
              </Typography>
            </>
          ) : (
            <>
              <Typography
                variant="h6"
                component={RouterLink}
                to="/"
                sx={{
                  mr: 4,
                  fontWeight: 700,
                  textDecoration: 'none',
                  display: 'flex',
                  alignItems: 'center',
                  backgroundImage: 'linear-gradient(45deg, #3367D6, #4285F4)',
                  backgroundClip: 'text',
                  WebkitBackgroundClip: 'text',
                  color: 'transparent',
                }}
              >
                投票应用
              </Typography>
              
              <Box sx={{ display: 'flex', flexGrow: 1 }}>
                {menuItems.map((item) => (
                  <Link
                    key={item.text}
                    component={RouterLink}
                    to={item.path}
                    color="inherit"
                    underline="none"
                    sx={{
                      mx: 2,
                      display: 'flex',
                      alignItems: 'center',
                      fontWeight: 500,
                      color: theme.palette.text.secondary,
                      transition: 'all 0.2s ease',
                      '&:hover': {
                        color: theme.palette.primary.main,
                        transform: 'translateY(-2px)',
                      },
                    }}
                  >
                    <Box 
                      component="span" 
                      sx={{ 
                        mr: 0.5, 
                        display: 'flex',
                        color: theme.palette.primary.main,
                      }}
                    >
                      {item.icon}
                    </Box>
                    {item.text}
                  </Link>
                ))}
              </Box>
            </>
          )}
          <Button 
            component={RouterLink} 
            to="/polls/create" 
            variant="contained" 
            color="primary"
            startIcon={<AddIcon />}
            sx={{
              borderRadius: '20px',
              textTransform: 'none',
              py: 1,
              px: 2.5,
              fontWeight: 600,
              boxShadow: '0 4px 10px rgba(66, 133, 244, 0.25)',
              background: 'linear-gradient(45deg, #4285F4, #5E97F6)',
              '&:hover': {
                boxShadow: '0 6px 15px rgba(66, 133, 244, 0.35)',
                background: 'linear-gradient(45deg, #3367D6, #4285F4)',
              }
            }}
          >
            创建投票
          </Button>
        </Toolbar>
      </AppBar>
      
      <Drawer
        anchor="left"
        open={drawerOpen}
        onClose={toggleDrawer}
        PaperProps={{
          sx: {
            width: 250,
            background: 'linear-gradient(to bottom, rgba(255,255,255,0.98), rgba(250,252,255,0.95))',
            borderRight: '1px solid rgba(0, 0, 0, 0.05)',
            boxShadow: '5px 0 15px rgba(0, 0, 0, 0.08)',
          }
        }}
      >
        {drawer}
      </Drawer>
      
      <Box component="main" sx={{ flexGrow: 1, py: 3 }}>
        <Container>
          <Outlet />
        </Container>
      </Box>
      
      <Box
        component="footer"
        sx={{
          py: 3,
          px: 2,
          mt: 'auto',
          background: 'linear-gradient(to top, rgba(245,245,247,0.95), rgba(250,250,252,0.95))',
          backdropFilter: 'blur(8px)',
          borderTop: '1px solid',
          borderColor: 'divider',
        }}
      >
        <Container maxWidth="lg">
          <Typography 
            variant="body2" 
            color="text.secondary" 
            align="center"
            sx={{
              fontWeight: 500,
            }}
          >
            {'高并发投票应用 © '}
            {new Date().getFullYear()}
          </Typography>
        </Container>
      </Box>
    </Box>
  );
};

export default Layout; 