import React, { useState, useEffect, useRef, useCallback, useMemo } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Card, Typography, Radio, Checkbox, Button, Spin, notification, Progress, Result, Badge, message, Tabs, Modal } from 'antd';
import { BarChartOutlined, ArrowLeftOutlined, EditOutlined, DeleteOutlined, PieChartOutlined, LineChartOutlined, ReloadOutlined } from '@ant-design/icons';
import { PieChart, Pie, BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, Cell } from 'recharts';
import pollService from '../api/pollService';
import './PollDetail.css';
import dayjs from 'dayjs';

const { Title, Text, Paragraph } = Typography;
const { TabPane } = Tabs;

// 图表的颜色
const COLORS = ['#1890ff', '#13c2c2', '#52c41a', '#faad14', '#f5222d', '#722ed1', '#eb2f96', '#fa8c16', '#a0d911', '#eb2f96'];

const PollDetail = () => {
  const { id } = useParams();
  const navigate = useNavigate();
  const [poll, setPoll] = useState(null);
  const [selectedOptions, setSelectedOptions] = useState([]);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [voted, setVoted] = useState(false);
  const [error, setError] = useState(null);
  const webSocketRef = useRef(null);
  const [votingSuccess, setVotingSuccess] = useState(false);
  const [selectedOption, setSelectedOption] = useState(null);
  const [wsConnected, setWsConnected] = useState(false);
  const [chartType, setChartType] = useState('pie');
  // 添加最后更新时间戳，用于消息去重
  const lastUpdateTimestampRef = useRef(0);
  // 添加手动刷新锁，防止短时间重复刷新
  const refreshLockRef = useRef(false);
  // 添加定时刷新计时器引用
  const periodicRefreshTimerRef = useRef(null);
  // 添加上次消息计数，用于检测数据不一致
  const lastMessageCountsRef = useRef({});

  // 计算总票数和百分比
  const calculateTotalVotes = useCallback((options) => {
    if (!options || !Array.isArray(options)) return 0;
    return options.reduce((sum, option) => sum + (option.votes || 0), 0);
  }, []);

  // 计算选项百分比
  const calculatePercentage = useCallback((votes, totalVotes) => {
    if (!totalVotes) return 0;
    return (votes / totalVotes) * 100;
  }, []);

  // 实时更新的总票数（用于显示）
  const totalVotes = useMemo(() => {
    return poll?.options ? calculateTotalVotes(poll.options) : 0;
  }, [poll?.options, calculateTotalVotes]);

  // 更新后的投票选项（包含百分比）
  const optionsWithPercentage = useMemo(() => {
    if (!poll?.options) return [];
    
    return poll.options.map(option => ({
      ...option,
      percentage: calculatePercentage(option.votes || 0, totalVotes)
    }));
  }, [poll?.options, totalVotes, calculatePercentage]);

  // 提交投票后需处理的操作
  const handleVoteSuccess = useCallback((updatedPollData) => {
    // 直接更新poll数据，所有显示会基于此数据计算
    setPoll(updatedPollData);
    
    // 更新UI状态
    setVotingSuccess(true);
    setVoted(true);
    
    // 显示成功提示
    notification.success({
      message: '投票成功',
      description: '您的投票已成功提交，结果已更新',
      placement: 'bottomRight',
      duration: 3,
    });
  }, []);

  // 手动刷新投票数据的功能
  const refreshPollData = useCallback(async (force = false) => {
    // 如果锁定中且非强制刷新，则跳过
    if (refreshLockRef.current && !force) {
      console.log('刷新操作被锁定，跳过此次刷新');
      return;
    }
    
    // 设置锁定，防止短时间内重复刷新
    refreshLockRef.current = true;
    setTimeout(() => {
      refreshLockRef.current = false;
    }, 2000); // 2秒刷新锁
    
    try {
      console.log('手动刷新投票数据...');
      const pollId = id;
      const updatedPollData = await pollService.getPollById(Number(pollId));
      
      console.log('手动刷新获取的数据:', updatedPollData);
      setPoll(updatedPollData);
      
      // 更新最后更新时间戳
      lastUpdateTimestampRef.current = Date.now();
      
      // 如果连接已断开，尝试给用户提示可手动刷新
      if (!wsConnected) {
        notification.info({
          message: '数据已刷新',
          description: '已获取最新投票数据。实时连接未建立，可手动刷新获取最新数据。',
          placement: 'bottomRight',
          duration: 3,
          key: 'manual-refresh-success'
        });
      }
      
      return updatedPollData;
    } catch (err) {
      console.error('手动刷新投票数据失败:', err);
      // 仅在强制刷新时显示错误通知
      if (force) {
        notification.error({
          message: '刷新失败',
          description: '获取最新数据失败，请稍后再试',
          placement: 'bottomRight',
          duration: 3,
        });
      }
      return null;
    }
  }, [id, wsConnected]);

  // 设置定期轮询，作为WebSocket的备份机制
  useEffect(() => {
    if (!poll) return;
    
    // 如果WebSocket连接断开，则启动定期轮询
    if (!wsConnected) {
      console.log('WebSocket连接断开，启动备份轮询机制');
      
      // 清除之前的计时器
      if (periodicRefreshTimerRef.current) {
        clearInterval(periodicRefreshTimerRef.current);
      }
      
      // 设置30秒轮询
      periodicRefreshTimerRef.current = setInterval(() => {
        refreshPollData(false); // 静默刷新，无需提示
      }, 30000);
    } else {
      // WebSocket连接恢复，清除轮询
      if (periodicRefreshTimerRef.current) {
        console.log('WebSocket连接恢复，停止备份轮询');
        clearInterval(periodicRefreshTimerRef.current);
        periodicRefreshTimerRef.current = null;
      }
    }
    
    // 清理函数
    return () => {
      if (periodicRefreshTimerRef.current) {
        clearInterval(periodicRefreshTimerRef.current);
        periodicRefreshTimerRef.current = null;
      }
    };
  }, [wsConnected, poll, refreshPollData]);

  // 改进WebSocket消息处理
  const handleWebSocketMessage = useCallback((data) => {
    console.log('处理WebSocket消息:', data);
    
    // 检查消息格式
    if (!data) return;
    
    try {
      // 检查消息时间戳，忽略旧消息
      const messageTimestamp = data.data?.timestamp || (data.timestamp ? Number(data.timestamp) : Date.now());
      if (messageTimestamp <= lastUpdateTimestampRef.current) {
        console.log('收到过时的WebSocket消息，忽略处理', {
          messageTimestamp,
          lastUpdateTimestamp: lastUpdateTimestampRef.current,
          difference: messageTimestamp - lastUpdateTimestampRef.current
        });
        return;
      }
      
      // 更新最后消息时间戳
      lastUpdateTimestampRef.current = messageTimestamp;
      
      // 只处理投票更新类型的消息
      if (data.type === 'VOTE_UPDATE' || data.type === 'vote_update' || data.current_results) {
        let updatedOptions = [];
        
        // 检查包含options属性的多种不同格式
        if (data.data && data.data.options) {
          // 确认data.data.options中包含选项数据
          if (Array.isArray(data.data.options)) {
            // 格式1: data.data.options是数组
            updatedOptions = data.data.options.map(option => ({
              id: option.id || option.ID,
              votes: parseInt(option.votes || 0)
            }));
          } else if (data.data.options.results && Array.isArray(data.data.options.results)) {
            // 格式2: data.data.options.results是数组
            updatedOptions = data.data.options.results.map(option => ({
              id: option.id || option.ID,
              votes: parseInt(option.votes || 0)
            }));
          } else if (typeof data.data.options === 'object' && data.data.options.poll_id) {
            // 格式3: data.data.options是包含poll_id和results的对象
            const results = data.data.options.results;
            if (Array.isArray(results)) {
              updatedOptions = results.map(option => ({
                id: option.id || option.ID,
                votes: parseInt(option.votes || 0)
              }));
            }
          }
        } 
        // 直接使用options数组
        else if (data.options && Array.isArray(data.options)) {
          updatedOptions = data.options.map(option => ({
            id: option.id || option.ID,
            votes: parseInt(option.votes || 0)
          }));
        }
        // 使用results数组
        else if (data.results && Array.isArray(data.results)) {
          updatedOptions = data.results.map(option => ({
            id: option.id || option.ID,
            votes: parseInt(option.votes || 0)
          }));
        }
        // 直接使用data.data.results数组
        else if (data.data && data.data.results && Array.isArray(data.data.results)) {
          updatedOptions = data.data.results.map(option => ({
            id: option.id || option.ID,
            votes: parseInt(option.votes || 0)
          }));
        }
        // 处理current_results格式
        else if (data.current_results && Array.isArray(data.current_results)) {
          updatedOptions = data.current_results.map(option => ({
            id: option.id || option.ID,
            votes: parseInt(option.votes || 0)
          }));
        }
        
        // 如果成功提取了选项数据，更新UI
        if (updatedOptions.length > 0 && poll && poll.options) {
          // 检查数据一致性，比较各选项的总票数变化
          let totalNewVotes = 0;
          let totalCurrentVotes = 0;
          
          // 记录每个选项的新票数
          const newCountsByOption = {};
          updatedOptions.forEach(opt => {
            const votes = parseInt(opt.votes || 0);
            newCountsByOption[opt.id] = votes;
            totalNewVotes += votes;
          });
          
          // 记录当前显示的票数
          const currentCounts = {};
          poll.options.forEach(opt => {
            const id = opt.id || opt.ID;
            const votes = parseInt(opt.votes || 0);
            currentCounts[id] = votes;
            totalCurrentVotes += votes;
          });
          
          console.log('票数变化分析:', {
            currentTotal: totalCurrentVotes,
            newTotal: totalNewVotes,
            currentByOption: currentCounts,
            newByOption: newCountsByOption
          });
          
          // 检查是否有任何选项的票数减少（这可能是重置操作）
          let hasDecreased = false;
          let isSuspiciousDiff = false;
          
          for (const id in currentCounts) {
            const currentVotes = currentCounts[id];
            const newVotes = newCountsByOption[id] || 0;
            
            if (newVotes < currentVotes) {
              hasDecreased = true;
              console.log(`选项 ${id} 票数减少: ${currentVotes} -> ${newVotes}`);
            }
            
            // 检查是否有票数差异过大（可能是数据不一致）
            // 对于高并发场景（比如测试），差异较大是正常的，所以仅作为日志记录，不影响更新
            if (Math.abs(newVotes - currentVotes) > 50) {
              isSuspiciousDiff = true;
              console.log(`警告: 选项 ${id} 票数变化过大: ${currentVotes} -> ${newVotes}`);
            }
          }
          
          // 更新当前poll的options数据
          const newOptions = [...poll.options];
          
          updatedOptions.forEach(updatedOption => {
            const index = newOptions.findIndex(opt => 
              (opt.id === updatedOption.id || opt.ID === updatedOption.id)
            );
            
            if (index !== -1) {
              // 只更新票数，保留其他属性
              newOptions[index] = {
                ...newOptions[index],
                votes: updatedOption.votes
              };
            }
          });
          
          // 设置更新后的poll数据
          setPoll(prev => {
            if (!prev) return prev;
            
            const updatedPoll = {
              ...prev,
              options: newOptions
            };
            
            // 打印更新后的票数详情，用于调试
            console.log('更新后的投票数据详情:');
            let totalVotesLog = 0;
            updatedPoll.options.forEach(opt => {
              const votes = opt.votes || 0;
              totalVotesLog += votes;
              console.log(`  选项 ${opt.id}: ${votes} 票`);
            });
            console.log(`  总票数: ${totalVotesLog}`);
            
            // 储存这次消息的票数，用于下次比较
            lastMessageCountsRef.current = {
              total: totalVotesLog,
              byOption: updatedPoll.options.reduce((acc, opt) => {
                acc[opt.id || opt.ID] = opt.votes || 0;
                return acc;
              }, {})
            };
            
            return updatedPoll;
          });
          
          console.log('已通过WebSocket更新投票数据');
          
          // 如果票数减少，可能是投票已重置
          if (hasDecreased && totalNewVotes === 0) {
            notification.info({
              message: '投票已重置',
              description: '投票已被重置，所有选项票数已清零',
              placement: 'bottomRight',
              duration: 3
            });
          }
          
          // 如果发现数据可能不一致，提示用户可手动刷新
          if (isSuspiciousDiff && totalNewVotes > 0 && totalCurrentVotes > 0) {
            // 使用防抖函数，避免频繁显示通知
            if (!refreshLockRef.current) {
              refreshLockRef.current = true;
              setTimeout(() => {
                refreshLockRef.current = false;
              }, 10000); // 10秒内不再显示刷新提示
              
              notification.info({
                message: '数据可能不完整',
                description: '检测到投票数据可能不完整，点击"刷新"按钮获取最新数据',
                placement: 'bottomRight',
                duration: 5,
                key: 'suspicious-data-diff'
              });
            }
          }
        } else {
          console.warn('无法从WebSocket消息中提取选项数据:', data);
        }
      } 
      // 处理系统消息
      else if (data.type === 'CONNECT_SUCCESS' || data.type === 'PONG') {
        // 心跳或连接成功消息，无需特殊处理
        console.log('收到系统消息:', data.type);
      }
      // 处理普通消息
      else if (data.message) {
        // 显示服务器发送的消息通知
        notification.info({
          message: '系统消息',
          description: data.message,
          placement: 'bottomRight',
          key: `ws-message-${Date.now()}`,
          duration: 3
        });
      } 
      // 未识别的消息格式
      else {
        console.warn('收到未知格式的WebSocket消息:', data);
      }
    } catch (err) {
      console.error('处理WebSocket消息时出错:', err);
    }
  }, [poll]);

  // 格式化日期函数
  const formatDate = (dateString) => {
    if (!dateString) return '未设置';
    
    try {
      // 尝试多种格式解析日期
      let date;
      if (typeof dateString === 'string' && dateString.includes('+')) {
        // 处理带时区的日期
        date = new Date(dateString);
      } else {
        // 尝试使用 dayjs 解析
        date = dayjs(dateString).toDate();
      }
      
      // 检查日期是否有效
      if (isNaN(date.getTime())) {
        console.error('无效的日期:', dateString);
        return '日期无效';
      }
      
      // 格式化日期
      return date.toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
      });
    } catch (err) {
      console.error('日期格式化错误:', err, dateString);
      return '日期格式错误';
    }
  };

  // 获取投票详情
  useEffect(() => {
    const fetchPoll = async () => {
      setLoading(true);
      setError(null);
      
      try {
        const pollId = id;
        
        // 获取投票数据
        const data = await pollService.getPollById(pollId);
        console.log('获取到原始投票数据:', data);
        
        // 规范化数据 - 处理字段名不一致问题
        const normalizedPoll = {
          ...data,
          // 统一使用小写字段名
          id: data.id || data.ID,
          question: data.question || data.Question,
          description: data.description || data.Description,
          poll_type: data.poll_type !== undefined ? data.poll_type : data.PollType,
          is_active: data.is_active !== undefined ? data.is_active : data.IsActive,
          options: Array.isArray(data.options) ? data.options : 
                  Array.isArray(data.Options) ? data.Options : []
        };
        
        console.log('规范化后的投票数据:', normalizedPoll);
        
        // 设置投票数据
        setPoll(normalizedPoll);
      } catch (err) {
        console.error('获取投票详情失败:', err);
        setError(err.message || '获取投票详情失败');
      } finally {
        setLoading(false);
      }
    };
    
    if (id) {
      fetchPoll();
    }
    
    return () => {
      // 组件卸载时关闭WebSocket连接
      if (webSocketRef.current) {
        webSocketRef.current.close();
        webSocketRef.current = null;
      }
      setVotingSuccess(false);
      setVoted(false);
    };
  }, [id]);

  // 创建WebSocket连接
  useEffect(() => {
    if (!poll) {
      console.log('无投票数据，尚未创建WebSocket连接');
      return;
    }

    // 获取正确的ID
    const pollId = poll.id || poll.ID;
    if (!pollId) {
      console.error('投票数据中缺少ID字段:', poll);
      return;
    }

    // 清理函数 - 关闭现有的WebSocket连接
    const cleanupWebSocket = () => {
      if (webSocketRef.current) {
        console.log('关闭现有WebSocket连接');
        webSocketRef.current.close();
        webSocketRef.current = null;
        setWsConnected(false);
      }
    };

    // 防止重复创建WebSocket连接
    if (webSocketRef.current && webSocketRef.current.readyState !== WebSocket.CLOSED) {
      console.log('WebSocket连接已存在且未关闭，跳过创建');
      return;
    }

    // 重连计数器和最大重试次数
    let reconnectAttempts = 0;
    const MAX_RECONNECT_ATTEMPTS = 5;
    let reconnectTimeout = null;

    // 创建WebSocket连接的函数
    const createWebSocketConnection = () => {
      try {
        console.log(`尝试为投票ID ${pollId} 创建WebSocket连接 (尝试#${reconnectAttempts+1})`);
        const ws = pollService.createWebSocketConnection(pollId);
        
        if (ws) {
          webSocketRef.current = ws;
          
          ws.onopen = () => {
            if (ws.readyState === WebSocket.OPEN) {
              console.log('WebSocket连接成功');
              setWsConnected(true);
              reconnectAttempts = 0; // 连接成功，重置重连次数
              
              // 仅当之前没有连接时显示通知
              if (!wsConnected) {
                notification.success({
                  message: '实时连接已建立',
                  description: '您将收到投票的实时更新',
                  placement: 'bottomRight',
                  duration: 3,
                  key: 'ws-connect-success' // 使用固定key防止重复
                });
              }
            }
          };
          
          ws.onmessage = (event) => {
            try {
              console.log('收到WebSocket消息:', event.data);
              const data = JSON.parse(event.data);
              
              // 处理接收到的消息
              handleWebSocketMessage(data);
            } catch (err) {
              console.error('解析WebSocket消息失败:', err);
            }
          };
          
          ws.onclose = (event) => {
            console.log('WebSocket连接已关闭, 代码:', event.code, '原因:', event.reason);
            setWsConnected(false);
            
            // 如果连接异常关闭，尝试重连
            if (event.code !== 1000 && event.code !== 1001) { // 非正常关闭
              if (reconnectAttempts < MAX_RECONNECT_ATTEMPTS) {
                reconnectAttempts++;
                const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000); // 指数退避，最长30秒
                console.log(`将在 ${delay}ms 后尝试重连 (尝试 ${reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS})`);
                
                reconnectTimeout = setTimeout(() => {
                  if (webSocketRef.current === ws) {
                    webSocketRef.current = null;
                    createWebSocketConnection(); // 重连
                  }
                }, delay);
              } else {
                console.error(`达到最大重试次数 (${MAX_RECONNECT_ATTEMPTS})，停止重连`);
                notification.error({
                  message: '实时连接失败',
                  description: '无法建立实时连接，请刷新页面重试',
                  placement: 'bottomRight',
                  key: 'ws-reconnect-failed'
                });
              }
            } else {
              // 正常关闭，清理连接
              if (webSocketRef.current === ws) {
                webSocketRef.current = null;
              }
            }
          };
          
          ws.onerror = (error) => {
            console.error('WebSocket连接错误:', error);
            
            // 只有当前状态为已连接时才显示错误通知
            if (wsConnected) {
              setWsConnected(false);
              notification.error({
                message: '实时连接失败',
                description: '连接发生错误，将尝试重新连接',
                placement: 'bottomRight',
                key: 'ws-connect-error'
              });
            }
            // 错误处理由onclose处理重连
          };
        }
      } catch (err) {
        console.error('创建WebSocket连接失败:', err);
        // 连接创建失败，也尝试重连
        if (reconnectAttempts < MAX_RECONNECT_ATTEMPTS) {
          reconnectAttempts++;
          const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000);
          console.log(`连接失败，将在 ${delay}ms 后尝试重连 (尝试 ${reconnectAttempts}/${MAX_RECONNECT_ATTEMPTS})`);
          
          reconnectTimeout = setTimeout(() => {
            createWebSocketConnection();
          }, delay);
        }
      }
    };

    // 立即创建连接，不再延迟
    createWebSocketConnection();

    // 当组件卸载或poll改变时，清理资源
    return () => {
      if (reconnectTimeout) {
        clearTimeout(reconnectTimeout);
      }
      cleanupWebSocket();
    };
  // 只在poll更改时重新创建连接，不依赖wsConnected状态
  }, [poll, handleWebSocketMessage]);

  // 处理选项选择
  const handleOptionSelect = (optionId) => {
    // 添加调试输出
    console.log('投票类型检查:', { 
      poll_type: poll?.poll_type, 
      PollType: poll?.PollType,
      poll_type_value: typeof poll?.poll_type,
      is_single: poll?.poll_type === 0,
      raw_poll: poll
    });

    // 统一检查逻辑，优先使用poll_type，如果不存在则使用PollType
    const pollTypeValue = poll?.poll_type !== undefined ? poll.poll_type : poll?.PollType;
    
    if (pollTypeValue === 0) {
      // 单选模式
      setSelectedOptions([optionId]);
    } else {
      // 多选模式
      if (selectedOptions.includes(optionId)) {
        setSelectedOptions(selectedOptions.filter(id => id !== optionId));
      } else {
        setSelectedOptions([...selectedOptions, optionId]);
      }
    }
  };

  // 提交投票
  const handleSubmit = async () => {
    if (!selectedOptions.length) {
      notification.warning({
        message: '请选择选项',
        description: '提交前请至少选择一个选项',
      });
      return;
    }

    try {
      setSubmitting(true);
      console.log(`提交投票，ID: ${id}, 选项: ${selectedOptions}`);
      
      // 准备提交数据，使用正确的格式
      const voteData = { option_ids: selectedOptions };
      
      const response = await pollService.submitVote(Number(id), voteData);
      console.log('投票提交成功，响应:', response);
      
      // 提交成功后重新获取最新投票数据
      try {
        const updatedPollData = await pollService.getPollById(Number(id));
        console.log('获取更新后的投票数据:', updatedPollData);
        
        // 使用通用处理函数更新UI
        handleVoteSuccess(updatedPollData);
      } catch (refreshErr) {
        console.error('刷新投票数据失败:', refreshErr);
        // 即使刷新失败，也更新UI状态
        setVoted(true);
        setVotingSuccess(true);
        
        notification.success({
          message: '投票成功',
          description: '您的投票已提交，但获取最新结果失败',
          placement: 'bottomRight',
        });
      }
    } catch (err) {
      console.error('投票提交失败:', err);
      notification.error({
        message: '投票失败',
        description: err.message || '提交投票时发生错误',
      });
    } finally {
      setSubmitting(false);
    }
  };

  // 返回列表
  const handleBack = () => {
    navigate('/');
  };

  // 编辑投票
  const handleEdit = () => {
    navigate(`/poll/${id}/edit`);
  };

  // 删除投票
  const handleDelete = () => {
    // 弹出确认对话框
    Modal.confirm({
      title: '确认删除',
      content: '您确定要删除这个投票吗？此操作不可撤销。',
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          setLoading(true);
          await pollService.deletePoll(Number(id));
          
          notification.success({
            message: '删除成功',
            description: '投票已成功删除',
          });
          
          // 删除成功后返回列表页
          navigate('/');
        } catch (err) {
          console.error('删除投票失败:', err);
          notification.error({
            message: '删除失败',
            description: err.message || '无法删除投票，请稍后再试',
          });
        } finally {
          setLoading(false);
        }
      },
    });
  };

  // 处理单个选项选择
  const handleSingleOptionSelect = (optionId) => {
    setSelectedOption(optionId);
  };

  // 提交单个选项投票
  const handleSubmitVote = async () => {
    if (!selectedOption) {
      message.error('请选择一个选项');
      return;
    }

    try {
      setSubmitting(true);
      // 提交投票，使用正确的格式
      const response = await pollService.submitVote(id, { option_ids: [selectedOption] });
      console.log('单选投票提交成功，响应:', response);
      
      // 提交成功后重新获取最新投票数据
      try {
        const updatedPollData = await pollService.getPollById(Number(id));
        console.log('获取更新后的投票数据:', updatedPollData);
        
        // 使用通用处理函数更新UI
        handleVoteSuccess(updatedPollData);
      } catch (refreshErr) {
        console.error('刷新投票数据失败:', refreshErr);
        // 即使刷新失败，也更新UI状态
        setVoted(true);
        setVotingSuccess(true);
        
        notification.success({
          message: '投票成功',
          description: '您的投票已提交，但获取最新结果失败',
          placement: 'bottomRight',
        });
      }
    } catch (err) {
      console.error('单选投票提交失败:', err);
      message.error(err.message || '投票失败，请重试');
    } finally {
      setSubmitting(false);
    }
  };

  const isPollActive = () => {
    if (!poll) return false;
    const now = dayjs();
    const startTime = dayjs(poll.start_time || poll.StartTime);
    const endTime = dayjs(poll.end_time || poll.EndTime);
    return now.isAfter(startTime) && now.isBefore(endTime);
  };

  const isPollClosed = () => {
    if (!poll) return false;
    const now = dayjs();
    const endTime = dayjs(poll.end_time || poll.EndTime);
    return now.isAfter(endTime);
  };

  const isPollNotStarted = () => {
    if (!poll) return false;
    const now = dayjs();
    const startTime = dayjs(poll.start_time || poll.StartTime);
    return now.isBefore(startTime);
  };

  // 准备图表数据
  const prepareChartData = useCallback(() => {
    if (!poll || !poll.options) return [];
    
    // 确保总票数计算正确
    const totalVotes = calculateTotalVotes(poll.options);
    
    return poll.options.map((option, index) => {
      const text = option.text || option.Text || option.content || option.Content || '未命名选项';
      const votes = parseInt(option.votes) || 0;
      
      return {
        name: text,
        value: votes,
        // 计算百分比 - 注意这里不要乘以100，因为图表组件内部会处理
        percent: calculatePercentage(votes, totalVotes) / 100,
        fill: COLORS[index % COLORS.length]
      };
    });
  }, [poll, calculatePercentage, calculateTotalVotes]);

  // 自定义ToolTip组件
  const CustomTooltip = ({ active, payload, label }) => {
    if (active && payload && payload.length) {
      return (
        <div className="custom-tooltip">
          <p className="tooltip-name">{payload[0].name}</p>
          <p className="tooltip-value">{`票数: ${payload[0].value}`}</p>
          <p className="tooltip-percent">
            {`比例: ${calculatePercentage(payload[0].value, totalVotes)}%`}
          </p>
        </div>
      );
    }
    return null;
  };

  // 在投票详情页上添加刷新按钮的渲染
  const renderRefreshButton = () => {
    return (
      <Button 
        icon={<ReloadOutlined />} 
        onClick={() => refreshPollData(true)} 
        loading={loading}
        style={{ marginLeft: 8 }}
      >
        刷新
      </Button>
    );
  };

  if (loading) {
    return (
      <div className="poll-detail-container loading">
        <Spin size="large" />
        <p>加载中...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="poll-detail-container error">
        <Result
          status="error"
          title="加载失败"
          subTitle={error}
          extra={
            <Button type="primary" onClick={handleBack}>
              返回列表
            </Button>
          }
        />
      </div>
    );
  }

  if (!poll) {
    return (
      <div className="poll-detail-container error">
        <Result
          status="404"
          title="未找到"
          subTitle="抱歉，该投票不存在或已被删除"
          extra={
            <Button type="primary" onClick={handleBack}>
              返回列表
            </Button>
          }
        />
      </div>
    );
  }

  return (
    <div className="poll-detail-container">
      <div className="navigation-header">
        <Button
          type="link"
          icon={<ArrowLeftOutlined />}
          onClick={() => navigate('/')}
          className="back-button"
        >
          返回列表
        </Button>
        {poll && (
          <div className="buttons-group">
            <Button
              type="link"
              icon={<ReloadOutlined />}
              onClick={() => refreshPollData(true)}
              className="refresh-button"
              loading={loading}
            >
              刷新
            </Button>
            <Button
              type="link"
              icon={<EditOutlined />}
              onClick={handleEdit}
              className="edit-button"
            >
              编辑投票
            </Button>
            <Button
              type="link"
              icon={<DeleteOutlined />}
              onClick={handleDelete}
              className="delete-button"
              danger
            >
              删除投票
            </Button>
          </div>
        )}
      </div>

      <Card className="poll-card" title={poll.question || poll.Question || poll.title || poll.Title}>
        <p className="poll-description">{poll.description || poll.Description}</p>
        
        <div className="poll-info">
          <div>
            {/* 统一获取投票类型 */}
            {(() => {
              const pollType = poll.poll_type !== undefined ? poll.poll_type : poll.PollType;
              return (
                <span className="poll-type">
                  {pollType === 1 ? '多选' : '单选'} 
                  {pollType === 1 && <span className="poll-type-note">（可选择多个选项）</span>}
                </span>
              );
            })()}
            {wsConnected && (
              <Badge status="processing" text="实时连接" style={{ marginLeft: '8px' }} />
            )}
          </div>
        </div>
        
        {votingSuccess ? (
          <div className="vote-success">
            <h3>感谢您的投票！</h3>
            <p>您可以在下方查看当前结果</p>
          </div>
        ) : isPollClosed() ? (
          <div className="poll-closed">
            <h3>投票已结束</h3>
            <p>查看以下投票结果</p>
          </div>
        ) : isPollNotStarted() ? (
          <div className="poll-closed">
            <h3>投票尚未开始</h3>
          </div>
        ) : (
          <div className="options-container">
            <h3>请选择{(poll.poll_type === 1 || poll.PollType === 1) ? '多个' : '一个'}选项：</h3>
            {poll.options && poll.options.map((option) => {
              const optionId = option.id || option.ID;
              const optionText = option.text || option.Text || option.content || option.Content;
              
              if (poll.poll_type === 1 || poll.PollType === 1) {
                // 多选
                return (
                  <div key={optionId} className="option-item">
                    <Checkbox 
                      checked={selectedOptions.includes(optionId)}
                      onChange={() => handleOptionSelect(optionId)}
                    >
                      {optionText}
                    </Checkbox>
                  </div>
                );
              } else {
                // 单选
                return (
                  <div
                    key={optionId}
                    className="option-item"
                    style={{
                      backgroundColor: selectedOption === optionId ? '#e6f7ff' : '#f9f9f9',
                      border: selectedOption === optionId ? '1px solid #1890ff' : 'none',
                      cursor: 'pointer'
                    }}
                    onClick={() => handleSingleOptionSelect(optionId)}
                  >
                    {optionText}
                  </div>
                );
              }
            })}
            
            <Button
              type="primary"
              className="vote-button"
              onClick={(poll.poll_type === 1 || poll.PollType === 1) ? handleSubmit : handleSubmitVote}
              loading={submitting}
              disabled={(poll.poll_type === 1 || poll.PollType === 1) ? 
                selectedOptions.length === 0 : !selectedOption}
            >
              提交投票
            </Button>
          </div>
        )}
        
        <div className="results-container">
          <div className="results-title">
            <h3>
              <PieChartOutlined /> 实时结果
              <span className="total-votes">总票数: {totalVotes}</span>
            </h3>
          </div>
          
          <Tabs defaultActiveKey="progress" className="result-tabs">
            <TabPane 
              tab={<span><LineChartOutlined /> 进度条</span>} 
              key="progress"
            >
              {poll.options && poll.options.map((option) => {
                const optionId = option.id || option.ID;
                const optionText = option.text || option.Text || option.content || option.Content;
                const votes = parseInt(option.votes) || 0;
                // 直接计算百分比，确保实时准确
                const percent = calculatePercentage(votes, totalVotes);
                
                return (
                  <div key={optionId} className="result-item">
                    <div className="result-header">
                      <span className="option-text">{optionText}</span>
                      <span className="vote-stats">
                        <strong>{votes}</strong> 票 ({percent.toFixed(2)}%)
                      </span>
                    </div>
                    <Progress
                      key={`progress-${optionId}-${votes}`}
                      percent={percent}
                      status="active"
                      showInfo={true}
                      format={() => `${percent.toFixed(2)}%`}
                      strokeColor={{
                        '0%': '#108ee9',
                        '100%': '#87d068',
                      }}
                      trailColor="#f5f5f5"
                      strokeWidth={15}
                    />
                  </div>
                );
              })}
            </TabPane>
            
            <TabPane 
              tab={<span><PieChartOutlined /> 饼图</span>} 
              key="pie"
            >
              <div className="chart-container">
                {poll.options && poll.options.length > 0 ? (
                  <ResponsiveContainer width="100%" height={300}>
                    <PieChart>
                      <Pie
                        data={prepareChartData()}
                        cx="50%"
                        cy="50%"
                        labelLine={true}
                        label={({ name, value, percent }) => {
                          // 使用已经计算好的百分比
                          return `${name} (${Math.round(percent * 100)}%)`;
                        }}
                        outerRadius={80}
                        fill="#8884d8"
                        dataKey="value"
                      >
                        {poll.options.map((_, index) => (
                          <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                        ))}
                      </Pie>
                      <Tooltip 
                        formatter={(value, name, props) => {
                          // 直接使用数据点中包含的百分比
                          const percent = props.payload.percent * 100;
                          return [`${value} 票 (${Math.round(percent)}%)`, '投票数'];
                        }} 
                      />
                      <Legend />
                    </PieChart>
                  </ResponsiveContainer>
                ) : (
                  <div className="no-data">暂无投票数据</div>
                )}
              </div>
            </TabPane>
            
            <TabPane 
              tab={<span><BarChartOutlined /> 柱状图</span>} 
              key="column"
            >
              <div className="chart-container">
                {poll.options && poll.options.length > 0 ? (
                  <ResponsiveContainer width="100%" height={300}>
                    <BarChart
                      data={prepareChartData()}
                      margin={{
                        top: 5,
                        right: 30,
                        left: 20,
                        bottom: 5,
                      }}
                    >
                      <CartesianGrid strokeDasharray="3 3" />
                      <XAxis dataKey="name" />
                      <YAxis />
                      <Tooltip 
                        formatter={(value, name, props) => {
                          // 直接使用数据点中包含的百分比
                          const percent = props.payload.percent * 100;
                          return [`${value} 票 (${Math.round(percent)}%)`, '投票数'];
                        }}
                      />
                      <Legend />
                      <Bar dataKey="value" fill="#8884d8" name="票数">
                        {poll.options.map((_, index) => (
                          <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                        ))}
                      </Bar>
                    </BarChart>
                  </ResponsiveContainer>
                ) : (
                  <div className="no-data">暂无投票数据</div>
                )}
              </div>
            </TabPane>
          </Tabs>
        </div>
      </Card>
    </div>
  );
};

export default PollDetail; 