import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { 
  Container, 
  Typography, 
  Button, 
  Box, 
  Card, 
  CardContent,
  CardActions,
  Grid,
  Chip,
  CircularProgress,
  useTheme,
  alpha,
  Paper,
  Skeleton
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import { getPolls } from '../api/pollsApi';
import { Poll } from '../types';
import HowToVoteIcon from '@mui/icons-material/HowToVote';
import BarChartIcon from '@mui/icons-material/BarChart';

const PollList: React.FC = () => {
  const [polls, setPolls] = useState<Poll[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const navigate = useNavigate();
  const theme = useTheme();

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

  // 渐变背景样式
  const gradientStyle = {
    background: 'linear-gradient(145deg, rgba(255,255,255,0.9) 0%, rgba(249,250,251,0.95) 100%)',
    backdropFilter: 'blur(20px)'
  };

  useEffect(() => {
    const fetchPolls = async () => {
      try {
        setLoading(true);
        const data = await getPolls();
        
        // 处理字段映射，确保前端组件能够访问所需属性
        const processedData = data.map(poll => ({
          ...poll,
          id: poll.id || poll.ID,
          title: poll.title || poll.question,
          active: poll.active !== undefined ? poll.active : poll.is_active
        }));
        
        setPolls(processedData);
        setError(null);
      } catch (err) {
        console.error('Failed to fetch polls:', err);
        setError('获取投票列表失败，请稍后再试');
      } finally {
        setLoading(false);
      }
    };

    fetchPolls();
  }, []);

  const handleCreatePoll = () => {
    navigate('/create');
  };

  const handleViewPoll = (pollId: number | undefined) => {
    if (pollId !== undefined) {
      navigate(`/poll/${pollId}`);
    } else {
      // 如果ID未定义，显示错误信息
      setError('无法查看投票详情：投票ID未定义');
    }
  };

  if (loading) {
    return (
      <Container maxWidth="md" sx={{ my: 6 }}>
        <Box sx={{ 
          display: 'flex', 
          justifyContent: 'space-between', 
          alignItems: 'center',
          mb: 4 
        }}>
          <Skeleton variant="text" width="40%" height={50} />
          <Skeleton variant="rectangular" width={120} height={40} sx={{ borderRadius: 5 }} />
        </Box>
        <Grid container spacing={3}>
          {[1, 2, 3, 4].map((item) => (
            <Grid item xs={12} md={6} key={item}>
              <Skeleton 
                variant="rectangular" 
                height={180} 
                sx={{ borderRadius: 4 }} 
              />
            </Grid>
          ))}
        </Grid>
      </Container>
    );
  }

  if (error) {
    return (
      <Container maxWidth="md" sx={{ my: 6 }}>
        <Paper 
          elevation={0}
          sx={{ 
            p: 4, 
            borderRadius: '24px',
            border: '1px solid',
            borderColor: 'divider',
            textAlign: 'center',
            ...gradientStyle
          }}
        >
          <HowToVoteIcon color="error" sx={{ fontSize: 60, opacity: 0.7, mb: 2 }} />
          <Typography color="error" variant="h5" gutterBottom>
            {error}
          </Typography>
          <Typography color="text.secondary" sx={{ mb: 3 }}>
            请检查您的网络连接，稍后再试
          </Typography>
          <Button 
            variant="contained" 
            onClick={() => window.location.reload()}
            sx={{ 
              borderRadius: '20px',
              textTransform: 'none',
              px: 4
            }}
          >
            重新加载
          </Button>
        </Paper>
      </Container>
    );
  }

  return (
    <>
      <Box sx={pageBackground} />
      
      <Container maxWidth="md" sx={{ my: 6, position: 'relative' }}>
        <Paper 
          elevation={0}
          sx={{ 
            p: 3, 
            mb: 4, 
            borderRadius: '24px',
            display: 'flex', 
            justifyContent: 'space-between', 
            alignItems: 'center',
            ...gradientStyle,
            boxShadow: '0 4px 20px rgba(0,0,0,0.06)',
            border: '1px solid',
            borderColor: 'divider',
            background: 'linear-gradient(to right, rgba(255,255,255,0.98), rgba(250,252,255,0.98))',
          }}
        >
          <Box sx={{ display: 'flex', alignItems: 'center' }}>
            <HowToVoteIcon 
              color="primary" 
              sx={{ 
                fontSize: 40, 
                mr: 2,
                color: theme.palette.primary.main,
                filter: 'drop-shadow(0 2px 5px rgba(66, 133, 244, 0.3))'
              }} 
            />
            <Typography 
              variant="h4" 
              component="h1"
              sx={{ 
                fontWeight: 600,
                letterSpacing: '-0.5px',
                textShadow: '0 1px 1px rgba(0,0,0,0.05)',
                backgroundImage: 'linear-gradient(45deg, #3367D6, #4285F4)',
                backgroundClip: 'text',
                WebkitBackgroundClip: 'text',
                color: 'transparent',
              }}
            >
              实时投票系统
            </Typography>
          </Box>
          <Button 
            variant="contained" 
            color="primary" 
            onClick={handleCreatePoll}
            startIcon={<AddIcon />}
            sx={{ 
              borderRadius: '20px',
              textTransform: 'none',
              boxShadow: '0 4px 10px rgba(66, 133, 244, 0.25)',
              py: 1.2,
              px: 3,
              fontWeight: 600,
              background: 'linear-gradient(45deg, #4285F4, #5E97F6)',
              '&:hover': {
                boxShadow: '0 6px 15px rgba(66, 133, 244, 0.35)',
                background: 'linear-gradient(45deg, #3367D6, #4285F4)',
              }
            }}
          >
            创建投票
          </Button>
        </Paper>

        {polls.length === 0 ? (
          <Paper 
            elevation={0} 
            sx={{ 
              p: 6, 
              textAlign: 'center', 
              my: 4,
              borderRadius: '24px',
              border: '1px solid',
              borderColor: 'divider',
              ...gradientStyle,
              background: 'linear-gradient(to bottom, rgba(255,255,255,0.98), rgba(250,252,255,0.95))',
              boxShadow: '0 5px 20px rgba(0,0,0,0.05)',
            }}
          >
            <BarChartIcon sx={{ 
              fontSize: 70, 
              color: alpha(theme.palette.primary.main, 0.8), 
              mb: 2,
              filter: 'drop-shadow(0 3px 6px rgba(66, 133, 244, 0.2))'
            }} />
            <Typography variant="h5" sx={{ 
              mb: 2, 
              color: theme.palette.text.primary,
              fontWeight: 600,
              letterSpacing: '-0.5px',
            }}>
              暂无投票
            </Typography>
            <Typography variant="body1" color="text.secondary" sx={{ 
              mb: 4, 
              maxWidth: '500px', 
              mx: 'auto',
              fontSize: '1.05rem',
              lineHeight: 1.6,
            }}>
              创建您的第一个投票，开始收集实时反馈！您可以创建单选或多选题，实时查看投票结果。
            </Typography>
            <Button 
              variant="contained" 
              color="primary" 
              onClick={handleCreatePoll}
              startIcon={<AddIcon />}
              sx={{ 
                borderRadius: '20px',
                textTransform: 'none',
                py: 1.2,
                px: 4,
                fontWeight: 600,
                boxShadow: '0 4px 10px rgba(66, 133, 244, 0.25)',
                background: 'linear-gradient(45deg, #4285F4, #5E97F6)',
                '&:hover': {
                  boxShadow: '0 6px 15px rgba(66, 133, 244, 0.35)',
                  background: 'linear-gradient(45deg, #3367D6, #4285F4)',
                }
              }}
            >
              创建第一个投票
            </Button>
          </Paper>
        ) : (
          <Grid container spacing={3}>
            {polls.map((poll) => (
              <Grid item xs={12} md={6} key={poll.id}>
                <Card 
                  elevation={0}
                  sx={{ 
                    height: '100%', 
                    display: 'flex', 
                    flexDirection: 'column',
                    transition: 'all 0.3s ease-in-out',
                    borderRadius: '24px',
                    border: '1px solid',
                    borderColor: 'divider',
                    overflow: 'hidden',
                    ...gradientStyle,
                    background: 'linear-gradient(to bottom, rgba(255,255,255,0.97), rgba(250,252,255,0.97))',
                    '&:hover': {
                      transform: 'translateY(-8px)',
                      boxShadow: '0 12px 24px rgba(66, 133, 244, 0.15)',
                      borderColor: alpha(theme.palette.primary.main, 0.1),
                    }
                  }}
                >
                  <CardContent sx={{ flexGrow: 1, p: 3 }}>
                    <Typography 
                      variant="h5" 
                      component="h2" 
                      gutterBottom
                      sx={{ 
                        fontWeight: 600,
                        fontSize: '1.3rem',
                        color: theme.palette.primary.dark,
                        lineHeight: 1.3,
                      }}
                    >
                      {poll.title || poll.question}
                    </Typography>
                    {(poll.description || (poll.question && poll.title !== poll.question)) && (
                      <Typography 
                        variant="body2" 
                        color="text.secondary" 
                        gutterBottom
                        sx={{ 
                          mb: 2,
                          display: '-webkit-box',
                          WebkitLineClamp: 2,
                          WebkitBoxOrient: 'vertical',
                          overflow: 'hidden',
                          textOverflow: 'ellipsis',
                          lineHeight: 1.5
                        }}
                      >
                        {poll.description || (poll.question !== poll.title ? poll.question : '')}
                      </Typography>
                    )}
                    <Box sx={{ mt: 3, display: 'flex', gap: 1, flexWrap: 'wrap' }}>
                      <Chip 
                        label={`${poll.options.length} 个选项`} 
                        size="small" 
                        color="primary" 
                        variant="outlined" 
                        sx={{ 
                          borderRadius: '10px',
                          color: theme.palette.primary.main,
                          borderColor: alpha(theme.palette.primary.main, 0.4),
                          fontWeight: 500,
                          fontSize: '0.8125rem',
                        }}
                      />
                      {poll.poll_type === 1 && (
                        <Chip 
                          label="多选题" 
                          size="small" 
                          color="info" 
                          variant="outlined" 
                          sx={{ 
                            borderRadius: '10px',
                            color: theme.palette.info.main,
                            borderColor: alpha(theme.palette.info.main, 0.4),
                            fontWeight: 500,
                            fontSize: '0.8125rem',
                          }}
                        />
                      )}
                    </Box>
                  </CardContent>
                  <CardActions sx={{ px: 3, pb: 3, pt: 0 }}>
                    <Button 
                      onClick={() => handleViewPoll(poll.id)}
                      color="primary"
                      sx={{ 
                        borderRadius: '16px',
                        textTransform: 'none',
                        py: 0.8,
                        px: 2.5,
                        fontWeight: 500,
                        color: theme.palette.primary.main,
                        '&:hover': {
                          backgroundColor: alpha(theme.palette.primary.main, 0.08),
                          color: theme.palette.primary.dark,
                        }
                      }}
                    >
                      查看详情
                    </Button>
                  </CardActions>
                </Card>
              </Grid>
            ))}
          </Grid>
        )}
      </Container>
    </>
  );
};

export default PollList; 