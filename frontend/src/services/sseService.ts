import { PollOptionResult } from '../types';
import { API_BASE_URL } from '../config';

interface SSECallbacks {
  onMessage: (data: any) => void;
  onOpen: () => void;
  onError: (error: string) => void;
}

class SSEService {
  private eventSource: EventSource | null = null;
  private retryCount: number = 0;
  private maxRetries: number = 3;
  private retryInterval: number = 3000; // 3秒后重试
  private pollId: number | null = null;
  private callbacks: SSECallbacks | null = null;

  /**
   * 连接到投票结果的SSE流
   * @param pollId 投票ID
   * @param callbacks 事件处理选项
   */
  connect(pollId: number, callbacks: SSECallbacks): void {
    // 如果已经连接，先断开
    this.disconnect();
    
    this.pollId = pollId;
    this.callbacks = callbacks;
    this.retryCount = 0;
    
    try {
      // 尝试连接投票实时更新的端点
      console.log(`[SSEService] 正在连接SSE: /api/polls/${pollId}/live-results`);
      this.createEventSource(`${API_BASE_URL}/api/polls/${pollId}/live-results`);
    } catch (error) {
      console.error('[SSEService] 连接SSE时发生错误:', error);
      if (this.callbacks) {
        this.callbacks.onError(`连接SSE时发生错误: ${error}`);
      }
    }
  }

  private createEventSource(url: string) {
    try {
      console.log(`[SSEService] 创建EventSource: ${url}`);
      this.eventSource = new EventSource(url);
      
      this.eventSource.onopen = (event) => {
        console.log('[SSEService] SSE连接已开启');
        this.retryCount = 0; // 重置重试计数
        if (this.callbacks) {
          this.callbacks.onOpen();
        }
      };
      
      this.eventSource.onmessage = (event) => {
        try {
          console.log(`[SSEService] 收到SSE消息: ${event.data.substring(0, 100)}...`);
          const data = JSON.parse(event.data);
          if (this.callbacks) {
            this.callbacks.onMessage(data);
          }
        } catch (error) {
          console.error('[SSEService] 处理SSE消息时出错:', error);
          console.error('[SSEService] 原始消息:', event.data);
        }
      };
      
      this.eventSource.onerror = (event) => {
        console.error('[SSEService] SSE连接错误:', event);
        
        if (this.eventSource) {
          if (this.eventSource.readyState === EventSource.CLOSED) {
            console.log('[SSEService] SSE连接已关闭');
            // 尝试重新连接
            this.retryConnection();
          } else if (this.eventSource.readyState === EventSource.CONNECTING) {
            console.log('[SSEService] SSE正在重新连接...');
          }
        }
        
        if (this.callbacks) {
          this.callbacks.onError('SSE连接出错');
        }
      };
    } catch (error) {
      console.error('[SSEService] 创建EventSource时出错:', error);
      if (this.callbacks) {
        this.callbacks.onError(`创建EventSource时出错: ${error}`);
      }
    }
  }

  private retryConnection() {
    if (this.retryCount < this.maxRetries && this.pollId !== null && this.callbacks !== null) {
      console.log(`[SSEService] 尝试重新连接 (${this.retryCount + 1}/${this.maxRetries})...`);
      
      setTimeout(() => {
        this.retryCount++;
        
        // 优先尝试live-results端点
        try {
          console.log(`[SSEService] 重试连接: /api/polls/${this.pollId}/live-results`);
          this.createEventSource(`${API_BASE_URL}/api/polls/${this.pollId}/live-results`);
        } catch (error) {
          console.error('[SSEService] 重试连接live-results端点时出错:', error);
          
          // 如果live-results失败，尝试live端点
          try {
            console.log(`[SSEService] 尝试备用连接: /api/polls/${this.pollId}/live`);
            this.createEventSource(`${API_BASE_URL}/api/polls/${this.pollId}/live`);
          } catch (fallbackError) {
            console.error('[SSEService] 备用连接也失败:', fallbackError);
            if (this.callbacks) {
              this.callbacks.onError('所有SSE连接尝试均失败');
            }
          }
        }
      }, this.retryInterval);
    } else if (this.callbacks) {
      console.error('[SSEService] 达到最大重试次数，放弃连接');
      this.callbacks.onError('达到最大重试次数，SSE连接失败');
    }
  }

  /**
   * 断开SSE连接
   */
  disconnect(): void {
    if (this.eventSource) {
      console.log('[SSEService] 断开SSE连接');
      this.eventSource.close();
      this.eventSource = null;
    }
    this.pollId = null;
    this.callbacks = null;
  }

  /**
   * 检查是否正在连接特定投票的结果
   */
  isActive(): boolean {
    return this.eventSource !== null;
  }
}

// 导出单例实例
export const sseService = new SSEService(); 