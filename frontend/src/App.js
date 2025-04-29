import React from 'react';
import { Routes, Route } from 'react-router-dom';
import { ConfigProvider } from 'antd';
import zhCN from 'antd/lib/locale/zh_CN';
import PollList from './components/PollList';
import PollDetail from './components/PollDetail';
import CreatePoll from './components/CreatePoll';
import EditPoll from './components/EditPoll';
import './App.css';

function App() {
  console.log('[App] 渲染App组件');
  
  return (
    <ConfigProvider locale={zhCN}>
      <div className="app-container">
        <Routes>
          <Route path="/" element={<PollList />} />
          <Route path="/polls/:id" element={<PollDetail />} />
          <Route path="/poll/:id" element={<PollDetail />} />
          <Route path="/poll/:id/edit" element={<EditPoll />} />
          <Route path="/create" element={<CreatePoll />} />
          {/* 捕获其他所有路径，重定向到主页 */}
          <Route path="*" element={<PollList />} />
        </Routes>
      </div>
    </ConfigProvider>
  );
}

export default App; 