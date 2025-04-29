import React from 'react';
import { Routes, Route, BrowserRouter, Navigate } from 'react-router-dom';
import { CssBaseline, ThemeProvider, createTheme } from '@mui/material';
import PollList from './pages/PollList';
import PollDetail from './pages/PollDetail';
import CreatePoll from './pages/CreatePoll';
import PerformanceTest from './pages/PerformanceTest';
import Layout from './components/Layout';

// 创建一个结合苹果和谷歌风格的主题
const theme = createTheme({
  palette: {
    primary: {
      main: '#4285F4', // Google 蓝色
      light: '#5E97F6',
      dark: '#3367D6',
    },
    secondary: {
      main: '#34A853', // Google 绿色
      light: '#69C077',
      dark: '#2E7D32',
    },
    error: {
      main: '#EA4335', // Google 红色
    },
    warning: {
      main: '#FBBC04', // Google 黄色
    },
    info: {
      main: '#4285F4', // Google 蓝色
      light: '#5E97F6',
    },
    success: {
      main: '#34A853', // Google 绿色
    },
    background: {
      default: '#F5F5F7', // Apple 灰色背景
      paper: '#FFFFFF',
    },
    text: {
      primary: '#1D1D1F', // Apple 文本颜色
      secondary: '#6E6E73', // Apple 次要文本
    },
  },
  typography: {
    fontFamily: [
      '-apple-system',
      'BlinkMacSystemFont',
      'Segoe UI',
      'Roboto',
      'Helvetica Neue',
      'Arial',
      'sans-serif',
    ].join(','),
    h4: {
      fontWeight: 600,
      letterSpacing: '-0.5px',
    },
    h5: {
      fontWeight: 600,
      letterSpacing: '-0.3px',
    },
    h6: {
      fontWeight: 600,
      letterSpacing: '-0.2px',
    },
    button: {
      textTransform: 'none',
      fontWeight: 500,
    },
  },
  shape: {
    borderRadius: 12,
  },
  components: {
    MuiButton: {
      styleOverrides: {
        root: {
          textTransform: 'none',
          borderRadius: 20,
          padding: '8px 18px',
          fontWeight: 500,
          fontSize: '0.9375rem',
        },
        containedPrimary: {
          boxShadow: '0px 2px 8px rgba(66, 133, 244, 0.2)',
          '&:hover': {
            boxShadow: '0px 4px 12px rgba(66, 133, 244, 0.3)',
          },
        },
      },
    },
    MuiCard: {
      styleOverrides: {
        root: {
          borderRadius: 16,
          boxShadow: '0px 2px 12px rgba(0, 0, 0, 0.06)',
        },
      },
    },
    MuiTextField: {
      styleOverrides: {
        root: {
          '& .MuiOutlinedInput-root': {
            borderRadius: 12,
          },
        },
      },
    },
    MuiPaper: {
      styleOverrides: {
        root: {
          backgroundImage: 'none',
        },
      },
    },
    MuiChip: {
      styleOverrides: {
        root: {
          fontWeight: 500,
        },
      },
    },
  },
});

function App() {
  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<Layout />}>
            <Route index element={<PollList />} />
            <Route path="poll/:id" element={<PollDetail />} />
            <Route path="polls/create" element={<CreatePoll />} />
            <Route path="polls/:id" element={<PollDetail />} />
            <Route path="create" element={<CreatePoll />} />
            <Route path="performance-test" element={<PerformanceTest />} />
            <Route path="*" element={<Navigate to="/" />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </ThemeProvider>
  );
}

export default App; 