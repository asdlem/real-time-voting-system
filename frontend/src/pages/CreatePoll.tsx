import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { 
  Container, 
  Typography, 
  TextField, 
  Button, 
  Box, 
  IconButton, 
  Paper,
  List,
  ListItem,
  ListItemText,
  Divider,
  Alert,
  FormControlLabel,
  Switch,
  FormControl,
  FormLabel,
  RadioGroup,
  Radio,
  alpha,
  useTheme,
  CircularProgress
} from '@mui/material';
import DeleteIcon from '@mui/icons-material/Delete';
import AddIcon from '@mui/icons-material/Add';
import CreateIcon from '@mui/icons-material/Create';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import { createPoll } from '../api/pollsApi';
import { CreatePollRequest } from '../types';

const CreatePoll: React.FC = () => {
  const navigate = useNavigate();
  const theme = useTheme();
  const [title, setTitle] = useState<string>('');
  const [description, setDescription] = useState<string>('');
  const [options, setOptions] = useState<string[]>(['', '']);
  const [active, setActive] = useState<boolean>(true);
  const [pollType, setPollType] = useState<number>(0); // 0: 单选, 1: 多选
  const [loading, setLoading] = useState<boolean>(false);
  const [error, setError] = useState<string | null>(null);
  const [validationErrors, setValidationErrors] = useState<{
    title?: string;
    options?: string[];
  }>({});
  const [minOptions, setMinOptions] = useState<number>(1);
  const [maxOptions, setMaxOptions] = useState<number>(0); // 0表示不限制

  // 页面背景样式
  const pageBackground = {
    position: 'fixed',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    background: 'linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%)',
    zIndex: -1,
  } as React.CSSProperties;

  // 渐变背景样式
  const gradientStyle = {
    background: 'linear-gradient(145deg, rgba(255,255,255,0.9) 0%, rgba(249,250,251,0.95) 100%)',
    backdropFilter: 'blur(20px)'
  };

  const handleTitleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setTitle(e.target.value);
    if (validationErrors.title) {
      setValidationErrors({
        ...validationErrors,
        title: undefined
      });
    }
  };

  const handleDescriptionChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setDescription(e.target.value);
  };

  const handleOptionChange = (index: number, value: string) => {
    const newOptions = [...options];
    newOptions[index] = value;
    setOptions(newOptions);

    if (validationErrors.options && validationErrors.options[index]) {
      const newOptionErrors = [...(validationErrors.options || [])];
      newOptionErrors[index] = '';
      setValidationErrors({
        ...validationErrors,
        options: newOptionErrors
      });
    }
  };

  const handleAddOption = () => {
    setOptions([...options, '']);
  };

  const handleRemoveOption = (index: number) => {
    if (options.length <= 2) {
      return; // 保持最少两个选项
    }
    const newOptions = [...options];
    newOptions.splice(index, 1);
    setOptions(newOptions);
  };

  const handleActiveChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setActive(e.target.checked);
  };

  const handlePollTypeChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    // 确保始终转换为数字
    setPollType(parseInt(e.target.value, 10));
    console.log('[创建投票] 投票类型已更改为:', parseInt(e.target.value, 10));
  };

  const handleMinOptionsChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = parseInt(e.target.value, 10);
    setMinOptions(value);
    // 如果最小值大于最大值（且最大值不为0），则更新最大值
    if (maxOptions !== 0 && value > maxOptions) {
      setMaxOptions(value);
    }
  };

  const handleMaxOptionsChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = parseInt(e.target.value, 10);
    setMaxOptions(value);
    // 如果最大值小于最小值且不为0，则更新最小值
    if (value !== 0 && value < minOptions) {
      setMinOptions(value);
    }
  };

  const validateForm = (): boolean => {
    const errors: {
      title?: string;
      options?: string[];
    } = {};
    let isValid = true;

    // 验证标题
    if (!title.trim()) {
      errors.title = '请输入投票标题';
      isValid = false;
    }

    // 验证选项
    const optionErrors: string[] = [];
    let hasOptionError = false;

    options.forEach((option, index) => {
      if (!option.trim()) {
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

    setValidationErrors(errors);
    return isValid;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!validateForm()) {
      return;
    }
    
    try {
      setLoading(true);
      setError(null);
      
      const filteredOptions = options.filter(opt => opt.trim() !== '');
      
      // 简化请求数据，只保留必要字段
      const pollData: CreatePollRequest = {
        question: title,
        poll_type: parseInt(String(pollType), 10), // 确保poll_type是数字
        options: filteredOptions.map(text => ({ text })),
        active: active
      };
      
      // 仅当是多选且设置了最小/最大选项数时才添加
      if (pollType === 1) {
        if (minOptions > 0) {
          pollData.min_options = minOptions;
        }
        if (maxOptions > 0) {
          pollData.max_options = maxOptions;
        }
      }
      
      // 仅当描述不为空时才添加
      if (description.trim()) {
        pollData.description = description.trim();
      }
      
      console.log('[创建投票] 准备发送的数据:', pollData);
      
      const createdPoll = await createPoll(pollData);
      navigate(`/poll/${createdPoll.id || createdPoll.ID}`);
      
    } catch (err) {
      console.error('Failed to create poll:', err);
      setError('创建投票失败，请稍后再试');
    } finally {
      setLoading(false);
    }
  };

  const handleCancel = () => {
    navigate('/');
  };

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
            alignItems: 'center',
            ...gradientStyle,
            boxShadow: '0 4px 20px rgba(0,0,0,0.06)'
          }}
        >
          <Button 
            variant="outlined" 
            onClick={handleCancel} 
            startIcon={<ArrowBackIcon />}
            sx={{ 
              mr: 3,
              borderRadius: '20px',
              textTransform: 'none',
              px: 2.5
            }}
          >
            返回列表
          </Button>
          
          <Box sx={{ display: 'flex', alignItems: 'center' }}>
            <CreateIcon 
              color="primary" 
              sx={{ 
                fontSize: 30, 
                mr: 2,
                color: theme.palette.primary.main
              }} 
            />
            <Typography 
              variant="h4" 
              component="h1"
              sx={{ 
                fontWeight: 500,
                letterSpacing: '-0.5px',
                color: theme.palette.primary.dark
              }}
            >
              创建新投票
            </Typography>
          </Box>
        </Paper>

        <Paper 
          elevation={0} 
          sx={{ 
            p: 4, 
            borderRadius: '24px',
            border: '1px solid',
            borderColor: 'divider',
            ...gradientStyle
          }}
        >
          <Box component="form" onSubmit={handleSubmit} noValidate>
            <TextField
              margin="normal"
              required
              fullWidth
              id="title"
              label="投票标题"
              name="title"
              value={title}
              onChange={handleTitleChange}
              error={!!validationErrors.title}
              helperText={validationErrors.title}
              autoFocus
              sx={{
                '& .MuiOutlinedInput-root': {
                  borderRadius: 2,
                  fontSize: '1.1rem'
                },
                '& .MuiInputLabel-root': {
                  fontSize: '1.1rem'
                }
              }}
            />
            
            <TextField
              margin="normal"
              fullWidth
              id="description"
              label="描述（可选）"
              name="description"
              value={description}
              onChange={handleDescriptionChange}
              multiline
              rows={3}
              sx={{
                '& .MuiOutlinedInput-root': {
                  borderRadius: 2
                }
              }}
            />
            
            <Box sx={{ 
              mt: 4,
              p: 3,
              borderRadius: 3,
              bgcolor: alpha(theme.palette.primary.main, 0.03),
              border: '1px solid',
              borderColor: alpha(theme.palette.primary.main, 0.1)
            }}>
              <FormControl component="fieldset">
                <FormLabel 
                  component="legend"
                  sx={{ 
                    fontSize: '1.1rem',
                    fontWeight: 500,
                    color: theme.palette.primary.dark,
                    '&.Mui-focused': {
                      color: theme.palette.primary.dark
                    }
                  }}
                >
                  投票类型
                </FormLabel>
                <RadioGroup
                  row
                  value={pollType.toString()}
                  onChange={handlePollTypeChange}
                  sx={{ mt: 1 }}
                >
                  <FormControlLabel 
                    value="0" 
                    control={
                      <Radio 
                        sx={{ 
                          '&.Mui-checked': { color: theme.palette.primary.main }
                        }} 
                      />
                    } 
                    label={
                      <Typography sx={{ fontWeight: 500 }}>
                        单选题
                      </Typography>
                    }
                    sx={{ mr: 4 }}
                  />
                  <FormControlLabel 
                    value="1" 
                    control={
                      <Radio 
                        sx={{ 
                          '&.Mui-checked': { color: theme.palette.primary.main }
                        }} 
                      />
                    } 
                    label={
                      <Typography sx={{ fontWeight: 500 }}>
                        多选题
                      </Typography>
                    }
                  />
                </RadioGroup>
              </FormControl>
            </Box>
            
            {pollType === 1 && (
              <Box sx={{ mt: 2 }}>
                <Typography variant="subtitle2" sx={{ mb: 1, color: theme.palette.text.secondary }}>
                  选项限制设置
                </Typography>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                  <TextField
                    label="最少选择"
                    type="number"
                    size="small"
                    value={minOptions}
                    onChange={handleMinOptionsChange}
                    InputProps={{
                      inputProps: { min: 1, max: options.length }
                    }}
                    sx={{ width: 120 }}
                    helperText="至少选择的选项数"
                  />
                  <TextField
                    label="最多选择"
                    type="number"
                    size="small"
                    value={maxOptions}
                    onChange={handleMaxOptionsChange}
                    InputProps={{
                      inputProps: { min: 0, max: options.length }
                    }}
                    sx={{ width: 120 }}
                    helperText="0表示不限制"
                  />
                </Box>
                <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 1 }}>
                  {maxOptions === 0 
                    ? `用户必须至少选择 ${minOptions} 个选项，无上限限制`
                    : maxOptions === minOptions
                    ? `用户必须选择恰好 ${minOptions} 个选项`
                    : `用户必须选择 ${minOptions} 到 ${maxOptions} 个选项`}
                </Typography>
              </Box>
            )}
            
            <Box sx={{ mt: 4 }}>
              <Typography 
                variant="h6" 
                gutterBottom
                sx={{ 
                  fontSize: '1.2rem',
                  fontWeight: 500,
                  color: theme.palette.primary.dark,
                  display: 'flex',
                  alignItems: 'center'
                }}
              >
                投票选项
                <Typography 
                  component="span" 
                  sx={{ 
                    ml: 2, 
                    fontSize: '0.9rem', 
                    color: theme.palette.text.secondary,
                    fontWeight: 'normal'
                  }}
                >
                  (至少需要两个选项)
                </Typography>
              </Typography>
              
              <List>
                {options.map((option, index) => (
                  <ListItem
                    key={index}
                    secondaryAction={
                      <IconButton 
                        edge="end" 
                        onClick={() => handleRemoveOption(index)}
                        disabled={options.length <= 2}
                        sx={{ 
                          color: options.length <= 2 ? 'action.disabled' : 'error.light',
                          '&:hover': {
                            color: options.length <= 2 ? 'action.disabled' : 'error.main',
                          }
                        }}
                      >
                        <DeleteIcon />
                      </IconButton>
                    }
                    sx={{ pl: 0 }}
                  >
                    <ListItemText 
                      primary={
                        <TextField
                          fullWidth
                          required
                          size="small"
                          label={`选项 ${index + 1}`}
                          value={option}
                          onChange={(e) => handleOptionChange(index, e.target.value)}
                          error={!!(validationErrors.options && validationErrors.options[index])}
                          helperText={validationErrors.options && validationErrors.options[index]}
                          sx={{
                            '& .MuiOutlinedInput-root': {
                              borderRadius: 2
                            }
                          }}
                        />
                      } 
                    />
                  </ListItem>
                ))}
                
                <ListItem sx={{ pl: 0 }}>
                  <Button 
                    startIcon={<AddIcon />} 
                    onClick={handleAddOption}
                    sx={{ 
                      mt: 1, 
                      borderRadius: '10px',
                      textTransform: 'none',
                      color: theme.palette.primary.main,
                      fontWeight: 500,
                      bgcolor: alpha(theme.palette.primary.main, 0.04),
                      '&:hover': {
                        bgcolor: alpha(theme.palette.primary.main, 0.08)
                      }
                    }}
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
                    checked={active} 
                    onChange={handleActiveChange} 
                    name="active" 
                    color="primary"
                  />
                }
                label={
                  <Typography sx={{ fontWeight: 500 }}>
                    立即激活投票
                  </Typography>
                }
                sx={{
                  '& .MuiSwitch-switchBase.Mui-checked': {
                    color: theme.palette.primary.main
                  },
                  '& .MuiSwitch-switchBase.Mui-checked + .MuiSwitch-track': {
                    backgroundColor: alpha(theme.palette.primary.main, 0.5)
                  }
                }}
              />
              <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5, ml: 6.5 }}>
                激活后，用户可以立即开始投票
              </Typography>
            </Box>
            
            {error && (
              <Alert 
                severity="error" 
                sx={{ 
                  mt: 3, 
                  borderRadius: '12px',
                  boxShadow: '0 2px 8px rgba(0,0,0,0.05)'
                }}
              >
                {error}
              </Alert>
            )}
            
            <Box sx={{ mt: 4, display: 'flex', justifyContent: 'flex-end' }}>
              <Button
                type="button"
                onClick={handleCancel}
                sx={{ 
                  mr: 2,
                  borderRadius: '20px',
                  textTransform: 'none',
                  px: 3,
                  py: 1,
                  borderColor: alpha(theme.palette.grey[400], 0.5),
                  color: theme.palette.grey[700]
                }}
              >
                取消
              </Button>
              <Button
                type="submit"
                variant="contained"
                disabled={loading}
                sx={{ 
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
                {loading ? (
                  <Box sx={{ display: 'flex', alignItems: 'center' }}>
                    <CircularProgress size={16} color="inherit" sx={{ mr: 1 }} />
                    创建中...
                  </Box>
                ) : '创建投票'}
              </Button>
            </Box>
          </Box>
        </Paper>
      </Container>
    </>
  );
};

export default CreatePoll; 