import axios from 'axios';
import { 
  Poll, 
  PollVoteRequest, 
  PollVoteResponse, 
  CreatePollRequest 
} from '../types';

// API 基础 URL 配置
// 根据环境变量配置API基址，如果未设置则使用相对路径
// 使用8090端口（默认后端端口）
const API_BASE_URL = process.env.REACT_APP_API_BASE_URL || 'http://localhost:8090';

console.log('[API] 配置API基础URL:', API_BASE_URL);

// 创建axios实例
const api = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json'
  },
  // 启用凭证（cookies）发送，用于跨域请求
  withCredentials: false, // 改为false以避免CORS问题
  // 超时设置
  timeout: 10000
});

// 请求拦截器 - 用于调试
api.interceptors.request.use(config => {
  console.log(`[API] 发送请求: ${config.method?.toUpperCase()} ${config.baseURL}${config.url}`);
  return config;
}, error => {
  console.error('[API] 请求配置错误:', error);
  return Promise.reject(error);
});

// 响应拦截器 - 用于调试
api.interceptors.response.use(response => {
  console.log(`[API] 收到响应: ${response.status} ${response.config.url}`, response.data);
  return response;
}, error => {
  console.error('[API] 请求失败:', error.message);
  if (error.response) {
    console.error(`[API] 服务器返回: ${error.response.status}`, error.response.data);
  } else if (error.request) {
    console.error('[API] 没有收到响应:', error.request);
  }
  return Promise.reject(error);
});

// 获取所有投票
export const getPolls = async (): Promise<Poll[]> => {
  try {
    // 尝试使用不同的API路径格式
    try {
      const response = await api.get('/api/polls');
      console.log('[API] 获取所有投票成功:', response.data);
      return response.data;
    } catch (err) {
      console.log('[API] 尝试备用API路径...');
      const response = await api.get('/polls');
      console.log('[API] 获取所有投票成功(备用路径):', response.data);
      return response.data;
    }
  } catch (error) {
    console.error('[API] 获取所有投票失败:', error);
    throw error;
  }
};

// 获取单个投票详情
export const getPoll = async (id: number): Promise<Poll> => {
  try {
    // 尝试不同的API路径格式
    try {
      const response = await api.get(`/api/polls/${id}`);
      console.log(`[API] 获取投票(ID=${id})成功:`, response.data);
      return response.data;
    } catch (err) {
      console.log('[API] 尝试备用API路径...');
      const response = await api.get(`/polls/${id}`);
      console.log(`[API] 获取投票(ID=${id})成功(备用路径):`, response.data);
      return response.data;
    }
  } catch (error) {
    console.error(`[API] 获取投票(ID=${id})失败:`, error);
    throw error;
  }
};

// 创建新投票
export const createPoll = async (pollData: CreatePollRequest): Promise<Poll> => {
  try {
    const response = await api.post('/api/polls', pollData);
    console.log('[API] 创建投票成功:', response.data);
    return response.data;
  } catch (error) {
    console.error('[API] 创建投票失败:', error);
    throw error;
  }
};

// 提交投票
export const submitVote = async (
  pollId: number, 
  voteData: PollVoteRequest
): Promise<PollVoteResponse> => {
  try {
    const response = await api.post(`/api/polls/${pollId}/vote`, voteData);
    console.log(`[API] 提交投票(ID=${pollId})成功:`, response.data);
    return response.data;
  } catch (error) {
    console.error(`[API] 提交投票(ID=${pollId})失败:`, error);
    throw error;
  }
};

// 更新投票（如活动状态）
export const updatePoll = async (
  pollId: number, 
  data: Partial<Poll>
): Promise<Poll> => {
  try {
    const response = await api.put(`/api/polls/${pollId}`, data);
    console.log(`[API] 更新投票(ID=${pollId})成功:`, response.data);
    return response.data;
  } catch (error) {
    console.error(`[API] 更新投票(ID=${pollId})失败:`, error);
    throw error;
  }
};

// 删除投票
export const deletePoll = async (pollId: number): Promise<{ message: string }> => {
  try {
    const response = await api.delete(`/api/polls/${pollId}`);
    console.log(`[API] 删除投票(ID=${pollId})成功:`, response.data);
    return response.data;
  } catch (error) {
    console.error(`[API] 删除投票(ID=${pollId})失败:`, error);
    throw error;
  }
}; 