import React, { useState, useEffect } from 'react';
import { 
  Container, 
  Typography, 
  Button, 
  Box, 
  Paper, 
  TextField, 
  FormControl, 
  FormLabel, 
  RadioGroup, 
  FormControlLabel, 
  Radio,
  Divider,
  CircularProgress,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Slider,
  Alert,
  useTheme,
  alpha,
  LinearProgress,
  Card,
  CardContent,
  Grid,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Switch,
  MenuItem,
  Select,
  InputLabel,
  IconButton,
  Tooltip
} from '@mui/material';
import EditIcon from '@mui/icons-material/Edit';
import SaveIcon from '@mui/icons-material/Save';
import DeleteIcon from '@mui/icons-material/Delete';
import PollIcon from '@mui/icons-material/Poll';
import axios from 'axios';

// 性能测试结果类型
interface TestResult {
  id: string;
  timestamp: string;
  concurrentUsers: number;
  requestsPerUser: number;
  totalRequests: number;
  successfulRequests: number;
  failedRequests: number;
  avgResponseTime: number;
  minResponseTime: number;
  maxResponseTime: number;
  p95ResponseTime: number;
  p99ResponseTime: number;
  throughput: number; // 每秒请求数
  errorRate: number;
  concurrencyControl: string;
  testDuration: number; // 秒
}

interface TestParams {
  pollId: number;
  concurrentUsers: number;
  requestsPerUser: number;
  rampUpTime: number; // 秒
  testDuration: number; // 秒
  concurrencyControl: string;
}

interface PollOption {
  id: number;
  text: string;
  votes: number;
}

interface Poll {
  id: number;
  question: string;
  options: PollOption[];
  type: 'single' | 'multiple';
}

const defaultParams: TestParams = {
  pollId: 1,
  concurrentUsers: 10,
  requestsPerUser: 5,
  rampUpTime: 5,
  testDuration: 30,
  concurrencyControl: 'redis_lock'
};

// 模拟的测试结果数据
const mockResults: TestResult[] = [
  {
    id: '1',
    timestamp: new Date().toISOString(),
    concurrentUsers: 10,
    requestsPerUser: 5,
    totalRequests: 50,
    successfulRequests: 48,
    failedRequests: 2,
    avgResponseTime: 120,
    minResponseTime: 45,
    maxResponseTime: 350,
    p95ResponseTime: 310,
    p99ResponseTime: 345,
    throughput: 25.5,
    errorRate: 4.0,
    concurrencyControl: 'redis_lock',
    testDuration: 30
  },
  {
    id: '2',
    timestamp: new Date(Date.now() - 3600000).toISOString(),
    concurrentUsers: 20,
    requestsPerUser: 5,
    totalRequests: 100,
    successfulRequests: 95,
    failedRequests: 5,
    avgResponseTime: 185,
    minResponseTime: 55,
    maxResponseTime: 420,
    p95ResponseTime: 390,
    p99ResponseTime: 415,
    throughput: 21.2,
    errorRate: 5.0,
    concurrencyControl: 'database_lock',
    testDuration: 30
  },
  {
    id: '3',
    timestamp: new Date(Date.now() - 7200000).toISOString(),
    concurrentUsers: 30,
    requestsPerUser: 5,
    totalRequests: 150,
    successfulRequests: 142,
    failedRequests: 8,
    avgResponseTime: 230,
    minResponseTime: 65,
    maxResponseTime: 530,
    p95ResponseTime: 480,
    p99ResponseTime: 520,
    throughput: 18.7,
    errorRate: 5.3,
    concurrencyControl: 'optimistic_lock',
    testDuration: 30
  }
];

// 响应时间实时数据类型
interface ResponseTimeData {
  time: number; // 秒
  avg: number;
  p95: number;
}

// 吞吐量实时数据类型
interface ThroughputData {
  time: number; // 秒
  throughput: number;
}

// 示例投票数据
const samplePolls: Poll[] = [
  {
    id: 1,
    question: "您最喜欢的编程语言是？",
    options: [
      { id: 1, text: "Java", votes: 324 },
      { id: 2, text: "Python", votes: 526 },
      { id: 3, text: "JavaScript", votes: 429 },
      { id: 4, text: "Go", votes: 298 },
      { id: 5, text: "C++", votes: 215 }
    ],
    type: "single"
  },
  {
    id: 2,
    question: "您使用过哪些前端框架？",
    options: [
      { id: 1, text: "React", votes: 857 },
      { id: 2, text: "Vue", votes: 642 },
      { id: 3, text: "Angular", votes: 423 },
      { id: 4, text: "Svelte", votes: 295 }
    ],
    type: "multiple"
  }
];

