import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Form, Input, Button, Radio, Card, Typography, notification, Divider, Spin } from 'antd';
import { PlusOutlined, MinusCircleOutlined, ArrowLeftOutlined } from '@ant-design/icons';
import pollService from '../api/pollService';
import './CreatePoll.css'; // 重用创建投票页面的样式

const { Title, Text } = Typography;
const { TextArea } = Input;

const EditPoll = () => {
  const { id } = useParams();
  const [form] = Form.useForm();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [pollType, setPollType] = useState(0);
  const [poll, setPoll] = useState(null);
  const [error, setError] = useState(null);
  const [hasVotes, setHasVotes] = useState(false);
  
  // 加载投票数据
  useEffect(() => {
    const fetchPoll = async () => {
      try {
        setLoading(true);
        setError(null);
        
        if (!id || isNaN(parseInt(id))) {
          setError('无效的投票ID');
          setLoading(false);
          return;
        }
        
        console.log(`获取投票编辑数据，ID: ${id}`);
        // 尝试使用编辑专用API，如果不存在则使用普通获取API
        let pollData;
        try {
          pollData = await pollService.getPollForEdit(Number(id));
        } catch (err) {
          console.log('编辑专用API不可用，使用普通API获取数据');
          pollData = await pollService.getPollById(Number(id));
        }
        
        console.log('获取到投票数据:', pollData);
        setPoll(pollData);
        
        // 检查是否有投票记录
        let votesExist = false;
        
        // 检查所有可能的投票记录字段
        if (pollData.votes_count > 0 || 
            pollData.VotesCount > 0 || 
            pollData.total_votes > 0 || 
            pollData.vote_count > 0 ||
            pollData.TotalVotes > 0) {
          votesExist = true;
        }
        
        // 或检查每个选项是否有投票
        if (pollData.options && Array.isArray(pollData.options)) {
          const hasOptionVotes = pollData.options.some(option => 
            (option.votes > 0 || option.Votes > 0 || option.vote_count > 0 || option.VoteCount > 0)
          );
          if (hasOptionVotes) {
            votesExist = true;
          }
        }
        
        setHasVotes(votesExist);
        console.log('投票是否有记录:', votesExist);
        
        // 初始化表单数据
        const title = pollData.title || pollData.question || pollData.Question || pollData.Title;
        const description = pollData.description || pollData.Description || '';
        const pollTypeValue = pollData.poll_type !== undefined ? pollData.poll_type : 
                            (pollData.PollType !== undefined ? pollData.PollType : 0);
        setPollType(pollTypeValue);
        
        // 处理选项数据
        const options = (pollData.options || []).map(option => ({
          id: option.id || option.ID,
          text: option.text || option.content || option.Text || option.Content
        }));
        
        // 设置表单初始值
        form.setFieldsValue({
          question: title,
          description: description,
          poll_type: pollTypeValue,
          options: options.length > 0 ? options : [{ text: '' }, { text: '' }]
        });
      } catch (err) {
        console.error('获取投票数据失败:', err);
        setError(err.message || '获取投票数据失败');
        notification.error({
          message: '加载失败',
          description: err.message || '无法加载投票数据',
        });
      } finally {
        setLoading(false);
      }
    };

    if (id) {
      fetchPoll();
    }
  }, [id, form]);
  
  // 返回列表
  const handleBack = () => {
    navigate(`/poll/${id}`);
  };
  
  // 提交表单
  const onFinish = async (values) => {
    try {
      setSubmitting(true);
      
      // 检查选项数据
      const optionsData = values.options.map(opt => {
        // 确保选项数据格式正确
        if (opt && typeof opt.text === 'string') {
          const option = { Text: opt.text.trim() }; // 使用大写开头的Text字段
          // 如果是已有选项，保留ID
          if (opt.id) {
            option.ID = opt.id;
          }
          return option;
        }
        return null;
      }).filter(opt => opt !== null);
      
      if (optionsData.length < 2) {
        notification.error({
          message: '选项不足',
          description: '请至少提供两个有效选项',
        });
        setSubmitting(false);
        return;
      }
      
      // 准备提交数据，使用后端期望的字段名（大写开头）
      const pollData = {
        ID: Number(id),
        Question: values.question, // 使用Question而不是title/question
        Description: values.description,
        Status: poll.status || poll.Status || "active", 
        PollType: values.poll_type,  // 使用表单中的poll_type值
        Options: optionsData
      };
      
      console.log('提交更新数据:', pollData);
      
      // 更新投票
      const response = await pollService.updatePoll(Number(id), pollData);
      
      notification.success({
        message: '更新成功',
        description: '投票已成功更新',
      });
      
      // 跳转到投票详情页
      navigate(`/poll/${id}`);
    } catch (err) {
      console.error('更新失败:', err);
      notification.error({
        message: '更新失败',
        description: err.message || '无法更新投票，请稍后再试',
      });
    } finally {
      setSubmitting(false);
    }
  };
  
  // 表单验证失败
  const onFinishFailed = (errorInfo) => {
    notification.error({
      message: '表单验证失败',
      description: '请检查表单填写是否正确',
    });
    console.error('表单验证失败:', errorInfo);
  };
  
  if (loading) {
    return (
      <div className="create-poll-container loading">
        <Spin size="large" tip="加载中..." />
      </div>
    );
  }

  if (error) {
    return (
      <div className="create-poll-container error">
        <Title level={4}>加载投票数据时出错</Title>
        <p>{error}</p>
        <Button type="primary" icon={<ArrowLeftOutlined />} onClick={() => navigate('/')}>
          返回列表
        </Button>
      </div>
    );
  }
  
  return (
    <div className="create-poll-container">
      <Button 
        className="back-button" 
        type="default" 
        icon={<ArrowLeftOutlined />} 
        onClick={handleBack}
      >
        返回
      </Button>
      
      <Card className="form-card">
        <Title level={2}>编辑投票</Title>
        
        <Form
          form={form}
          name="editPoll"
          layout="vertical"
          onFinish={onFinish}
          onFinishFailed={onFinishFailed}
        >
          {/* 投票标题 */}
          <Form.Item
            label="投票问题"
            name="question"
            rules={[
              { required: true, message: '请输入投票问题' },
              { min: 3, message: '问题至少需要3个字符' },
              { max: 100, message: '问题最多100个字符' }
            ]}
          >
            <Input placeholder="请输入投票问题" maxLength={100} />
          </Form.Item>
          
          {/* 投票描述 */}
          <Form.Item
            label="描述（可选）"
            name="description"
            rules={[
              { max: 500, message: '描述最多500个字符' }
            ]}
          >
            <TextArea 
              placeholder="请输入投票描述（可选）" 
              rows={4}
              maxLength={500}
              showCount
            />
          </Form.Item>
          
          {/* 投票类型 */}
          <Form.Item
            label="投票类型"
            name="poll_type"
            rules={[{ required: true, message: '请选择投票类型' }]}
          >
            <Radio.Group 
              onChange={(e) => setPollType(e.target.value)}
              disabled={hasVotes}
            >
              <Radio value={0}>单选</Radio>
              <Radio value={1}>多选</Radio>
            </Radio.Group>
            {hasVotes && (
              <span className="poll-type-disabled-hint">
                （投票已有人参与，无法修改投票类型）
              </span>
            )}
          </Form.Item>
          
          {/* 选项列表 */}
          <Divider orientation="left">投票选项</Divider>
          <Form.List
            name="options"
            rules={[
              {
                validator: async (_, options) => {
                  if (!options || options.length < 2) {
                    return Promise.reject(new Error('至少需要添加2个选项'));
                  }
                  
                  // 检查是否有空选项
                  const emptyOption = options.find(opt => !opt.text || opt.text.trim() === '');
                  if (emptyOption) {
                    return Promise.reject(new Error('选项内容不能为空'));
                  }
                  
                  return Promise.resolve();
                },
              },
            ]}
          >
            {(fields, { add, remove }, { errors }) => (
              <>
                {fields.map((field, index) => {
                  // 单独提取key避免React warning
                  const key = field.key;
                  const restField = { ...field };
                  delete restField.key;
                  
                  // 获取当前选项数据
                  const fieldOption = form.getFieldValue(['options', field.name]);
                  const hasOptionVotes = fieldOption && fieldOption.votes_count > 0;
                  
                  return (
                    <Form.Item
                      required={false}
                      key={key}
                      className="option-form-item"
                    >
                      <div className="option-row">
                        <div className="option-index">
                          {index + 1}.
                        </div>
                        <Form.Item
                          {...restField}
                          name={[field.name, 'text']}
                          validateTrigger={['onChange', 'onBlur']}
                          rules={[
                            {
                              required: true,
                              whitespace: true,
                              message: "请输入选项内容或删除此选项",
                            },
                            {
                              max: 100,
                              message: "选项内容最多100个字符",
                            },
                          ]}
                          noStyle
                        >
                          <Input placeholder="输入选项内容" maxLength={100} />
                        </Form.Item>
                        
                        {/* 保存选项ID */}
                        <Form.Item 
                          {...restField}
                          name={[field.name, 'id']} 
                          hidden={true} 
                          noStyle
                        >
                          <Input type="hidden" />
                        </Form.Item>
                        
                        {/* 仅当选项没有投票记录且选项数量大于2时，才显示删除按钮 */}
                        {fields.length > 2 && !hasOptionVotes && (
                          <MinusCircleOutlined
                            className="remove-icon"
                            onClick={() => remove(field.name)}
                          />
                        )}
                      </div>
                    </Form.Item>
                  );
                })}
                
                <Form.Item>
                  <Button
                    type="dashed"
                    onClick={() => add({ text: '' })}
                    block
                    icon={<PlusOutlined />}
                    disabled={fields.length >= 10}
                  >
                    添加选项
                  </Button>
                  <Text type="secondary" className="option-limit-text">
                    最多可添加10个选项
                  </Text>
                  <Form.ErrorList errors={errors} />
                </Form.Item>
              </>
            )}
          </Form.List>
          
          {/* 提交按钮 */}
          <Form.Item className="submit-form-item">
            <Button 
              type="primary" 
              htmlType="submit" 
              loading={submitting}
              className="submit-button"
              size="large"
            >
              保存更改
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
};

export default EditPoll; 