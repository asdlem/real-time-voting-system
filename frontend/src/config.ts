// 定义应用程序的基本配置

// API基础URL，优先使用环境变量，如果没有则使用localhost
export const API_BASE_URL = process.env.REACT_APP_API_BASE_URL || 'http://localhost:8090';

// SSE配置
export const USE_MOCK_SSE = process.env.REACT_APP_MOCK_SSE === 'true';

// 应用程序默认配置
export const APP_CONFIG = {
  defaultPollDuration: 7 * 24 * 60 * 60 * 1000, // 默认投票持续7天（毫秒）
  maxPollOptions: 10, // 最大投票选项数量
  minPollOptions: 2, // 最小投票选项数量
  simulationDefaultVotes: 10, // 默认模拟投票数量
  simulationDefaultInterval: 100, // 默认模拟投票间隔（毫秒）
}; 