const PerformanceTest: React.FC = () => {
  const theme = useTheme();
  const [testParams, setTestParams] = useState<TestParams>(defaultParams);
  const [testing, setTesting] = useState(false);
  const [testResults, setTestResults] = useState<TestResult[]>(() => {
    // 从localStorage读取历史测试结果
    const savedResults = localStorage.getItem('performanceTestResults');
    return savedResults ? JSON.parse(savedResults) : mockResults;
  });
  const [selectedResult, setSelectedResult] = useState<TestResult | null>(null);
  const [responseTimeData, setResponseTimeData] = useState<ResponseTimeData[]>(() => {
    // 从localStorage读取上次的响应时间数据
    const savedData = localStorage.getItem('responseTimeData');
    return savedData ? JSON.parse(savedData) : [];
  });
  const [throughputData, setThroughputData] = useState<ThroughputData[]>(() => {
    // 从localStorage读取上次的吞吐量数据
    const savedData = localStorage.getItem('throughputData');
    return savedData ? JSON.parse(savedData) : [];
  });
  const [remainingTime, setRemainingTime] = useState(0);
  const [error, setError] = useState<string | null>(null);
  const [currentAvgResponse, setCurrentAvgResponse] = useState<number>(() => {
    // 从localStorage读取上次的平均响应时间
    const lastData = localStorage.getItem('lastPerformanceMetrics');
    return lastData ? JSON.parse(lastData).avgResponse : 0;
  });
  const [currentP95Response, setCurrentP95Response] = useState<number>(() => {
    // 从localStorage读取上次的P95响应时间
    const lastData = localStorage.getItem('lastPerformanceMetrics');
    return lastData ? JSON.parse(lastData).p95Response : 0;
  });
  const [currentThroughput, setCurrentThroughput] = useState<number>(() => {
    // 从localStorage读取上次的吞吐量
    const lastData = localStorage.getItem('lastPerformanceMetrics');
    return lastData ? JSON.parse(lastData).throughput : 0;
  });
  const [polls, setPolls] = useState<Poll[]>(() => {
    const savedPolls = localStorage.getItem('polls');
    return savedPolls ? JSON.parse(savedPolls) : samplePolls;
  });
  const [selectedPoll, setSelectedPoll] = useState<Poll | null>(null);
  const [showSimulateDialog, setShowSimulateDialog] = useState(false);
  const [showEditPollDialog, setShowEditPollDialog] = useState(false);
  const [votesToAdd, setVotesToAdd] = useState(10);
  const [saveVotes, setSaveVotes] = useState(true);
  const [editingPoll, setEditingPoll] = useState<Poll | null>(null);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  
  // 每当测试结果更新时保存到localStorage
  useEffect(() => {
    localStorage.setItem('performanceTestResults', JSON.stringify(testResults));
  }, [testResults]);
  
  // 保存实时数据到localStorage
  useEffect(() => {
    if (responseTimeData.length > 0) {
      localStorage.setItem('responseTimeData', JSON.stringify(responseTimeData));
    }
  }, [responseTimeData]);
  
  useEffect(() => {
    if (throughputData.length > 0) {
      localStorage.setItem('throughputData', JSON.stringify(throughputData));
    }
  }, [throughputData]);
  
  // 保存当前显示的性能指标
  useEffect(() => {
    if (currentAvgResponse || currentP95Response || currentThroughput) {
      localStorage.setItem('lastPerformanceMetrics', JSON.stringify({
        avgResponse: currentAvgResponse,
        p95Response: currentP95Response,
        throughput: currentThroughput
      }));
    }
  }, [currentAvgResponse, currentP95Response, currentThroughput]);
  
  // 保存投票数据到localStorage
  useEffect(() => {
    localStorage.setItem('polls', JSON.stringify(polls));
  }, [polls]);

  // 处理参数更改
  const handleParamChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const { name, value } = e.target;
    setTestParams(prev => ({
      ...prev,
      [name]: name === 'concurrencyControl' ? value : Number(value)
    }));
  };
  
  // 启动测试
  const startTest = async () => {
    try {
      setTesting(true);
      setError(null);
      // 不清除之前的数据，只添加新数据
      // setResponseTimeData([]);
      // setThroughputData([]);
      setRemainingTime(testParams.testDuration);
      
      // 实际应用中，这里会发送一个请求到后端启动测试
      console.log('启动性能测试，参数:', testParams);
      
      // 模拟启动测试的 API 调用
      // const response = await axios.post('/api/performance-test', testParams);
      
      // 模拟每秒接收实时测试数据
      let elapsedTime = 0;
      const dataInterval = setInterval(() => {
        elapsedTime += 1;
        setRemainingTime(prev => prev - 1);
        
        // 生成模拟的响应时间数据
        const avgResponse = 100 + Math.random() * 50 * (1 + elapsedTime/30);
        const p95Response = 200 + Math.random() * 100 * (1 + elapsedTime/20);
        const newThroughput = 20 + Math.random() * 10 - (elapsedTime > 15 ? (elapsedTime-15) * 0.5 : 0);
        
        setCurrentAvgResponse(avgResponse);
        setCurrentP95Response(p95Response);
        setCurrentThroughput(newThroughput);
        
        const newResponseTime = {
          time: responseTimeData.length + elapsedTime,
          avg: avgResponse,
          p95: p95Response
        };
        setResponseTimeData(prev => [...prev, newResponseTime]);
        
        // 生成模拟的吞吐量数据
        const newThroughputData = {
          time: throughputData.length + elapsedTime,
          throughput: newThroughput
        };
        setThroughputData(prev => [...prev, newThroughputData]);
        
        // 测试完成
        if (elapsedTime >= testParams.testDuration) {
          clearInterval(dataInterval);
          
          // 生成测试结果
          const newResult: TestResult = {
            id: Date.now().toString(),
            timestamp: new Date().toISOString(),
            concurrentUsers: testParams.concurrentUsers,
            requestsPerUser: testParams.requestsPerUser,
            totalRequests: testParams.concurrentUsers * testParams.requestsPerUser,
            successfulRequests: Math.floor(testParams.concurrentUsers * testParams.requestsPerUser * 0.95),
            failedRequests: Math.floor(testParams.concurrentUsers * testParams.requestsPerUser * 0.05),
            avgResponseTime: 120 + Math.random() * 50 + testParams.concurrentUsers * 2,
            minResponseTime: 45 + Math.random() * 20,
            maxResponseTime: 350 + Math.random() * 100 + testParams.concurrentUsers * 5,
            p95ResponseTime: 310 + Math.random() * 80 + testParams.concurrentUsers * 3,
            p99ResponseTime: 345 + Math.random() * 90 + testParams.concurrentUsers * 4,
            throughput: 25 - testParams.concurrentUsers * 0.2,
            errorRate: Math.random() * 3 + 2 + testParams.concurrentUsers * 0.1,
            concurrencyControl: testParams.concurrencyControl,
            testDuration: testParams.testDuration
          };
          
          setTestResults(prev => [newResult, ...prev]);
          setSelectedResult(newResult);
          setTesting(false);
          
          // 保存测试结果到localStorage
          const updatedResults = [newResult, ...testResults];
          localStorage.setItem('performanceTestResults', JSON.stringify(updatedResults));
        }
      }, 1000);
      
      return () => clearInterval(dataInterval);
      
    } catch (err: any) {
      console.error('测试启动失败:', err);
      setError(err.message || '测试启动失败，请稍后再试');
      setTesting(false);
    }
  };
  
  // 清除所有测试数据
  const clearAllData = () => {
    if (window.confirm('确定要清除所有测试数据吗？这将删除所有历史记录和当前数据。')) {
      setResponseTimeData([]);
      setThroughputData([]);
      setTestResults(mockResults);
      setSelectedResult(null);
      setCurrentAvgResponse(0);
      setCurrentP95Response(0);
      setCurrentThroughput(0);
      
      // 清除localStorage中的数据
      localStorage.removeItem('responseTimeData');
      localStorage.removeItem('throughputData');
      localStorage.removeItem('performanceTestResults');
      localStorage.removeItem('lastPerformanceMetrics');
    }
  };
  
  // 显示/查看测试结果
  const viewTestResult = (result: TestResult) => {
    setSelectedResult(result);
  };
  
  // 格式化时间
  const formatTime = (timeInSeconds: number) => {
    const minutes = Math.floor(timeInSeconds / 60);
    const seconds = timeInSeconds % 60;
    return `${minutes}:${seconds.toString().padStart(2, '0')}`;
  };

  // 根据值返回颜色
  const getColorForValue = (value: number, max: number) => {
    const ratio = value / max;
    if (ratio < 0.5) return theme.palette.success.main;
    if (ratio < 0.75) return theme.palette.warning.main;
    return theme.palette.error.main;
  };

  // 获取百分比值
  const getPercentage = (value: number, max: number) => {
    return Math.min(Math.round((value / max) * 100), 100);
  };
  
  // 打开模拟投票对话框
  const openSimulateDialog = () => {
    if (selectedResult) {
      // 查找与测试相关的投票
      const poll = polls.find(p => p.id === testParams.pollId);
      if (poll) {
        setSelectedPoll(poll);
        setShowSimulateDialog(true);
      } else {
        setError(`未找到ID为${testParams.pollId}的投票`);
      }
    } else {
      setError('请先选择一个测试结果');
    }
  };

  // 模拟投票
  const simulateVoting = () => {
    if (!selectedPoll) return;
    
    const updatedPoll = { ...selectedPoll };
    // 随机分配投票
    updatedPoll.options = updatedPoll.options.map(option => {
      // 随机决定这个选项获得多少票
      const newVotes = Math.floor(Math.random() * votesToAdd);
      return {
        ...option,
        votes: option.votes + newVotes
      };
    });
    
    // 若选择保存，则更新polls状态
    if (saveVotes) {
      const updatedPolls = polls.map(p => 
        p.id === updatedPoll.id ? updatedPoll : p
      );
      setPolls(updatedPolls);
      localStorage.setItem('polls', JSON.stringify(updatedPolls));
    }
    
    // 展示更新后的结果
    setSelectedPoll(updatedPoll);
  };
  
  // 打开编辑投票对话框
  const openEditPollDialog = () => {
    if (selectedResult) {
      const poll = polls.find(p => p.id === testParams.pollId);
      if (poll) {
        setEditingPoll(JSON.parse(JSON.stringify(poll))); // 深拷贝
        setShowEditPollDialog(true);
      } else {
        setError(`未找到ID为${testParams.pollId}的投票`);
      }
    } else {
      setError('请先选择一个测试结果');
    }
  };
  
  // 更新编辑中的投票
  const handleEditPollChange = (field: string, value: any) => {
    if (!editingPoll) return;
    setEditingPoll({
      ...editingPoll,
      [field]: value
    });
  };
  
  // 更新编辑中的选项
  const handleEditOptionChange = (optionId: number, field: string, value: any) => {
    if (!editingPoll) return;
    const updatedOptions = editingPoll.options.map(opt => 
      opt.id === optionId ? { ...opt, [field]: value } : opt
    );
    setEditingPoll({
      ...editingPoll,
      options: updatedOptions
    });
  };
  
  // 添加新选项
  const addNewOption = () => {
    if (!editingPoll) return;
    const newId = Math.max(...editingPoll.options.map(o => o.id)) + 1;
    setEditingPoll({
      ...editingPoll,
      options: [
        ...editingPoll.options,
        { id: newId, text: `选项${newId}`, votes: 0 }
      ]
    });
  };
  
  // 删除选项
  const removeOption = (optionId: number) => {
    if (!editingPoll || editingPoll.options.length <= 2) return; // 至少保留两个选项
    setEditingPoll({
      ...editingPoll,
      options: editingPoll.options.filter(opt => opt.id !== optionId)
    });
  };
  
  // 保存编辑后的投票
  const saveEditedPoll = () => {
    if (!editingPoll) return;
    
    const updatedPolls = polls.map(p => 
      p.id === editingPoll.id ? editingPoll : p
    );
    setPolls(updatedPolls);
    localStorage.setItem('polls', JSON.stringify(updatedPolls));
    setShowEditPollDialog(false);
    
    // 如果当前选中的投票被编辑了，也要更新它
    if (selectedPoll && selectedPoll.id === editingPoll.id) {
      setSelectedPoll(editingPoll);
    }
  };
  
  // 删除投票
  const deletePoll = () => {
    if (!editingPoll) return;
    
    const updatedPolls = polls.filter(p => p.id !== editingPoll.id);
    setPolls(updatedPolls);
    localStorage.setItem('polls', JSON.stringify(updatedPolls));
    setShowEditPollDialog(false);
    setShowDeleteConfirm(false);
    
    if (selectedPoll && selectedPoll.id === editingPoll.id) {
      setSelectedPoll(null);
    }
  };
  
  // 格式化投票结果为百分比
  const formatPercentage = (votes: number, total: number) => {
    if (total === 0) return "0%";
    return `${((votes / total) * 100).toFixed(1)}%`;
  };

  return (
    <Container maxWidth="lg" sx={{ mt: 4, mb: 8 }}>
      <Paper 
        elevation={0} 
        sx={{ 
          p: 4, 
          borderRadius: '24px',
          border: '1px solid',
          borderColor: 'divider',
          mb: 4,
          background: 'linear-gradient(145deg, rgba(255,255,255,0.8) 0%, rgba(249,250,251,0.9) 100%)',
        }}
      >
        <Typography variant="h4" gutterBottom>性能测试面板</Typography>
        <Typography variant="body1" color="text.secondary" paragraph>
          此工具用于测试投票系统在高并发场景下的性能表现，可以评估不同并发控制机制的效果。
        </Typography>
        
        <Box sx={{ mt: 4 }}>
          <Typography variant="h6" gutterBottom>测试参数</Typography>
          
          <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 4, mt: 2 }}>
            <TextField 
              label="投票ID" 
              type="number"
              name="pollId"
              value={testParams.pollId}
              onChange={handleParamChange}
              sx={{ width: 150 }}
              disabled={testing}
            />
            
            <Box sx={{ width: 250 }}>
              <Typography gutterBottom>并发用户数: {testParams.concurrentUsers}</Typography>
              <Slider
                name="concurrentUsers"
                value={testParams.concurrentUsers}
                onChange={(_, value) => setTestParams(prev => ({ ...prev, concurrentUsers: value as number }))}
                min={1}
                max={100}
                disabled={testing}
              />
            </Box>
            
            <Box sx={{ width: 250 }}>
              <Typography gutterBottom>每用户请求数: {testParams.requestsPerUser}</Typography>
              <Slider
                name="requestsPerUser"
                value={testParams.requestsPerUser}
                onChange={(_, value) => setTestParams(prev => ({ ...prev, requestsPerUser: value as number }))}
                min={1}
                max={50}
                disabled={testing}
              />
            </Box>
            
            <Box sx={{ width: 250 }}>
              <Typography gutterBottom>测试时长(秒): {testParams.testDuration}</Typography>
              <Slider
                name="testDuration"
                value={testParams.testDuration}
                onChange={(_, value) => setTestParams(prev => ({ ...prev, testDuration: value as number }))}
                min={10}
                max={120}
                step={5}
                disabled={testing}
              />
            </Box>
          </Box>
          
          <FormControl component="fieldset" sx={{ mt: 3 }}>
            <FormLabel component="legend">并发控制机制</FormLabel>
            <RadioGroup 
              name="concurrencyControl" 
              value={testParams.concurrencyControl} 
              onChange={handleParamChange}
              row
            >
              <FormControlLabel 
                value="redis_lock" 
                control={<Radio />} 
                label="Redis分布式锁" 
                disabled={testing}
              />
              <FormControlLabel 
                value="database_lock" 
                control={<Radio />} 
                label="数据库锁" 
                disabled={testing}
              />
              <FormControlLabel 
                value="optimistic_lock" 
                control={<Radio />} 
                label="乐观锁" 
                disabled={testing}
              />
              <FormControlLabel 
                value="bloom_filter" 
                control={<Radio />} 
                label="布隆过滤器" 
                disabled={testing}
              />
            </RadioGroup>
          </FormControl>
          
          <Box sx={{ mt: 3, display: 'flex', gap: 2 }}>
            <Button 
              variant="contained" 
              color="primary" 
              size="large"
              onClick={startTest}
              disabled={testing}
              sx={{ 
                borderRadius: '20px',
                px: a => a.spacing(4),
                py: 1.2
              }}
            >
              {testing ? '测试中...' : '启动测试'}
            </Button>
            
            <Button 
              variant="outlined"
              color="error"
              size="large"
              onClick={clearAllData}
              disabled={testing}
              sx={{ 
                borderRadius: '20px',
                px: a => a.spacing(4),
                py: 1.2
              }}
            >
              清除所有数据
            </Button>
            
            {testing && (
              <Box sx={{ display: 'flex', alignItems: 'center', ml: 2 }}>
                <CircularProgress size={24} sx={{ mr: 2 }} />
                <Typography>
                  测试进行中，剩余时间: {formatTime(remainingTime)}
                </Typography>
              </Box>
            )}
            
            {error && (
              <Alert severity="error" sx={{ mt: 2 }}>{error}</Alert>
            )}
          </Box>
        </Box>
      </Paper>
      
      <Paper 
        elevation={0} 
        sx={{ 
          p: 4, 
          borderRadius: '24px', 
          border: '1px solid',
          borderColor: 'divider',
          mb: 4,
          background: 'linear-gradient(145deg, rgba(255,255,255,0.8) 0%, rgba(249,250,251,0.9) 100%)',
        }}
      >
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
          <Typography variant="h6" gutterBottom>测试与投票操作</Typography>
          <Box sx={{ display: 'flex', gap: 2 }}>
            <Button
              variant="contained"
              color="secondary"
              startIcon={<PollIcon />}
              onClick={openSimulateDialog}
              disabled={!selectedResult}
            >
              模拟投票
            </Button>
            <Button
              variant="outlined"
              color="primary"
              startIcon={<EditIcon />}
              onClick={openEditPollDialog}
              disabled={!selectedResult}
            >
              编辑投票
            </Button>
          </Box>
        </Box>
        
        <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
          在此您可以模拟投票活动或编辑投票内容，也可以查看性能测试结果。
        </Typography>
      </Paper>
      
      {/* 实时测试数据面板 - 即使不在测试中也显示 */}
      <Paper 
        elevation={0} 
        sx={{ 
          p: 4, 
          borderRadius: '24px', 
          border: '1px solid',
          borderColor: 'divider',
          mb: 4,
          background: 'linear-gradient(145deg, rgba(255,255,255,0.8) 0%, rgba(249,250,251,0.9) 100%)',
        }}
      >
        <Typography variant="h6" gutterBottom>实时测试数据</Typography>
        
        <Grid container spacing={4} sx={{ mt: 1 }}>
          <Grid item xs={12} md={4}>
            <Card elevation={0} sx={{ bgcolor: 'background.paper', border: '1px solid', borderColor: 'divider' }}>
              <CardContent>
                <Typography color="text.secondary" gutterBottom>平均响应时间</Typography>
                <Typography variant="h3" color={getColorForValue(currentAvgResponse, 500)} sx={{ fontWeight: 'bold', mb: 1 }}>
                  {currentAvgResponse.toFixed(1)} ms
                </Typography>
                <LinearProgress 
                  variant="determinate" 
                  value={getPercentage(currentAvgResponse, 500)} 
                  color={currentAvgResponse > 300 ? 'error' : currentAvgResponse > 200 ? 'warning' : 'success'} 
                  sx={{ height: 8, borderRadius: 4 }}
                />
              </CardContent>
            </Card>
          </Grid>
          
          <Grid item xs={12} md={4}>
            <Card elevation={0} sx={{ bgcolor: 'background.paper', border: '1px solid', borderColor: 'divider' }}>
              <CardContent>
                <Typography color="text.secondary" gutterBottom>95% 响应时间</Typography>
                <Typography variant="h3" color={getColorForValue(currentP95Response, 800)} sx={{ fontWeight: 'bold', mb: 1 }}>
                  {currentP95Response.toFixed(1)} ms
                </Typography>
                <LinearProgress 
                  variant="determinate" 
                  value={getPercentage(currentP95Response, 800)} 
                  color={currentP95Response > 500 ? 'error' : currentP95Response > 300 ? 'warning' : 'success'} 
                  sx={{ height: 8, borderRadius: 4 }}
                />
              </CardContent>
            </Card>
          </Grid>
          
          <Grid item xs={12} md={4}>
            <Card elevation={0} sx={{ bgcolor: 'background.paper', border: '1px solid', borderColor: 'divider' }}>
              <CardContent>
                <Typography color="text.secondary" gutterBottom>吞吐量</Typography>
                <Typography variant="h3" color={getColorForValue(currentThroughput, 50)} sx={{ fontWeight: 'bold', mb: 1 }}>
                  {currentThroughput.toFixed(1)} 请求/秒
                </Typography>
                <LinearProgress 
                  variant="determinate" 
                  value={getPercentage(currentThroughput, 50)} 
                  color="primary" 
                  sx={{ height: 8, borderRadius: 4 }}
                />
              </CardContent>
            </Card>
          </Grid>
        </Grid>
        
        {responseTimeData.length > 0 && (
          <Box sx={{ mt: 4 }}>
            <Typography variant="subtitle1" gutterBottom>实时数据表格</Typography>
            <TableContainer sx={{ maxHeight: 250 }}>
              <Table stickyHeader size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>时间(秒)</TableCell>
                    <TableCell>平均响应时间(ms)</TableCell>
                    <TableCell>P95响应时间(ms)</TableCell>
                    <TableCell>吞吐量(请求/秒)</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {responseTimeData.slice(-10).map((data, index) => {
                    const throughput = throughputData[responseTimeData.indexOf(data)]?.throughput || 0;
                    
                    return (
                      <TableRow key={data.time} hover>
                        <TableCell>{data.time}</TableCell>
                        <TableCell>{data.avg.toFixed(1)}</TableCell>
                        <TableCell>{data.p95.toFixed(1)}</TableCell>
                        <TableCell>{throughput.toFixed(1)}</TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </TableContainer>
          </Box>
        )}
      </Paper>
      
      <Box sx={{ display: 'flex', gap: 4, flexWrap: { xs: 'wrap', md: 'nowrap' } }}>
        <Paper 
          elevation={0} 
          sx={{ 
            p: 4, 
            borderRadius: '24px', 
            border: '1px solid',
            borderColor: 'divider',
            flex: { xs: '1 1 100%', md: '1 1 40%' },
            background: 'linear-gradient(145deg, rgba(255,255,255,0.8) 0%, rgba(249,250,251,0.9) 100%)',
          }}
        >
          <Typography variant="h6" gutterBottom>历史测试记录</Typography>
          
          <TableContainer sx={{ mt: 2, maxHeight: 400 }}>
            <Table stickyHeader size="small">
              <TableHead>
                <TableRow>
                  <TableCell>测试时间</TableCell>
                  <TableCell>并发用户</TableCell>
                  <TableCell>控制机制</TableCell>
                  <TableCell>成功率</TableCell>
                  <TableCell>操作</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {testResults.map(result => (
                  <TableRow 
                    key={result.id} 
                    sx={{ 
                      cursor: 'pointer',
                      bgcolor: selectedResult?.id === result.id ? alpha(theme.palette.primary.main, 0.1) : 'inherit',
                      '&:hover': { bgcolor: alpha(theme.palette.primary.main, 0.05) }
                    }}
                    onClick={() => viewTestResult(result)}
                  >
                    <TableCell>
                      {new Date(result.timestamp).toLocaleString()}
                    </TableCell>
                    <TableCell>{result.concurrentUsers}</TableCell>
                    <TableCell>
                      {result.concurrencyControl === 'redis_lock' && 'Redis分布式锁'}
                      {result.concurrencyControl === 'database_lock' && '数据库锁'}
                      {result.concurrencyControl === 'optimistic_lock' && '乐观锁'}
                      {result.concurrencyControl === 'bloom_filter' && '布隆过滤器'}
                    </TableCell>
                    <TableCell>
                      {((result.successfulRequests / result.totalRequests) * 100).toFixed(1)}%
                    </TableCell>
                    <TableCell>
                      <Button 
                        size="small" 
                        variant="outlined"
                        onClick={(e) => {
                          e.stopPropagation();
                          viewTestResult(result);
                        }}
                      >
                        查看
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </Paper>
        
        <Paper 
          elevation={0} 
          sx={{ 
            p: 4, 
            borderRadius: '24px', 
            border: '1px solid',
            borderColor: 'divider',
            flex: { xs: '1 1 100%', md: '1 1 60%' },
            background: 'linear-gradient(145deg, rgba(255,255,255,0.8) 0%, rgba(249,250,251,0.9) 100%)',
          }}
        >
          <Typography variant="h6" gutterBottom>测试结果详情</Typography>
          
          {selectedResult ? (
            <Box sx={{ mt: 2 }}>
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 2, mb: 3 }}>
                <Box sx={{ p: 2, bgcolor: alpha(theme.palette.info.main, 0.1), borderRadius: 2, minWidth: 150 }}>
                  <Typography variant="body2" color="text.secondary" gutterBottom>并发用户数</Typography>
                  <Typography variant="h6">{selectedResult.concurrentUsers}</Typography>
                </Box>
                
                <Box sx={{ p: 2, bgcolor: alpha(theme.palette.primary.main, 0.1), borderRadius: 2, minWidth: 150 }}>
                  <Typography variant="body2" color="text.secondary" gutterBottom>总请求数</Typography>
                  <Typography variant="h6">{selectedResult.totalRequests}</Typography>
                </Box>
                
                <Box sx={{ p: 2, bgcolor: alpha(theme.palette.success.main, 0.1), borderRadius: 2, minWidth: 150 }}>
                  <Typography variant="body2" color="text.secondary" gutterBottom>成功请求</Typography>
                  <Typography variant="h6">{selectedResult.successfulRequests}</Typography>
                </Box>
                
                <Box sx={{ p: 2, bgcolor: alpha(theme.palette.error.main, 0.1), borderRadius: 2, minWidth: 150 }}>
                  <Typography variant="body2" color="text.secondary" gutterBottom>失败请求</Typography>
                  <Typography variant="h6">{selectedResult.failedRequests}</Typography>
                </Box>
              </Box>
              
              <Divider sx={{ my: 3 }} />
              
              <Typography variant="subtitle1" gutterBottom>响应时间统计(毫秒)</Typography>
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 2, mb: 3 }}>
                <Box sx={{ p: 2, bgcolor: alpha(theme.palette.primary.main, 0.1), borderRadius: 2, minWidth: 120 }}>
                  <Typography variant="body2" color="text.secondary" gutterBottom>平均</Typography>
                  <Typography variant="h6">{selectedResult.avgResponseTime.toFixed(2)}</Typography>
                </Box>
                
                <Box sx={{ p: 2, bgcolor: alpha(theme.palette.primary.main, 0.05), borderRadius: 2, minWidth: 120 }}>
                  <Typography variant="body2" color="text.secondary" gutterBottom>最小</Typography>
                  <Typography variant="h6">{selectedResult.minResponseTime.toFixed(2)}</Typography>
                </Box>
                
                <Box sx={{ p: 2, bgcolor: alpha(theme.palette.warning.main, 0.1), borderRadius: 2, minWidth: 120 }}>
                  <Typography variant="body2" color="text.secondary" gutterBottom>P95</Typography>
                  <Typography variant="h6">{selectedResult.p95ResponseTime.toFixed(2)}</Typography>
                </Box>
                
                <Box sx={{ p: 2, bgcolor: alpha(theme.palette.warning.main, 0.2), borderRadius: 2, minWidth: 120 }}>
                  <Typography variant="body2" color="text.secondary" gutterBottom>P99</Typography>
                  <Typography variant="h6">{selectedResult.p99ResponseTime.toFixed(2)}</Typography>
                </Box>
                
                <Box sx={{ p: 2, bgcolor: alpha(theme.palette.error.main, 0.1), borderRadius: 2, minWidth: 120 }}>
                  <Typography variant="body2" color="text.secondary" gutterBottom>最大</Typography>
                  <Typography variant="h6">{selectedResult.maxResponseTime.toFixed(2)}</Typography>
                </Box>
              </Box>
              
              <Divider sx={{ my: 3 }} />
              
              <Typography variant="subtitle1" gutterBottom>并发控制详情</Typography>
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 2, mb: 3 }}>
                <Box sx={{ p: k => k.spacing(2), bgcolor: alpha(theme.palette.info.main, 0.1), borderRadius: 2, minWidth: 150 }}>
                  <Typography variant="body2" color="text.secondary" gutterBottom>控制机制</Typography>
                  <Typography variant="h6">
                    {selectedResult.concurrencyControl === 'redis_lock' && 'Redis分布式锁'}
                    {selectedResult.concurrencyControl === 'database_lock' && '数据库锁'}
                    {selectedResult.concurrencyControl === 'optimistic_lock' && '乐观锁'}
                    {selectedResult.concurrencyControl === 'bloom_filter' && '布隆过滤器'}
                  </Typography>
                </Box>
                
                <Box sx={{ p: k => k.spacing(2), bgcolor: alpha(theme.palette.success.main, 0.1), borderRadius: 2, minWidth: 150 }}>
                  <Typography variant="body2" color="text.secondary" gutterBottom>吞吐量(请求/秒)</Typography>
                  <Typography variant="h6">{selectedResult.throughput.toFixed(2)}</Typography>
                </Box>
                
                <Box sx={{ p: k => k.spacing(2), bgcolor: alpha(theme.palette.error.main, 0.1), borderRadius: 2, minWidth: 150 }}>
                  <Typography variant="body2" color="text.secondary" gutterBottom>错误率</Typography>
                  <Typography variant="h6">{selectedResult.errorRate.toFixed(2)}%</Typography>
                </Box>
              </Box>
              
              <Box sx={{ mt: 2 }}>
                <Typography variant="subtitle1" gutterBottom>响应时间对比</Typography>
                <Box sx={{ mt: 2 }}>
                  <Typography gutterBottom>平均响应时间: {selectedResult.avgResponseTime.toFixed(1)} ms</Typography>
                  <LinearProgress 
                    variant="determinate" 
                    value={getPercentage(selectedResult.avgResponseTime, 500)}
                    sx={{ height: 20, borderRadius: 2, mb: 2 }}
                  />
                  
                  <Typography gutterBottom>P95响应时间: {selectedResult.p95ResponseTime.toFixed(1)} ms</Typography>
                  <LinearProgress 
                    variant="determinate" 
                    value={getPercentage(selectedResult.p95ResponseTime, 800)}
                    color="secondary"
                    sx={{ height: 20, borderRadius: 2, mb: 2 }}
                  />
                  
                  <Typography gutterBottom>P99响应时间: {selectedResult.p99ResponseTime.toFixed(1)} ms</Typography>
                  <LinearProgress 
                    variant="determinate" 
                    value={getPercentage(selectedResult.p99ResponseTime, 1000)}
                    color="warning"
                    sx={{ height: 20, borderRadius: 2, mb: 2 }}
                  />
                  
                  <Typography gutterBottom>最大响应时间: {selectedResult.maxResponseTime.toFixed(1)} ms</Typography>
                  <LinearProgress 
                    variant="determinate" 
                    value={getPercentage(selectedResult.maxResponseTime, 1000)}
                    color="error"
                    sx={{ height: 20, borderRadius: 2 }}
                  />
                </Box>
              </Box>
            </Box>
          ) : (
            <Box sx={{ 
              height: 300, 
              display: 'flex', 
              justifyContent: 'center', 
              alignItems: 'center',
              color: 'text.secondary'
            }}>
              <Typography>请选择一个测试结果查看详情</Typography>
            </Box>
          )}
        </Paper>
      </Box>
      
      {/* 模拟投票对话框 */}
      <Dialog open={showSimulateDialog} onClose={() => setShowSimulateDialog(false)} maxWidth="md" fullWidth>
        <DialogTitle>模拟投票 - {selectedPoll?.question}</DialogTitle>
        <DialogContent dividers>
          {selectedPoll ? (
            <>
              <Box sx={{ mb: 3 }}>
                <Typography variant="subtitle1" gutterBottom>当前投票情况：</Typography>
                <TableContainer>
                  <Table size="small">
                    <TableHead>
                      <TableRow>
                        <TableCell>选项</TableCell>
                        <TableCell align="right">票数</TableCell>
                        <TableCell align="right">百分比</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {selectedPoll.options.map(option => {
                        const totalVotes = selectedPoll.options.reduce((sum, opt) => sum + opt.votes, 0);
                        return (
                          <TableRow key={option.id}>
                            <TableCell>{option.text}</TableCell>
                            <TableCell align="right">{option.votes}</TableCell>
                            <TableCell align="right">{formatPercentage(option.votes, totalVotes)}</TableCell>
                          </TableRow>
                        );
                      })}
                    </TableBody>
                  </Table>
                </TableContainer>
              </Box>
              
              <Divider sx={{ my: 2 }} />
              
              <Box sx={{ mt: 3 }}>
                <Typography variant="subtitle1" gutterBottom>模拟设置：</Typography>
                <Box sx={{ mb: 2 }}>
                  <Typography gutterBottom>模拟投票数量: {votesToAdd}</Typography>
                  <Slider
                    value={votesToAdd}
                    onChange={(_, value) => setVotesToAdd(value as number)}
                    min={1}
                    max={100}
                    valueLabelDisplay="auto"
                    marks={[
                      { value: 1, label: '1' },
                      { value: 25, label: '25' },
                      { value: 50, label: '50' },
                      { value: 75, label: '75' },
                      { value: 100, label: '100' }
                    ]}
                  />
                </Box>
                
                <FormControlLabel
                  control={<Switch checked={saveVotes} onChange={(e) => setSaveVotes(e.target.checked)} />}
                  label="保存模拟结果"
                />
                
                <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
                  {saveVotes 
                    ? "启用后，模拟投票的结果将被永久保存" 
                    : "禁用后，模拟投票只会临时显示，不会保存结果"}
                </Typography>
              </Box>
            </>
          ) : (
            <Typography>未找到投票信息</Typography>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setShowSimulateDialog(false)}>取消</Button>
          <Button 
            variant="contained" 
            color="primary" 
            onClick={() => {
              simulateVoting();
              if (!saveVotes) {
                // 如果不保存，直接关闭对话框
                setTimeout(() => setShowSimulateDialog(false), 1500);
              }
            }}
            startIcon={saveVotes ? <SaveIcon /> : null}
          >
            {saveVotes ? "模拟并保存" : "仅模拟显示"}
          </Button>
        </DialogActions>
      </Dialog>
      
      {/* 编辑投票对话框 */}
      <Dialog open={showEditPollDialog} onClose={() => setShowEditPollDialog(false)} maxWidth="md" fullWidth>
        <DialogTitle>
          编辑投票
          <IconButton 
            size="small" 
            color="error" 
            sx={{ position: 'absolute', right: 16, top: 12 }}
            onClick={() => setShowDeleteConfirm(true)}
          >
            <DeleteIcon />
          </IconButton>
        </DialogTitle>
        <DialogContent dividers>
          {editingPoll ? (
            <>
              <TextField
                label="投票问题"
                fullWidth
                value={editingPoll.question}
                onChange={(e) => handleEditPollChange('question', e.target.value)}
                margin="normal"
              />
              
              <FormControl component="fieldset" sx={{ mt: 2, mb: 3 }}>
                <FormLabel component="legend">投票类型</FormLabel>
                <RadioGroup 
                  row 
                  value={editingPoll.type} 
                  onChange={(e) => handleEditPollChange('type', e.target.value)}
                >
                  <FormControlLabel value="single" control={<Radio />} label="单选" />
                  <FormControlLabel value="multiple" control={<Radio />} label="多选" />
                </RadioGroup>
              </FormControl>
              
              <Typography variant="subtitle1" gutterBottom>选项：</Typography>
              
              {editingPoll.options.map((option, index) => (
                <Box key={option.id} sx={{ display: 'flex', gap: 2, mb: 2, alignItems: 'center' }}>
                  <TextField
                    label={`选项 ${index + 1}`}
                    value={option.text}
                    onChange={(e) => handleEditOptionChange(option.id, 'text', e.target.value)}
                    sx={{ flex: 1 }}
                  />
                  <TextField
                    label="票数"
                    type="number"
                    value={option.votes}
                    onChange={(e) => handleEditOptionChange(option.id, 'votes', parseInt(e.target.value))}
                    sx={{ width: 120 }}
                  />
                  <Tooltip title="删除选项">
                    <span>
                      <IconButton 
                        color="error" 
                        onClick={() => removeOption(option.id)}
                        disabled={editingPoll.options.length <= 2}
                      >
                        <DeleteIcon />
                      </IconButton>
                    </span>
                  </Tooltip>
                </Box>
              ))}
              
              <Button 
                variant="outlined" 
                color="primary" 
                onClick={addNewOption}
                sx={{ mt: 1 }}
              >
                添加选项
              </Button>
            </>
          ) : (
            <Typography>加载中...</Typography>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setShowEditPollDialog(false)}>取消</Button>
          <Button 
            variant="contained" 
            color="primary" 
            onClick={saveEditedPoll}
            startIcon={<SaveIcon />}
          >
            保存修改
          </Button>
        </DialogActions>
      </Dialog>
      
      {/* 删除确认对话框 */}
      <Dialog open={showDeleteConfirm} onClose={() => setShowDeleteConfirm(false)}>
        <DialogTitle>确认删除</DialogTitle>
        <DialogContent>
          <Typography>确定要删除这个投票吗？此操作不可撤销。</Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setShowDeleteConfirm(false)}>取消</Button>
          <Button 
            variant="contained" 
            color="error" 
            onClick={deletePoll}
          >
            确认删除
          </Button>
        </DialogActions>
      </Dialog>
    </Container>
  );
};

export default PerformanceTest; 