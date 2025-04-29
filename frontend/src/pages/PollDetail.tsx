import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { 
  Container, 
  Typography, 
  Button, 
  Card, 
  CardContent, 
  RadioGroup, 
  FormControlLabel, 
  Radio, 
  FormControl, 
  FormLabel,
  Box,
  Alert,
  LinearProgress,
  Divider,
  Chip,
  CircularProgress,
  Paper,
  Checkbox,
  alpha,
  useTheme,
  Grid,
  Tooltip,
  IconButton,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  DialogContentText,
  FormHelperText,
  Slider
} from '@mui/material';
import { getPoll, submitVote, deletePoll } from '../api/pollsApi';
import { Poll, PollOptionResult, PollVoteRequest } from '../types';
import { sseService } from '../services/sseService';
import CalendarTodayIcon from '@mui/icons-material/CalendarToday';
import StopIcon from '@mui/icons-material/Stop';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import EditIcon from '@mui/icons-material/Edit';
import CloseIcon from '@mui/icons-material/Close';
import SimulationIcon from '@mui/icons-material/Psychology';
import SettingsIcon from '@mui/icons-material/Settings';
import MenuItem from '@mui/material/MenuItem';
import Select from '@mui/material/Select';
import InputLabel from '@mui/material/InputLabel';
import TextField from '@mui/material/TextField';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import DeleteIcon from '@mui/icons-material/Delete';
import AddIcon from '@mui/icons-material/Add';
import SaveIcon from '@mui/icons-material/Save';
import Switch from '@mui/material/Switch';
import { DateTimePicker } from '@mui/x-date-pickers/DateTimePicker';
import { LocalizationProvider } from '@mui/x-date-pickers/LocalizationProvider';
import { AdapterDateFns } from '@mui/x-date-pickers/AdapterDateFns';
import { format } from 'date-fns';
import { zhCN } from 'date-fns/locale';
import PollChart from '../components/PollChart';

// 调试日志函数，便于排查数据问题
const logDebug = (message: string, data: any) => {
  console.log(`[PollDetail] ${message}:`, data);
};

// 增加调试信息显示组件
const RequestDebugInfo = ({ apiDebugInfo }: { apiDebugInfo: string | null }) => {
  const [expanded, setExpanded] = useState<boolean>(false);
  
  if (!apiDebugInfo) return null;
  
  const toggleExpanded = () => setExpanded(!expanded);
  
  return (
    <Box sx={{ 
      mt: 2, 
      bgcolor: 'rgba(0,0,0,0.03)', 
      border: '1px solid rgba(0,0,0,0.1)',
      borderRadius: 1,
      p: 2,
      maxHeight: expanded ? 'none' : '200px',
      overflowY: 'auto',
      position: 'relative',
      fontSize: '12px',
      fontFamily: 'monospace',
      whiteSpace: 'pre-wrap'
    }}>
      <Typography variant="subtitle2" gutterBottom>
        请求调试信息:
      </Typography>
      {apiDebugInfo.split('\n').map((line, i) => (
        <Box key={i} sx={{ 
          mb: 0.5,
          color: 
            line.includes('[错误]') ? 'error.main' : 
            line.includes('[警告]') ? 'warning.main' :
            line.includes('[成功]') ? 'success.main' :
            line.includes('[检查]') ? 'info.main' : 
            'text.primary'
        }}>
          {line}
        </Box>
      ))}
      {!expanded && (
        <Box 
          sx={{ 
            position: 'absolute', 
            bottom: 0, 
            left: 0, 
            right: 0, 
            textAlign: 'center',
            py: 1,
            background: 'linear-gradient(transparent, rgba(255,255,255,0.9) 50%)',
            cursor: 'pointer'
          }}
          onClick={toggleExpanded}
        >
          <Button size="small" variant="text">
            显示更多 ▼
          </Button>
        </Box>
      )}
      {expanded && (
        <Box sx={{ textAlign: 'center', mt: 1 }}>
          <Button size="small" variant="text" onClick={toggleExpanded}>
            收起 ▲
          </Button>
        </Box>
      )}
    </Box>
  );
};

const PollDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const theme = useTheme();
  const [poll, setPoll] = useState<Poll | null>(null);
  const [selectedOptions, setSelectedOptions] = useState<string[]>([]);
  const [results, setResults] = useState<PollOptionResult[]>([]);
  const [voted, setVoted] = useState<boolean>(false);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [submitLoading, setSubmitLoading] = useState<boolean>(false);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [liveUpdateActive, setLiveUpdateActive] = useState<boolean>(false);
  const [apiDebugInfo, setApiDebugInfo] = useState<string | null>(null); // 新增：调试信息
  const [showDebug, setShowDebug] = useState<boolean>(false);
  const [simulating, setSimulating] = useState(false);
  const [simulationInterval, setSimulationInterval] = useState<NodeJS.Timeout | null>(null);
  const [virtualSimulating, setVirtualSimulating] = useState(false);
  const [virtualSimulationInterval, setVirtualSimulationInterval] = useState<NodeJS.Timeout | null>(null);
  const [showEditDialog, setShowEditDialog] = useState(false);
  const [editTitle, setEditTitle] = useState('');
  const [editOptions, setEditOptions] = useState<{ text: string; id?: number }[]>([]);
  const [editPollType, setEditPollType] = useState(0);
  const [editDescription, setEditDescription] = useState('');
  const [editActive, setEditActive] = useState(true);
  const [editValidationErrors, setEditValidationErrors] = useState<any>({});
  const [showSimulationDialog, setShowSimulationDialog] = useState(false);
  const [simulationType, setSimulationType] = useState<'virtual' | 'highConcurrency'>('virtual');
  const [votesToAdd, setVotesToAdd] = useState(10);
  const [votingInterval, setVotingInterval] = useState(1000);
  const [saveSimulationResults, setSaveSimulationResults] = useState(true);
  const [simulationProgress, setSimulationProgress] = useState(0);
  const [simulationResults, setSimulationResults] = useState<PollOptionResult[]>([]);
  const [isProcessingResults, setIsProcessingResults] = useState(false);
  const [chartType, setChartType] = useState<'pie' | 'bar'>('pie');

  useEffect(() => {
    const fetchPoll = async () => {
      if (!id) return;
      
      try {
        setLoading(true);
        const pollId = parseInt(id, 10);
        logDebug("获取投票详情，ID", pollId);
        setApiDebugInfo(`正在请求 API: /api/polls/${pollId}`);
        
        const data = await getPoll(pollId);
        logDebug("获取到的原始数据", data);
        setApiDebugInfo(`API响应成功: ${JSON.stringify(data).substring(0, 100)}...`);
        
        // 如果返回的数据为空或无效
        if (!data || (typeof data === 'object' && Object.keys(data).length === 0)) {
          setError('服务器返回了空数据，请检查API是否正常工作');
          return;
        }
        
        // 深度复制并转换后端模型字段到前端模型字段
        const processedData = {
          ...data,
          id: data.id || data.ID,
          title: data.title || data.question,
          active: data.active !== undefined ? data.active : data.is_active,
          poll_type: data.poll_type !== undefined ? parseInt(data.poll_type.toString(), 10) : (data.type !== undefined ? parseInt(data.type.toString(), 10) : 0),
          options: Array.isArray(data.options) ? data.options.map(opt => ({
            ...opt,
            id: opt.id || opt.ID,
            text: opt.text || opt.option_text
          })) : []
        };
        
        logDebug("处理后的数据", processedData);
        setPoll(processedData);
        
        // 如果后端直接返回了结果数据，则显示结果视图
        if (data.results && data.results.length > 0) {
          logDebug("后端返回了结果数据", data.results);
          setResults(data.results);
          setVoted(true);
        } else if (data.options && data.options.some(opt => (opt.votes || 0) > 0)) {
          // 如果选项中有投票数大于0，则构造结果对象
          logDebug("选项中有投票数", data.options);
          const totalVotes = data.options.reduce((sum, opt) => sum + (opt.votes || 0), 0);
          const constructed = data.options.map(opt => ({
            id: opt.id || opt.ID,
            option_id: opt.id || opt.ID,
            text: opt.text || opt.option_text,
            votes: opt.votes || 0,
            percentage: totalVotes > 0 ? ((opt.votes || 0) / totalVotes) * 100 : 0
          }));
          logDebug("构造的结果数据", constructed);
          setResults(constructed);
          setVoted(totalVotes > 0);
        } else {
          // 如果没有投票数，也构造初始化为0的结果对象，但不显示结果视图
          logDebug("没有投票数据，初始化为0", data.options);
          const constructed = data.options.map(opt => ({
            id: opt.id || opt.ID,
            option_id: opt.id || opt.ID,
            text: opt.text || opt.option_text,
            votes: 0,
            percentage: 0
          }));
          logDebug("初始化的结果数据", constructed);
          setResults(constructed);
          // 关键修改：设置为未投票状态，而不是根据总票数决定
          setVoted(false);
        }
        
        // 初始化编辑对话框数据
        setEditTitle(data.title || data.question || '');
        setEditDescription(data.description || '');
        setEditPollType(data.poll_type || data.type || 0);
        setEditActive(data.active || data.is_active || false);
        setEditOptions(data.options.map(opt => ({
          text: opt.text || opt.option_text || '',
          id: opt.id || opt.ID
        })));
      
      } catch (err: any) {
        console.error('Failed to fetch poll:', err);
        // 增强错误消息，提供更详细的信息
        let errorMessage = '获取投票详情失败';
        if (err.response) {
          // 服务器响应了错误状态码
          errorMessage += `，服务器返回: ${err.response.status} ${err.response.statusText}`;
          setApiDebugInfo(`API错误: ${err.response.status} - ${JSON.stringify(err.response.data || {})}`);
        } else if (err.request) {
          // 请求发送了但没有收到响应
          errorMessage += '，没有收到服务器响应，请检查后端服务是否运行';
          setApiDebugInfo('API错误: 请求已发送但没有收到响应，可能是CORS问题或后端服务未运行');
        } else {
          // 请求设置时出现错误
          errorMessage += `，${err.message || '未知错误'}`;
          setApiDebugInfo(`API错误: ${err.message || '未知错误'}`);
        }
        setError(errorMessage);
      } finally {
        setLoading(false);
      }
    };

    fetchPoll();

    // 组件卸载时清理
    return () => {
      sseService.disconnect();
    };
  }, [id]);

  // 连接SSE实时更新
  useEffect(() => {
    // 修复：移除 !voted 条件，只要 poll 数据加载完成就尝试连接
    if (!id || !poll) return;

    const pollId = parseInt(id, 10);

    console.log(`[PollDetail] 尝试连接SSE，投票ID: ${pollId}`);
    setApiDebugInfo(prev => prev + `\\n尝试连接SSE: 投票ID=${pollId}`); // 移除 voted 状态显示

    // 强制关闭模拟模式，只有在本地调试且明确设置了环境变量时才使用模拟
    const useMockSSE = process.env.NODE_ENV === 'development' && process.env.REACT_APP_MOCK_SSE === 'true';

    if (useMockSSE) {
      console.log('[PollDetail] 使用模拟SSE数据(开发模式)');
      setApiDebugInfo(prev => prev + '\\n使用模拟SSE数据(开发模式)');

      // 保留当前的票数数据，不重置为0
      const mockData = [...results];

      // 如果没有结果数据，则从poll选项构造初始数据
      if (mockData.length === 0 && poll.options) {
        poll.options.forEach(opt => {
          const optionId = opt.id || opt.ID;
          if (optionId) {
            mockData.push({
              id: optionId,
              option_id: optionId,
              text: opt.text || opt.option_text || '',
              votes: opt.votes || 0,
              percentage: 0
            });
          }
        });

        // 计算初始百分比
        const totalVotes = mockData.reduce((sum, opt) => sum + opt.votes, 0);
        if (totalVotes > 0) {
          mockData.forEach(opt => {
            opt.percentage = (opt.votes / totalVotes) * 100;
          });
        }
      }

      // 模拟SSE连接成功但保留现有数据
      setTimeout(() => {
        setLiveUpdateActive(true);
        if (mockData.length > 0) {
          setResults(mockData);
        }
        setApiDebugInfo(prev => prev + '\\n模拟SSE连接成功，保留现有数据');
      }, 500);

      return () => {
        if (simulationInterval) {
          clearInterval(simulationInterval);
          setSimulationInterval(null);
        }
         if (virtualSimulationInterval) { // 也清理虚拟模拟定时器
           clearInterval(virtualSimulationInterval);
           setVirtualSimulationInterval(null);
         }
      };
    } else {
      // 真实SSE连接
      console.log('[PollDetail] 使用真实SSE连接');
      setApiDebugInfo(prev => prev + '\\n尝试建立真实SSE连接');

      sseService.connect(pollId, {
        onMessage: (data) => {
          console.log("[PollDetail] 收到SSE更新", data);
          setApiDebugInfo(prev => prev + `\\n收到SSE更新: ${JSON.stringify(data).substring(0, 100)}...`);

          // 确保收到的数据有效，否则不更新UI
          if (Array.isArray(data) && data.length > 0) {
            // 检查数据有效性
            const isValid = data.every(item =>
              (item.id !== undefined || item.option_id !== undefined) &&
              item.votes !== undefined
            );

            if (isValid) {
              setResults(data);
            } else {
              console.error("[PollDetail] 收到无效的SSE数据", data);
              setApiDebugInfo(prev => prev + '\\n警告：收到无效的SSE数据，不更新UI');
            }
          } else {
            console.warn("[PollDetail] 收到空的SSE数据", data);
            setApiDebugInfo(prev => prev + '\\n警告：收到空的SSE数据，不更新UI');
          }
        },
        onOpen: () => {
          console.log("[PollDetail] SSE连接已打开");
          setApiDebugInfo(prev => prev + '\\nSSE连接已打开');
          setLiveUpdateActive(true);
        },
        onError: (err) => {
          console.error("[PollDetail] SSE连接错误", err);
          setApiDebugInfo(prev => prev + `\\nSSE连接错误: ${err}`);
          setLiveUpdateActive(false);

          // 连接失败后切换到模拟模式，但保留现有数据
          console.log('[PollDetail] SSE连接失败，切换到模拟模式');
          setApiDebugInfo(prev => prev + '\\nSSE连接失败，切换到模拟模式但保留现有数据');

          // 使用当前的结果数据，不重置
          const currentData = [...results];

          // 如果没有结果数据，才从poll选项构造初始数据
          if (currentData.length === 0 && poll.options) {
            poll.options.forEach(opt => {
              const optionId = opt.id || opt.ID;
              if (optionId) {
                currentData.push({
                  id: optionId,
                  option_id: optionId,
                  text: opt.text || opt.option_text || '',
                  votes: opt.votes || 0,
                  percentage: 0
                });
              }
            });

            // 计算初始百分比
            const totalVotes = currentData.reduce((sum, opt) => sum + opt.votes, 0);
            if (totalVotes > 0) {
              currentData.forEach(opt => {
                opt.percentage = (opt.votes / totalVotes) * 100;
              });
            }
          }

          // 恢复连接状态并保留现有数据
          setTimeout(() => {
            setLiveUpdateActive(true);
            if (currentData.length > 0) {
              setResults(currentData);
            }
          }, 500);
        }
      });

      // 清理函数
      return () => {
        console.log("[PollDetail] 组件卸载，断开SSE连接");
        sseService.disconnect();

        // 确保相关的模拟定时器也被清理 (虽然我们可能稍后简化模拟)
        if (simulationInterval) {
          clearInterval(simulationInterval);
          setSimulationInterval(null);
        }
         if (virtualSimulationInterval) { // 也清理虚拟模拟定时器
           clearInterval(virtualSimulationInterval);
           setVirtualSimulationInterval(null);
         }
      };
    }
    // 修复：修改依赖数组，只依赖 id 和 poll
  }, [id, poll]);

  const handleOptionChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const value = event.target.value;
    
    // 如果是单选
    if (poll && (poll.poll_type === 0 || poll.poll_type === undefined)) {
      setSelectedOptions([value]);
    } else {
      // 如果是多选
      setSelectedOptions(prev => {
        if (prev.includes(value)) {
          return prev.filter(option => option !== value);
        } else {
          return [...prev, value];
        }
      });
    }
  };

  const handleVote = async () => {
    if (!poll || selectedOptions.length === 0 || !id) return;
    
    try {
      setSubmitLoading(true);
      setSubmitError(null);
      
      const pollId = parseInt(id, 10);
      let voteData: PollVoteRequest;
      
      // 根据投票类型构建请求
      if (poll.poll_type === 0) {
        // 单选
        voteData = {
          option_id: parseInt(selectedOptions[0], 10),
          option_ids: [parseInt(selectedOptions[0], 10)]
        };
      } else {
        // 多选
        voteData = {
          option_ids: selectedOptions.map(opt => parseInt(opt, 10))
        };
      }
      
      logDebug("提交投票数据", voteData);
      const response = await submitVote(pollId, voteData);
      logDebug("投票响应", response);
      
      // 更新结果并显示结果视图
      if (response.results) {
        setResults(response.results);
      } else if (response.current_results) {
        setResults(response.current_results);
      } else {
        // 如果后端没有返回结果，重新获取投票详情
        const updatedPoll = await getPoll(pollId);
        if (updatedPoll.options) {
          const totalVotes = updatedPoll.options.reduce((sum, opt) => sum + (opt.votes || 0), 0);
          const constructed = updatedPoll.options.map(opt => ({
            id: opt.id || opt.ID,
            option_id: opt.id || opt.ID,
            text: opt.text || opt.option_text,
            votes: opt.votes || 0,
            percentage: totalVotes > 0 ? ((opt.votes || 0) / totalVotes) * 100 : 0
          }));
          setResults(constructed);
        }
      }
      setVoted(true);
      
    } catch (err: any) {
      console.error('Failed to submit vote:', err);
      let errorMessage = '提交投票失败';
      if (err.response) {
        errorMessage += `，服务器返回: ${err.response.status} ${err.response.statusText}`;
      } else if (err.request) {
        errorMessage += '，没有收到服务器响应';
      } else {
        errorMessage += `，${err.message || '未知错误'}`;
      }
      setSubmitError(errorMessage);
    } finally {
      setSubmitLoading(false);
    }
  };

  const handleBack = () => {
    navigate('/');
  };

  const handleDelete = async () => {
    if (!poll || !id) return;
    
    if (window.confirm('确定要删除这个投票吗？此操作不可撤销。')) {
      try {
        await deletePoll(parseInt(id, 10));
        navigate('/');
      } catch (err) {
        console.error('Failed to delete poll:', err);
        alert('删除投票失败，请稍后再试');
      }
    }
  };

  // 打开模拟对话框
  const openSimulationDialog = () => {
    setShowSimulationDialog(true);
  };
  
  // 关闭模拟对话框
  const closeSimulationDialog = () => {
    setShowSimulationDialog(false);
  };
  
  // 处理模拟类型变更
  const handleSimulationTypeChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setSimulationType(event.target.value as 'virtual' | 'highConcurrency');
  };
  
  // 处理保存结果开关变更
  const handleSaveResultsChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setSaveSimulationResults(event.target.checked);
  };
  
  // 开始模拟
  const startSimulation = () => {
    if (simulating || virtualSimulating) {
      // 如果已经在模拟中，先停止
      stopSimulation();
    }
    
    if (simulationType === 'virtual') {
      startVirtualSimulation();
    } else {
      startHighConcurrencySimulation();
    }
    
    // 关闭对话框
    setShowSimulationDialog(false);
  };
  
  // 停止所有模拟
  const stopSimulation = () => {
    // 停止虚拟模拟
    if (virtualSimulationInterval) {
      clearInterval(virtualSimulationInterval);
      setVirtualSimulationInterval(null);
    }
    setVirtualSimulating(false);
    
    // 停止高并发测试
    if (simulationInterval) {
      clearInterval(simulationInterval);
      setSimulationInterval(null);
    }
    setSimulating(false);
    
    // 重置模拟状态
    setSimulationResults([]);
    setSimulationProgress(0);
    setIsProcessingResults(false);
    
    // 日志记录
    setApiDebugInfo(prev => prev + '\n已停止模拟，恢复原始数据状态');
  };
  
  // 开始虚拟模拟
  const startVirtualSimulation = () => {
    setVirtualSimulating(true);
    setApiDebugInfo(prev => prev + `\n开始界面模拟 - 仅前端临时显示，不影响实际数据`);
    
    // 复制当前结果作为起点，不要修改原始结果
    const mockData = [...results];
    
    // 使用单独的状态存储模拟数据，不影响原始结果
    const simulationData = [...mockData];
    setSimulationResults(simulationData);
    
    // 显示结果视图
    if (!voted) {
      setVoted(true);
    }
    
    // 每设定的间隔随机更新一次数据
    const interval = setInterval(() => {
      if (simulationData.length === 0) return;
      
      // 随机选择选项增加票数
      const updatesToMake = Math.min(
        votesToAdd, 
        Math.floor(Math.random() * votesToAdd) + 1
      );
      
      for (let i = 0; i < updatesToMake; i++) {
        const randomOptionIndex = Math.floor(Math.random() * simulationData.length);
        simulationData[randomOptionIndex].votes += 1;
        
        console.log(`[界面模拟] 选项 "${simulationData[randomOptionIndex].text}" 增加一票`);
      }
      
      // 重新计算百分比
      const totalVotes = simulationData.reduce((sum, opt) => sum + opt.votes, 0);
      simulationData.forEach(opt => {
        opt.percentage = (opt.votes / totalVotes) * 100;
      });
      
      // 更新模拟结果状态
      setSimulationResults([...simulationData]);
      
      // 记录日志
      setApiDebugInfo(prev => prev + `\n[界面模拟] 临时显示更新，总票数: ${totalVotes}`);
      
    }, votingInterval);
    
    setVirtualSimulationInterval(interval);
  };
  
  // 检查后端服务状态
  const checkBackendStatus = async () => {
    try {
      const API_BASE_URL = process.env.REACT_APP_API_BASE_URL || 'http://localhost:8090';
      console.log(`[后端检查] 尝试连接到 ${API_BASE_URL}`);
      setApiDebugInfo(prev => prev + `\n[检查] 尝试连接后端服务: ${API_BASE_URL}`);
      
      // 设置超时较短，快速检测
      const controller = new AbortController();
      const timeoutId = setTimeout(() => controller.abort(), 2000);
      
      // 使用根路径或/api/health路径
      const response = await fetch(`${API_BASE_URL}/api/health`, {
        method: 'GET',
        signal: controller.signal
      }).catch(async () => {
        // 如果/api/health失败，尝试直接访问根路径
        return await fetch(`${API_BASE_URL}`, {
          method: 'GET',
          signal: controller.signal
        });
      });
      
      clearTimeout(timeoutId);
      
      if (response.ok) {
        console.log('[后端检查] 连接成功');
        setApiDebugInfo(prev => prev + '\n[检查] 后端服务正常');
        return true;
      } else {
        console.warn(`[后端检查] 服务返回状态码: ${response.status}`);
        setApiDebugInfo(prev => prev + `\n[警告] 后端服务返回错误状态码: ${response.status}`);
        return false;
      }
    } catch (error: any) {
      console.error('[后端检查] 连接失败:', error);
      
      if (error.name === 'AbortError') {
        setApiDebugInfo(prev => prev + '\n[错误] 后端服务连接超时，请确认服务是否启动');
      } else {
        setApiDebugInfo(prev => prev + `\n[错误] 后端服务连接失败: ${error.message}`);
      }
      
      return false;
    }
  };

  // 修改提交真实请求方式，直接使用后端端口
  const startHighConcurrencySimulation = async () => {
    if (!poll || !poll.options || poll.options.length === 0 || !id) {
      console.error('[真实请求模拟] 缺少必要数据，无法启动模拟');
      setApiDebugInfo(prev => prev + '\n[错误] 缺少必要数据，无法启动模拟');
      return;
    }
    
    // 清空之前的API调试信息，方便追踪
    setApiDebugInfo('');
    
    // 先检查后端服务状态
    setApiDebugInfo('[检查] 开始检查后端服务状态...');
    const isBackendAvailable = await checkBackendStatus();
    
    if (!isBackendAvailable && saveSimulationResults) {
      if (!window.confirm('后端服务似乎无法访问。是否切换到"不保存结果"模式继续模拟？')) {
        setApiDebugInfo(prev => prev + '\n已取消模拟');
        return;
      }
      // 自动切换到不保存模式
      setSaveSimulationResults(false);
      setApiDebugInfo(prev => prev + '\n已切换到"不保存结果"模式');
    }
    
    setSimulating(true);
    setIsProcessingResults(true); // 设置处理状态，防止界面抖动
    setApiDebugInfo(prev => prev + `\n开始真实请求模拟 - ${saveSimulationResults ? '将保存到数据库' : '不保存到数据库'}`);
    
    // 始终使用当前结果作为起点，确保有初始数据
    const simulationData = results.length > 0 ? 
      [...results] : 
      poll?.options.map(opt => ({
        id: opt.id || opt.ID,
        option_id: opt.id || opt.ID,
        text: opt.text || '',
        votes: 0,
        percentage: 0
      })) || [];
    
    // 设置初始模拟结果
    setSimulationResults([...simulationData]);
    
    // 显示结果视图
    if (!voted) {
      setVoted(true);
    }
    
    // 模拟进度追踪
    let totalRequestsPlanned = 0;
    let completedRequests = 0;
    
    // 每设定的间隔发送一批请求
    const interval = setInterval(async () => {
      if (!poll || !poll.options || poll.options.length === 0 || !id) {
        clearInterval(interval);
        setSimulationInterval(null);
        setSimulating(false);
        setIsProcessingResults(false);
        return;
      }
      
      // 随机选择选项，决定这一批要发送多少请求
      const requestsToSend = Math.min(
        votesToAdd, 
        Math.floor(Math.random() * votesToAdd) + 1
      );
      
      totalRequestsPlanned += requestsToSend;
      setApiDebugInfo(prev => prev + `\n[请求] 准备并发发送 ${requestsToSend} 个请求`);
      
      // 如果不保存结果，仅更新前端显示
      if (!saveSimulationResults) {
        // 创建随机投票的临时模拟数据
        const updatedData = [...simulationResults];
        
        // 为每个要发送的请求生成随机投票
        for (let i = 0; i < requestsToSend; i++) {
          // 随机选择一个选项
          const randomOptionIndex = Math.floor(Math.random() * poll.options.length);
          const randomOption = poll.options[randomOptionIndex];
          
          // 确保选项有ID
          const optionId = randomOption.id || randomOption.ID;
          if (!optionId) {
            console.error('[真实请求模拟] 选项没有有效ID:', randomOption);
            continue;
          }
          
          // 更新临时模拟数据
          const optionToUpdate = updatedData.find(r => 
            (r.id === optionId || r.option_id === optionId)
          );
          
          if (optionToUpdate) {
            // 如果找到匹配的选项，增加票数
            optionToUpdate.votes += 1;
          } else {
            // 如果没有找到，添加新的选项
            updatedData.push({
              id: optionId,
              option_id: optionId,
              text: randomOption.text || '',
              votes: 1,
              percentage: 0
            });
          }
          
          completedRequests++;
          console.log(`[真实请求模拟] 模拟投票 ${completedRequests}/${totalRequestsPlanned}: 选项 "${randomOption.text}"`);
        }
        
        // 重新计算百分比
        const totalVotes = updatedData.reduce((sum, opt) => sum + opt.votes, 0);
        updatedData.forEach(opt => {
          opt.percentage = totalVotes > 0 ? (opt.votes / totalVotes) * 100 : 0;
        });
        
        // 更新模拟结果状态
        setSimulationResults([...updatedData]);
        
        // 更新进度
        setSimulationProgress(Math.floor((completedRequests / totalRequestsPlanned) * 100));
      } else {
        // 真实发送请求到后端
        const pollId = parseInt(id, 10);
        
        try {
          // 确保使用正确的API URL
          const API_BASE_URL = process.env.REACT_APP_API_BASE_URL || 'http://localhost:8090';
          console.log(`[真实请求模拟] 准备发送请求到: ${API_BASE_URL}/api/polls/${pollId}/vote`);
          setApiDebugInfo(prev => prev + `\n[API信息] 目标URL: ${API_BASE_URL}/api/polls/${pollId}/vote`);
          
          // 并行发送多个投票请求
          const requests = Array.from({ length: requestsToSend }).map(async (_, i) => {
            // 随机选择一个选项
            const randomOptionIndex = Math.floor(Math.random() * poll.options.length);
            const randomOption = poll.options[randomOptionIndex];
            
            // 确保选项有ID
            const optionId = randomOption.id || randomOption.ID;
            if (!optionId) {
              console.error(`[真实请求模拟] 请求 ${i+1}: 选项没有有效ID`, randomOption);
              return null;
            }
            
            try {
              console.log(`[真实请求模拟] 发送请求 ${i+1}/${requestsToSend}: 为选项 "${randomOption.text}" 投票`);
              
              // 构造投票数据 - 确保是标准格式
              const voteData = {
                option_id: optionId,
                option_ids: [optionId]
              };
              
              // 记录请求详情用于调试
              setApiDebugInfo(prev => prev + `\n[请求${i+1}] 发送POST到 ${API_BASE_URL}/api/polls/${pollId}/vote，数据: ${JSON.stringify(voteData)}`);
              
              // 使用axios和submitVote函数直接提交，增加调试日志
              try {
                const response = await submitVote(pollId, voteData);
                console.log(`[真实请求模拟] 请求 ${i+1} 成功:`, response);
                setApiDebugInfo(prev => prev + `\n[成功] 请求${i+1}成功，收到响应: ${JSON.stringify(response).substring(0, 100)}...`);
                
                completedRequests++;
                return response.current_results || response.results;
              } catch (axiosError: any) {
                // 请求失败，记录详细错误
                if (axiosError.response) {
                  setApiDebugInfo(prev => prev + 
                    `\n[错误] 请求${i+1}失败，状态码:${axiosError.response.status}，错误:${JSON.stringify(axiosError.response.data)}`);
                } else if (axiosError.request) {
                  setApiDebugInfo(prev => prev + 
                    `\n[错误] 请求${i+1}发送了但没收到响应，可能是网络问题或CORS限制`);
                } else {
                  setApiDebugInfo(prev => prev + 
                    `\n[错误] 请求${i+1}配置失败: ${axiosError.message}`);
                }
                
                completedRequests++;
                return null;
              }
            } catch (error: any) {
              console.error(`[真实请求模拟] 请求 ${i+1} 失败:`, error);
              setApiDebugInfo(prev => prev + `\n[错误] 请求 ${i+1} 整体失败: ${error.message}`);
              
              completedRequests++;
              return null;
            }
          });
          
          // 等待所有请求完成
          const results = await Promise.all(requests);
          const validResults = results.filter(Boolean);
          
          if (validResults.length > 0) {
            const lastValidResult = validResults[validResults.length - 1];
            if (lastValidResult) {
              setResults(lastValidResult);
              setSimulationResults(lastValidResult);
              console.log(`[真实请求模拟] 更新了结果数据`);
              setApiDebugInfo(prev => prev + `\n[成功] 收到有效结果并更新显示，有效响应数: ${validResults.length}/${requestsToSend}`);
            }
          } else {
            console.warn(`[真实请求模拟] 没有收到有效的结果数据`);
            setApiDebugInfo(prev => prev + `\n[警告] 所有请求都失败，没有收到有效结果`);
            
            // 如果所有请求都失败，自动切换到不保存模式
            if (completedRequests >= 5 && validResults.length === 0) {
              setSaveSimulationResults(false);
              setApiDebugInfo(prev => prev + `\n[系统] 检测到持续请求失败，已切换到"不保存结果"模式`);
            }
          }
        } catch (error: any) {
          console.error(`[真实请求模拟] 整体错误:`, error);
          setApiDebugInfo(prev => prev + `\n[错误] 模拟过程中发生错误: ${error.message}`);
        }
        
        // 更新进度
        setSimulationProgress(Math.floor((completedRequests / totalRequestsPlanned) * 100));
      }
      
      // 如果已完成所有计划的请求，停止模拟
      if (completedRequests >= totalRequestsPlanned) {
        clearInterval(interval);
        setSimulationInterval(null);
        setSimulating(false);
        setIsProcessingResults(false);
        setApiDebugInfo(prev => prev + `\n[真实请求模拟] 所有请求已完成，共 ${completedRequests} 个请求`);
      }
    }, votingInterval);
    
    setSimulationInterval(interval);
  };

  // 准备图表数据
  const prepareChartData = () => {
    if (!poll || !poll.options || !poll.results) return [];
    
    // 确定使用哪组数据
    const displayResults = simulating || virtualSimulating ? simulationResults : results;
    
    return displayResults.map(result => {
      // 查找对应的选项文本
      const option = poll.options.find(opt => 
        (opt.id || opt.ID) === (result.id || result.option_id)
      );
      
      return {
        name: option?.text || '未知选项',
        value: result.votes,
        id: result.id || result.option_id
      };
    });
  };

  // 渲染投票结果部分
  const renderResults = () => {
    // 确定使用哪组数据
    const displayResults = simulating || virtualSimulating ? simulationResults : results;
    
    if (!displayResults || displayResults.length === 0) {
      return <Typography>暂无投票数据</Typography>;
    }
    
    // 计算总票数
    const totalVotes = displayResults.reduce((sum, item) => sum + item.votes, 0);
    
    return (
      <>
        {displayResults.map((result) => {
          // 查找对应的选项文本
          const optionText = poll?.options.find(opt => 
            (opt.id || opt.ID) === (result.id || result.option_id)
          )?.text || '未知选项';
          
          // 计算百分比
          const percentage = totalVotes > 0 ? (result.votes / totalVotes) * 100 : 0;
          
          return (
            <Box key={result.id || result.option_id} sx={{ mb: 3 }}>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 0.5 }}>
                <Typography variant="body1" sx={{ fontWeight: 500 }}>
                  {optionText}
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  {result.votes} 票 ({Math.round(percentage)}%)
                </Typography>
              </Box>
              <LinearProgress 
                variant="determinate" 
                value={percentage} 
                sx={{ 
                  height: 10, 
                  borderRadius: 5,
                  bgcolor: alpha(theme.palette.primary.main, 0.15),
                  '& .MuiLinearProgress-bar': {
                    borderRadius: 5,
                    background: percentage > 50 
                      ? `linear-gradient(90deg, ${theme.palette.primary.main} 0%, ${theme.palette.primary.dark} 100%)`
                      : `linear-gradient(90deg, ${theme.palette.primary.light} 30%, ${theme.palette.primary.main} 100%)`
                  }
                }} 
              />
            </Box>
          );
        })}
        
        <Typography 
          variant="body2" 
          color="text.secondary" 
          sx={{ 
            mt: 3, 
            textAlign: 'right',
            fontWeight: 500,
            fontSize: '0.9rem'
          }}
        >
          总票数: {totalVotes}
        </Typography>
      </>
    );
  };

  // 格式化时间的函数
  const formatDate = (dateString?: string) => {
    if (!dateString) return '无截止日期';
    const date = new Date(dateString);
    return date.toLocaleString('zh-CN', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  };
  
  // 检查投票是否过期
  const isPollExpired = (poll: Poll) => {
    if (!poll.end_time) return false;
    const endTime = new Date(poll.end_time);
    return endTime < new Date();
  };

  // 打开编辑对话框
  const handleOpenEditDialog = () => {
    setShowEditDialog(true);
  };
  
  // 关闭编辑对话框
  const handleCloseEditDialog = () => {
    setShowEditDialog(false);
    setEditValidationErrors({});
  };
  
  // 处理标题变更
  const handleEditTitleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setEditTitle(e.target.value);
  };
  
  // 处理描述变更
  const handleEditDescriptionChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setEditDescription(e.target.value);
  };
  
  // 处理投票类型变更
  const handleEditPollTypeChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setEditPollType(parseInt(e.target.value, 10));
  };
  
  // 处理激活状态变更
  const handleEditActiveChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setEditActive(e.target.checked);
  };
  
  // 处理选项文本变更
  const handleEditOptionChange = (index: number, value: string) => {
    const newOptions = [...editOptions];
    newOptions[index].text = value;
    setEditOptions(newOptions);
  };
  
  // 添加新选项
  const handleAddEditOption = () => {
    setEditOptions([...editOptions, { text: '' }]);
  };
  
  // 删除选项
  const handleRemoveEditOption = (index: number) => {
    if (editOptions.length <= 2) {
      return; // 至少保留两个选项
    }
    const newOptions = [...editOptions];
    newOptions.splice(index, 1);
    setEditOptions(newOptions);
  };
  
  // 验证表单
  const validateEditForm = (): boolean => {
    const errors: {
      title?: string;
      options?: string[];
    } = {};
    let isValid = true;

    // 验证标题
    if (!editTitle.trim()) {
      errors.title = '请输入投票标题';
      isValid = false;
    }

    // 验证选项
    const optionErrors: string[] = [];
    let hasOptionError = false;

    editOptions.forEach((option, index) => {
      if (!option.text.trim()) {
        optionErrors[index] = '选项不能为空';
        hasOptionError = true;
        isValid = false;
      } else {
        optionErrors[index] = '';
      }
    });

    if (hasOptionError) {
      errors.options = optionErrors;
    }

    setEditValidationErrors(errors);
    return isValid;
  };
  
  // 保存编辑后的投票
  const handleSaveEdit = async () => {
    if (!validateEditForm() || !id || !poll) {
      return;
    }
    
    try {
      setSubmitLoading(true);
      setSubmitError(null);
      
      const pollData = {
        ID: parseInt(id, 10),
        Question: editTitle,
        PollType: parseInt(String(editPollType), 10),
        Description: editDescription.trim() ? editDescription : undefined,
        Options: editOptions.map((opt) => ({
          ID: opt.id,
          Text: opt.text
        })),
        is_active: editActive
      };
      
      // 添加日志以便调试
      console.log('准备发送更新请求，数据:', pollData);
      
      // 调用API更新投票
      const response = await fetch(`/api/polls/${id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(pollData),
      });
      
      if (!response.ok) {
        throw new Error(`服务器返回: ${response.status} ${response.statusText}`);
      }
      
      // 关闭对话框并重新获取投票数据
      setShowEditDialog(false);
      window.location.reload(); // 简单刷新页面以获取最新数据
      
    } catch (err: any) {
      console.error('Failed to update poll:', err);
      let errorMessage = '更新投票失败';
      if (err.response) {
        errorMessage += `，服务器返回: ${err.response.status} ${err.response.statusText}`;
      } else if (err.request) {
        errorMessage += '，没有收到服务器响应';
      } else {
        errorMessage += `，${err.message || '未知错误'}`;
      }
      setSubmitError(errorMessage);
    } finally {
      setSubmitLoading(false);
    }
  };

  if (loading) {
    return (
      <Container maxWidth="md" sx={{ 
        mt: 4, 
        textAlign: 'center', 
        minHeight: '70vh',
        display: 'flex',
        flexDirection: 'column',
        justifyContent: 'center',
        alignItems: 'center'
      }}>
        <CircularProgress size={60} thickness={4} />
        <Typography variant="h6" color="text.secondary" sx={{ mt: 3 }}>
          正在加载投票详情...
        </Typography>
        {apiDebugInfo && (
          <Typography variant="caption" color="text.secondary" sx={{ mt: 2, maxWidth: '80%', wordBreak: 'break-word' }}>
            {apiDebugInfo}
          </Typography>
        )}
      </Container>
    );
  }

  if (error || !poll) {
    return (
      <Container maxWidth="md" sx={{ mt: 4 }}>
        <Paper 
          elevation={0}
          sx={{ 
            p: 4, 
            borderRadius: '24px',
            border: '1px solid',
            borderColor: 'divider',
            textAlign: 'center',
            bgcolor: alpha(theme.palette.error.light, 0.05),
          }}
        >
          <Typography color="error" variant="h6" gutterBottom>
            {error || '找不到该投票'}
          </Typography>
          <Typography color="text.secondary" sx={{ mb: 3 }}>
            可能是投票已被删除、服务器未运行或您输入了错误的地址
          </Typography>
          {apiDebugInfo && (
            <Box sx={{ mb: 3, p: 2, bgcolor: 'background.paper', borderRadius: 2, maxWidth: '80%', mx: 'auto' }}>
              <Typography variant="caption" color="text.secondary" sx={{ wordBreak: 'break-word' }}>
                调试信息: {apiDebugInfo}
              </Typography>
            </Box>
          )}
          <Button 
            variant="contained" 
            onClick={handleBack}
            sx={{ 
              borderRadius: '20px',
              textTransform: 'none',
              px: 4
            }}
          >
            返回列表
          </Button>
        </Paper>
      </Container>
    );
  }

  // 渐变背景色
  const gradientStyle = {
    background: 'linear-gradient(145deg, rgba(255,255,255,0.8) 0%, rgba(249,250,251,0.9) 100%)',
    backdropFilter: 'blur(20px)'
  };

  // 页面背景样式
  const pageBackground = {
    position: 'fixed',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    background: 'linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%)',
    zIndex: -1,
  };

  // 添加在组件return之前以便开发时查看
  console.log("Poll对象内容:", poll);
  
  // 添加显示调试信息的组件
  const debugInfo = process.env.NODE_ENV === 'development' && (
    <Box sx={{ 
      mt: 2,
      p: 2, 
      bgcolor: 'rgba(0,0,0,0.05)',
      borderRadius: 2,
      overflow: 'auto',
      fontSize: '12px',
      fontFamily: 'monospace',
      display: showDebug ? 'block' : 'none'
    }}>
      <Button 
        size="small" 
        onClick={() => setShowDebug(!showDebug)}
        sx={{ mb: 1 }}
      >
        {showDebug ? '隐藏调试信息' : '显示调试信息'}
      </Button>
      {showDebug && (
        <pre>{JSON.stringify(poll, null, 2)}</pre>
      )}
    </Box>
  );

  return (
    <>
      <Box sx={pageBackground} />
      
      <Container maxWidth="md" sx={{ my: 4 }}>
        <Box sx={{ 
          mb: 4,
          px: 3,
          py: 2,
          borderRadius: '24px',
          ...gradientStyle,
          boxShadow: '0 4px 20px rgba(0,0,0,0.05)'
        }}>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
            <Button 
              variant="outlined" 
              onClick={handleBack} 
              sx={{ 
                borderRadius: '20px',
                textTransform: 'none',
                px: 3
              }}
            >
              ← 返回列表
            </Button>
            
            <Button
              variant="outlined"
              color="primary"
              startIcon={<EditIcon />}
              onClick={handleOpenEditDialog}
              sx={{ 
                borderRadius: '20px',
                textTransform: 'none',
                px: 3
              }}
            >
              编辑投票
            </Button>
          </Box>
          
          <Typography 
            variant="h4" 
            component="h1" 
            gutterBottom
            sx={{ 
              fontWeight: 500,
              letterSpacing: '-0.5px',
              color: theme.palette.primary.dark
            }}
          >
            {poll.title || poll.question}
          </Typography>
          
          {poll.description && (
            <Typography variant="body1" sx={{ mb: 2, color: theme.palette.text.secondary }}>
              {poll.description}
            </Typography>
          )}
          
          <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap', mt: 2 }}>
            <Chip
              label={`投票类型: ${parseInt(String(poll.poll_type || '0'), 10) === 0 ? '单选' : '多选'}`}
              sx={{ 
                borderRadius: '20px',
                bgcolor: alpha(theme.palette.primary.main, 0.1),
                color: theme.palette.primary.main,
                fontWeight: 500
              }}
            />
            <Chip
              label={(poll.active || poll.is_active) ? "进行中" : "未开始"}
              sx={{ 
                borderRadius: '20px',
                bgcolor: (poll.active || poll.is_active) ? 
                         alpha(theme.palette.success.main, 0.1) : 
                         alpha(theme.palette.warning.main, 0.1),
                color: (poll.active || poll.is_active) ? 
                       theme.palette.success.main : 
                       theme.palette.warning.main,
                fontWeight: 500
              }}
            />
          </Box>
        </Box>

        {!voted ? (
          <Paper 
            elevation={0} 
            sx={{ 
              p: 4, 
              mb: 4, 
              borderRadius: '24px',
              border: '1px solid',
              borderColor: 'divider',
              ...gradientStyle
            }}
          >
            <FormControl component="fieldset" sx={{ width: '100%' }}>
              <FormLabel 
                component="legend"
                sx={{ 
                  fontSize: '1.1rem', 
                  mb: 2,
                  color: theme.palette.text.primary,
                  fontWeight: 500,
                  '&.Mui-focused': {
                    color: theme.palette.primary.main
                  }
                }}
              >
                {parseInt(String(poll.poll_type || '0'), 10) === 0 ? 
                  '选择一个选项进行投票：' : 
                  '选择一个或多个选项进行投票：'}
              </FormLabel>
              {parseInt(String(poll.poll_type || '0'), 10) === 0 ? (
                <RadioGroup
                  value={selectedOptions[0] || ''}
                  onChange={handleOptionChange}
                  sx={{ mt: 2 }}
                >
                  {poll.options.map((option, index) => (
                    <FormControlLabel
                      key={option.id || option.ID || `option-${index}`}
                      value={(option.id || option.ID || 0).toString()}
                      control={
                        <Radio 
                          sx={{ 
                            '&.Mui-checked': { color: theme.palette.primary.main },
                            '& .MuiSvgIcon-root': { fontSize: 22 }
                          }} 
                        />
                      }
                      label={
                        <Typography sx={{ fontSize: '1rem' }}>
                          {option.text}
                        </Typography>
                      }
                      disabled={!(poll.active || poll.is_active)}
                      sx={{ 
                        mb: 1.5,
                        py: 0.5,
                        px: 1,
                        borderRadius: 2,
                        transition: 'background-color 0.2s',
                        '&:hover': {
                          bgcolor: alpha(theme.palette.primary.main, 0.04)
                        },
                        '&.Mui-checked': {
                          bgcolor: alpha(theme.palette.primary.main, 0.08)
                        },
                        '& .MuiFormControlLabel-label': { flex: 1 }
                      }}
                    />
                  ))}
                </RadioGroup>
              ) : (
                <Box sx={{ mt: 2 }}>
                  {poll.options.map((option, index) => (
                    <FormControlLabel
                      key={option.id || option.ID || `option-${index}`}
                      value={(option.id || option.ID || 0).toString()}
                      control={
                        <Checkbox 
                          checked={selectedOptions.includes((option.id || option.ID || 0).toString())}
                          onChange={handleOptionChange}
                          sx={{ 
                            '&.Mui-checked': { color: theme.palette.primary.main },
                            '& .MuiSvgIcon-root': { fontSize: 22 }
                          }} 
                        />
                      }
                      label={
                        <Typography sx={{ fontSize: '1rem' }}>
                          {option.text}
                        </Typography>
                      }
                      disabled={!(poll.active || poll.is_active)}
                      sx={{ 
                        mb: 1.5,
                        display: 'flex',
                        py: 0.5,
                        px: 1,
                        borderRadius: 2,
                        transition: 'background-color 0.2s',
                        '&:hover': {
                          bgcolor: alpha(theme.palette.primary.main, 0.04)
                        },
                        '& .MuiFormControlLabel-label': { flex: 1 }
                      }}
                    />
                  ))}
                </Box>
              )}
            </FormControl>

            {submitError && (
              <Alert 
                severity="error" 
                sx={{ 
                  mt: 2, 
                  borderRadius: '12px',
                  boxShadow: '0 2px 10px rgba(0,0,0,0.05)'
                }}
              >
                {submitError}
              </Alert>
            )}

            <Box sx={{ mt: 3 }}>
              <Button
                variant="contained"
                color="primary"
                onClick={handleVote}
                disabled={
                  selectedOptions.length === 0 || 
                  submitLoading || 
                  !(poll.active || poll.is_active)
                }
                sx={{ 
                  mr: 2, 
                  borderRadius: '20px',
                  textTransform: 'none',
                  py: 1.2,
                  px: 4,
                  fontWeight: 500,
                  boxShadow: '0 4px 10px rgba(25, 118, 210, 0.2)',
                  '&:hover': {
                    boxShadow: '0 6px 15px rgba(25, 118, 210, 0.3)'
                  }
                }}
              >
                {submitLoading ? 
                  <Box sx={{ display: 'flex', alignItems: 'center' }}>
                    <CircularProgress size={16} color="inherit" sx={{ mr: 1 }} />
                    提交中...
                  </Box> : 
                  '提交投票'
                }
              </Button>
            </Box>
          </Paper>
        ) : (
          <Paper 
            elevation={0} 
            sx={{ 
              p: 4, 
              mb: 4, 
              borderRadius: '24px',
              border: '1px solid',
              borderColor: 'divider',
              ...gradientStyle
            }}
          >
            <Typography 
              variant="h6" 
              gutterBottom
              sx={{ 
                fontWeight: 500,
                display: 'flex',
                alignItems: 'center',
                color: theme.palette.primary.dark
              }}
            >
              投票结果
              {liveUpdateActive && (
                <Chip 
                  label="实时" 
                  color="info" 
                  size="small"
                  sx={{ ml: 2, height: 20, fontSize: '0.7rem', borderRadius: '10px' }}
                />
              )}
            </Typography>
            <Divider sx={{ mb: 3 }} />
            
            {/* 图表类型选择 */}
            <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 2 }}>
              <FormControl component="fieldset">
                <RadioGroup
                  row
                  value={chartType}
                  onChange={(e) => setChartType(e.target.value as 'pie' | 'bar')}
                >
                  <FormControlLabel value="pie" control={<Radio />} label="饼图" />
                  <FormControlLabel value="bar" control={<Radio />} label="柱状图" />
                </RadioGroup>
              </FormControl>
            </Box>
            
            {/* 图表显示 */}
            <Box sx={{ mb: 4 }}>
              <PollChart data={prepareChartData()} type={chartType} />
            </Box>
            
            {/* 进度条显示 */}
            {renderResults()}
          </Paper>
        )}
        
        {(poll.active || poll.is_active) && (
          <Box sx={{ mt: 2, textAlign: 'right' }}>
            <Button 
              variant="outlined" 
              color="error" 
              onClick={handleDelete}
              sx={{ 
                borderRadius: '20px',
                textTransform: 'none',
                py: 1,
                px: 3,
                borderColor: alpha(theme.palette.error.main, 0.5),
                '&:hover': {
                  borderColor: theme.palette.error.main,
                  backgroundColor: alpha(theme.palette.error.main, 0.04)
                }
              }}
            >
              删除此投票
            </Button>
          </Box>
        )}
      </Container>
      {debugInfo}

      {/* 编辑投票对话框 */}
      <Dialog open={showEditDialog} onClose={handleCloseEditDialog} maxWidth="md" fullWidth>
        <DialogTitle>编辑投票</DialogTitle>
        
        <DialogContent dividers>
          <LocalizationProvider dateAdapter={AdapterDateFns}>
            <TextField
              margin="normal"
              required
              fullWidth
              id="editTitle"
              label="投票标题"
              value={editTitle}
              onChange={handleEditTitleChange}
              error={!!editValidationErrors.title}
              helperText={editValidationErrors.title}
              autoFocus
            />
            
            <TextField
              margin="normal"
              fullWidth
              id="editDescription"
              label="描述（可选）"
              value={editDescription}
              onChange={handleEditDescriptionChange}
              multiline
              rows={3}
            />
            
            <Box sx={{ mt: 3 }}>
              <FormControl component="fieldset">
                <FormLabel component="legend">投票类型</FormLabel>
                <RadioGroup
                  row
                  value={editPollType.toString()}
                  onChange={handleEditPollTypeChange}
                >
                  <FormControlLabel value="0" control={<Radio />} label="单选题" />
                  <FormControlLabel value="1" control={<Radio />} label="多选题" />
                </RadioGroup>
              </FormControl>
            </Box>
            
            <Box sx={{ mt: 3 }}>
              <Typography variant="subtitle1" gutterBottom>
                选项
              </Typography>
              <List>
                {editOptions.map((option, index) => (
                  <ListItem
                    key={index}
                    sx={{
                      display: 'flex',
                      alignItems: 'center',
                      pl: 0,
                      pr: 0
                    }}
                  >
                    <TextField
                      fullWidth
                      label={`选项 ${index + 1}`}
                      value={option.text}
                      onChange={(e) => handleEditOptionChange(index, e.target.value)}
                      error={!!editValidationErrors.options?.[index]}
                      helperText={editValidationErrors.options?.[index]}
                      sx={{ mr: 1 }}
                    />
                    <IconButton
                      onClick={() => handleRemoveEditOption(index)}
                      disabled={editOptions.length <= 2}
                    >
                      <DeleteIcon />
                    </IconButton>
                  </ListItem>
                ))}
                <ListItem sx={{ pl: 0 }}>
                  <Button
                    startIcon={<AddIcon />}
                    onClick={handleAddEditOption}
                    sx={{ mt: 1 }}
                  >
                    添加选项
                  </Button>
                </ListItem>
              </List>
            </Box>
            
            <Divider sx={{ my: 3 }} />
            
            <Box sx={{ mt: 3 }}>
              <FormControlLabel
                control={
                  <Switch
                    checked={editActive}
                    onChange={handleEditActiveChange}
                    name="editActive"
                    color="primary"
                  />
                }
                label="启用投票"
              />
              <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5, ml: 6.5 }}>
                启用后，用户可以立即开始投票
              </Typography>
            </Box>
          </LocalizationProvider>
        </DialogContent>
        
        <DialogActions>
          <Button onClick={handleCloseEditDialog}>取消</Button>
          <Button
            variant="contained"
            color="primary"
            onClick={handleSaveEdit}
            disabled={submitLoading}
            startIcon={submitLoading ? <CircularProgress size={24} /> : <SaveIcon />}
          >
            保存修改
          </Button>
        </DialogActions>
      </Dialog>

      {/* 模拟投票对话框 */}
      <Dialog open={showSimulationDialog} onClose={closeSimulationDialog} maxWidth="sm" fullWidth>
        <DialogTitle>
          <Box sx={{ display: 'flex', alignItems: 'center' }}>
            <SimulationIcon sx={{ mr: 1 }} />
            模拟投票设置
          </Box>
        </DialogTitle>
        
        <DialogContent dividers>
          <DialogContentText sx={{ mb: 3 }}>
            选择一种模拟投票的方式：界面模拟仅在前端显示效果，真实请求会发送实际的请求到服务器。
          </DialogContentText>
          
          <FormControl component="fieldset" sx={{ mb: 3, width: '100%' }}>
            <FormLabel component="legend">模拟类型</FormLabel>
            <RadioGroup
              row
              value={simulationType}
              onChange={handleSimulationTypeChange}
            >
              <FormControlLabel 
                value="virtual" 
                control={<Radio />} 
                label="界面模拟" 
              />
              <FormControlLabel 
                value="highConcurrency" 
                control={<Radio />} 
                label="真实请求" 
              />
            </RadioGroup>
            <FormHelperText>
              {simulationType === 'virtual' 
                ? '界面模拟：在前端临时显示结果变化，不会影响实际数据' 
                : '真实请求：向后端发送实际投票请求，可以选择是否保存到数据库'}
            </FormHelperText>
          </FormControl>
          
          {simulationType === 'highConcurrency' && (
            <Box sx={{ mb: 3 }}>
              <FormControlLabel
                control={
                  <Switch
                    checked={saveSimulationResults}
                    onChange={handleSaveResultsChange}
                    color="primary"
                  />
                }
                label="保存投票结果到数据库"
              />
              <FormHelperText>
                {saveSimulationResults 
                  ? '启用后，模拟投票将真实保存到数据库中' 
                  : '禁用后，仅发送请求测试性能，不会保存结果'}
              </FormHelperText>
            </Box>
          )}
          
          <Box sx={{ mb: 3 }}>
            <Typography gutterBottom>每次模拟投票数量: {votesToAdd}</Typography>
            <Slider
              value={votesToAdd}
              onChange={(_event: Event, value: number | number[]) => setVotesToAdd(value as number)}
              min={1}
              max={50}
              step={1}
              marks={[
                { value: 1, label: '1' },
                { value: 10, label: '10' },
                { value: 25, label: '25' },
                { value: 50, label: '50' }
              ]}
            />
          </Box>
          
          <Box sx={{ mb: 3 }}>
            <Typography gutterBottom>模拟间隔时间: {votingInterval}ms</Typography>
            <Slider
              value={votingInterval}
              onChange={(_event: Event, value: number | number[]) => setVotingInterval(value as number)}
              min={500}
              max={5000}
              step={500}
              marks={[
                { value: 500, label: '0.5秒' },
                { value: 1000, label: '1秒' },
                { value: 3000, label: '3秒' },
                { value: 5000, label: '5秒' }
              ]}
            />
          </Box>
        </DialogContent>
        
        <DialogActions>
          <Button onClick={closeSimulationDialog}>取消</Button>
          <Button
            variant="contained"
            color="primary"
            onClick={startSimulation}
          >
            开始模拟
          </Button>
        </DialogActions>
      </Dialog>

      {apiDebugInfo && (
        <RequestDebugInfo apiDebugInfo={apiDebugInfo} />
      )}
    </>
  );
};

export default PollDetail; 