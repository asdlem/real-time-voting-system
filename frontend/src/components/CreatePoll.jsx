import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Form, Input, Button, Radio, Card, Typography, notification, Divider } from 'antd';
import { PlusOutlined, MinusCircleOutlined, ArrowLeftOutlined } from '@ant-design/icons';
import pollService from '../api/pollService';
import './CreatePoll.css';

const { Title, Text } = Typography;
const { TextArea } = Input;

const CreatePoll = () => {
  const [form] = Form.useForm();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [pollType, setPollType] = useState(0); // 默认单选
  
  // 返回列表
  const handleBack = () => {
    navigate('/');
  };
  
  // 提交表单
  const onFinish = async (values) => {
    try {
      setLoading(true);
      
      // 获取表单当前值并输出调试信息
      const formValues = form.getFieldsValue();
      console.log('表单当前值:', formValues);
      console.log('pollType状态值:', pollType);
      console.log('表单中的poll_type值:', formValues.poll_type);
      
      // 确保使用最终选择的pollType值
      const finalPollType = formValues.poll_type;
      
      // 检查选项数据
      const optionsData = values.options.map(opt => {
        // 确保选项数据格式正确
        if (opt && typeof opt.text === 'string') {
          return { Text: opt.text.trim() }; // 使用大写开头的Text字段
        }
        return null;
      }).filter(opt => opt !== null);
      
      if (optionsData.length < 2) {
        notification.error({
          message: '选项不足',
          description: '请至少提供两个有效选项',
        });
        setLoading(false);
        return;
      }
      
      // 准备提交数据，使用后端期望的字段名（小写）
      const pollData = {
        question: values.question, // 使用小写字段名
        description: values.description,
        is_active: true, 
        poll_type: finalPollType,  // 使用小写的poll_type
        options: optionsData.map(opt => ({
          text: opt.Text || opt.text  // 确保选项文本字段也是小写
        }))
      };
      
      console.log('最终提交数据:', pollData);
      
      // 创建投票
      const response = await pollService.createPoll(pollData);
      console.log('创建成功，响应:', response);
      
      notification.success({
        message: '创建成功',
        description: '投票已成功创建',
      });
      
      // 跳转到投票详情页
      navigate(`/poll/${response.ID}`);
    } catch (err) {
      console.error('创建失败:', err);
      notification.error({
        message: '创建失败',
        description: err.message || '无法创建投票，请稍后再试',
      });
    } finally {
      setLoading(false);
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
        <Title level={2}>创建新投票</Title>
        
        <Form
          form={form}
          name="createPoll"
          layout="vertical"
          onFinish={onFinish}
          onFinishFailed={onFinishFailed}
          initialValues={{
            question: '',
            description: '',
            options: [{ text: '' }, { text: '' }],
            poll_type: 0
          }}
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
              onChange={(e) => {
                const newValue = e.target.value;
                console.log('投票类型选择改变:', newValue);
                setPollType(newValue);
                
                // 确保表单字段值同步更新
                form.setFieldsValue({ poll_type: newValue });
              }}
              value={form.getFieldValue('poll_type') || pollType}
            >
              <Radio value={0}>单选</Radio>
              <Radio value={1}>多选</Radio>
            </Radio.Group>
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
                        
                        {fields.length > 2 && (
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
                    onClick={() => add()}
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
              loading={loading}
              className="submit-button"
              size="large"
            >
              创建投票
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
};

export default CreatePoll; 