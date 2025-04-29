import React from 'react';
import ReactDOM from 'react-dom/client';
import './index.css';
import App from './App';
import { BrowserRouter } from 'react-router-dom';
import { ThemeProvider, createTheme } from '@mui/material/styles';
import reportWebVitals from './reportWebVitals';

// 输出运行环境信息，便于调试
console.log('[App] 启动前端应用');
console.log('[App] 环境变量:', {
  NODE_ENV: process.env.NODE_ENV,
  PUBLIC_URL: process.env.PUBLIC_URL,
  REACT_APP_API_BASE_URL: process.env.REACT_APP_API_BASE_URL,
});
console.log('[App] 当前URL:', window.location.href);

const theme = createTheme({
  palette: {
    primary: {
      main: '#4285F4',
      light: '#5E97F6',
      dark: '#3367D6',
    },
    secondary: {
      main: '#34A853',
    },
    error: {
      main: '#EA4335',
    },
    warning: {
      main: '#FBBC05',
    },
    info: {
      main: '#4285F4',
    },
    success: {
      main: '#34A853',
    },
  },
  typography: {
    fontFamily: [
      '-apple-system',
      'BlinkMacSystemFont',
      '"Segoe UI"',
      'Roboto',
      '"Helvetica Neue"',
      'Arial',
      'sans-serif',
      '"Apple Color Emoji"',
      '"Segoe UI Emoji"',
      '"Segoe UI Symbol"',
    ].join(','),
  },
  components: {
    MuiButton: {
      styleOverrides: {
        root: {
          textTransform: 'none',
        },
      },
    },
  },
});

const root = ReactDOM.createRoot(
  document.getElementById('root') as HTMLElement
);

// 去掉严格模式，防止组件渲染两次导致useRef错误
root.render(
  <ThemeProvider theme={theme}>
    <BrowserRouter>
      <App />
    </BrowserRouter>
  </ThemeProvider>
);

// If you want to start measuring performance in your app, pass a function
// to log results (for example: reportWebVitals(console.log))
// or send to an analytics endpoint. Learn more: https://bit.ly/CRA-vitals
reportWebVitals(); 