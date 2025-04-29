import axios from 'axios';

// API基础URL
const API_BASE_URL = 'http://localhost:8090/api';
const WS_BASE_URL = 'ws://localhost:8090/ws';

// 创建axios实例
const apiClient = axios.create({
  baseURL: API_BASE_URL,
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// 投票系统API服务
const pollService = {
  // 创建投票
  createPoll: async (pollData) => {
    try {
      console.log('发送创建投票请求:', pollData);
      
      // 确保字段名与后端一致
      const normalizedPollData = {
        ...pollData,
        // 确保大小写字段名都能正确工作
        question: pollData.Question || pollData.question,
        description: pollData.Description || pollData.description,
        poll_type: pollData.PollType !== undefined ? pollData.PollType : pollData.poll_type, // 确保使用小写的poll_type
        options: pollData.Options || pollData.options
      };
      
      console.log('规范化后的请求数据:', normalizedPollData);
      const response = await apiClient.post('/polls', normalizedPollData);
      return response.data;
    } catch (error) {
      console.error('创建投票错误:', error.response?.data || error.message);
      throw handleApiError(error);
    }
  },

  // 获取投票列表
  getPolls: async () => {
    try {
      const response = await apiClient.get('/polls');
      return response.data;
    } catch (error) {
      console.error('获取投票列表错误:', error.response?.data || error.message);
      throw handleApiError(error);
    }
  },

  // 获取投票详情
  getPollById: async (pollId) => {
    try {
      if (!pollId || isNaN(parseInt(pollId))) {
        throw new Error('无效的投票ID');
      }
      
      const response = await apiClient.get(`/polls/${pollId}`);
      return response.data;
    } catch (error) {
      console.error('获取投票详情错误:', error.response?.data || error.message);
      throw handleApiError(error);
    }
  },

  // 提交投票
  submitVote: async (pollId, voteData) => {
    try {
      if (!pollId || isNaN(parseInt(pollId))) {
        throw new Error('无效的投票ID');
      }
      
      // 确保voteData格式正确
      let processedVoteData = voteData;
      
      // 如果传入的是数组，转换为正确的格式
      if (Array.isArray(voteData)) {
        processedVoteData = { option_ids: voteData };
      } 
      // 如果voteData包含option_id字段（单数），转换为option_ids（复数）
      else if (voteData.option_id !== undefined) {
        processedVoteData = { 
          option_ids: Array.isArray(voteData.option_id) ? voteData.option_id : [voteData.option_id]
        };
      }
      // 如果没有任何选项ID，抛出错误
      else if (!voteData.option_ids || !Array.isArray(voteData.option_ids) || voteData.option_ids.length === 0) {
        throw new Error('请选择至少一个选项');
      }
      
      console.log(`提交投票 pollId=${pollId}:`, processedVoteData);
      const response = await apiClient.post(`/polls/${pollId}/vote`, processedVoteData);
      return response.data;
    } catch (error) {
      console.error('提交投票错误:', error.response?.data || error.message);
      throw handleApiError(error);
    }
  },

  // 创建WebSocket连接
  createWebSocketConnection: (pollId) => {
    // 检查环境和参数
    if (!pollId) {
      console.error('创建WebSocket连接失败: 缺少投票ID');
      return null;
    }

    try {
      // 建立WebSocket连接 - 确保使用正确的协议
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const host = process.env.REACT_APP_API_HOST || window.location.hostname;
      const port = process.env.REACT_APP_API_PORT || '8090';
      
      // 构建WebSocket URL，并携带keepalive参数以防止服务器自动关闭
      const wsUrl = `${protocol}//${host}:${port}/api/polls/${pollId}/ws?keepalive=true`;
      console.log('WebSocket连接地址:', wsUrl);
      
      // 创建WebSocket实例并返回
      const ws = new WebSocket(wsUrl);
      
      // 设置一个ping定时器，保持连接活跃 
      const pingInterval = setInterval(() => {
        if (ws && ws.readyState === WebSocket.OPEN) {
          try {
            // 发送ping消息保持连接活跃
            ws.send(JSON.stringify({ type: 'PING' }));
            console.log('发送WebSocket ping以保持连接');
          } catch (err) {
            console.error('发送ping消息失败:', err);
          }
        } else {
          // 如果连接已关闭，清除定时器
          clearInterval(pingInterval);
        }
      }, 30000); // 每30秒ping一次
      
      // 保存定时器引用，以便在关闭时清理
      ws.pingInterval = pingInterval;
      
      // 添加自定义关闭方法，确保清理定时器
      const originalClose = ws.close;
      ws.close = function() {
        if (this.pingInterval) {
          clearInterval(this.pingInterval);
          this.pingInterval = null;
        }
        return originalClose.apply(this, arguments);
      };
      
      return ws;
    } catch (error) {
      console.error('创建WebSocket连接时出错:', error);
      return null;
    }
  },
  
  // 获取投票编辑页面
  getPollForEdit: async (pollId) => {
    try {
      if (!pollId || isNaN(parseInt(pollId))) {
        throw new Error('无效的投票ID');
      }
      
      // 尝试使用专用的编辑接口
      try {
        const response = await apiClient.get(`/polls/${pollId}/edit`);
        return response.data;
      } catch (editError) {
        // 如果编辑接口不存在，使用普通的获取接口
        console.log('编辑接口不可用，使用普通获取接口:', editError);
        const response = await apiClient.get(`/polls/${pollId}`);
        return response.data;
      }
    } catch (error) {
      console.error('获取投票编辑数据错误:', error.response?.data || error.message);
      throw handleApiError(error);
    }
  },
  
  // 更新投票
  updatePoll: async (pollId, pollData) => {
    try {
      if (!pollId || isNaN(parseInt(pollId))) {
        throw new Error('无效的投票ID');
      }
      
      console.log(`更新投票 ID=${pollId}:`, pollData);
      const response = await apiClient.put(`/polls/${pollId}`, pollData);
      return response.data;
    } catch (error) {
      console.error('更新投票错误:', error.response?.data || error.message);
      throw handleApiError(error);
    }
  },
  
  // 删除投票
  deletePoll: async (pollId) => {
    try {
      if (!pollId || isNaN(parseInt(pollId))) {
        throw new Error('无效的投票ID');
      }
      
      console.log(`删除投票 ID=${pollId}`);
      const response = await apiClient.delete(`/polls/${pollId}`);
      return response.data;
    } catch (error) {
      console.error('删除投票错误:', error.response?.data || error.message);
      throw handleApiError(error);
    }
  },
};

// 创建模拟WebSocket对象，当后端不支持WebSocket时使用
const createMockWebSocket = (pollId) => {
  console.log('创建模拟WebSocket对象，定期轮询投票结果');
  
  // 模拟WebSocket对象
  const mockWs = {
    readyState: WebSocket.CONNECTING,
    onopen: null,
    onmessage: null,
    onerror: null,
    onclose: null,
    send: (data) => {
      console.log('模拟WebSocket发送消息:', data);
    },
    close: () => {
      console.log('关闭模拟WebSocket连接');
      mockWs.readyState = WebSocket.CLOSED;
      clearInterval(pollInterval);
      if (mockWs.onclose) {
        mockWs.onclose({ code: 1000, reason: '客户端手动关闭' });
      }
    }
  };
  
  // 设置为已连接状态
  setTimeout(() => {
    mockWs.readyState = WebSocket.OPEN;
    if (mockWs.onopen) {
      mockWs.onopen();
    }
  }, 500);
  
  // 定期轮询，模拟实时更新
  let lastData = null;
  const pollInterval = setInterval(async () => {
    if (mockWs.readyState !== WebSocket.OPEN) {
      return;
    }
    
    try {
      // 获取最新的投票数据
      const response = await apiClient.get(`/polls/${pollId}`);
      const data = response.data;
      
      // 如果数据变化了，触发onmessage事件
      if (lastData === null || JSON.stringify(data.options) !== JSON.stringify(lastData.options)) {
        lastData = data;
        
        // 转换为WebSocket消息格式 - 使用与后端一致的格式
        const message = {
          type: 'VOTE_UPDATE',  // 使用大写与后端保持一致
          data: {
            poll_id: pollId,
            options: data.options.map(opt => ({
              id: opt.id || opt.ID,
              text: opt.text || opt.Text,
              votes: opt.votes || 0,
              percentage: Math.round((opt.votes || 0) / (data.total_votes || 1) * 100)
            })),
            total_votes: data.total_votes || data.options.reduce((sum, opt) => sum + (opt.votes || 0), 0)
          }
        };
        
        if (mockWs.onmessage) {
          console.log('模拟WebSocket发送数据:', message);
          mockWs.onmessage({ data: JSON.stringify(message) });
        }
      }
    } catch (error) {
      console.error('模拟WebSocket轮询错误:', error);
    }
  }, 5000); // 每5秒轮询一次
  
  return mockWs;
};

// API错误处理
const handleApiError = (error) => {
  let errorMessage = '发生未知错误';

  if (error.response) {
    // 服务器返回了错误状态码
    const status = error.response.status;
    const responseData = error.response.data;
    console.log('API错误响应:', status, responseData);

    // 尝试提取错误消息
    if (responseData) {
      if (typeof responseData === 'string') {
        errorMessage = responseData;
      } else if (responseData.error) {
        errorMessage = responseData.error;
      } else if (responseData.message) {
        errorMessage = responseData.message;
      } else if (responseData.msg) {
        errorMessage = responseData.msg;
      } else {
        // 根据状态码返回通用错误
        switch (status) {
          case 400:
            errorMessage = '请求参数有误';
            break;
          case 401:
            errorMessage = '未授权，请重新登录';
            break;
          case 403:
            errorMessage = '无权访问该资源';
            break;
          case 404:
            errorMessage = '请求的资源不存在';
            break;
          case 422:
            errorMessage = '请求参数验证失败';
            break;
          case 500:
            errorMessage = '服务器内部错误';
            break;
          default:
            errorMessage = `请求失败 (${status})`;
        }
      }
    }
  } else if (error.request) {
    // 请求已发出，但没有收到响应
    errorMessage = '无法连接到服务器，请检查网络连接';
  } else {
    // 请求配置有误
    errorMessage = error.message;
  }

  return new Error(errorMessage);
};

export default pollService; 