import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import { List, Card, Button, Typography, Empty, Spin, notification } from 'antd';
import { PlusOutlined, FileOutlined } from '@ant-design/icons';
import pollService from '../api/pollService';
import './PollList.css';

const { Title, Text, Paragraph } = Typography;

const PollList = () => {
  const [polls, setPolls] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    const fetchPolls = async () => {
      try {
        setLoading(true);
        setError(null);
        console.log('获取投票列表');
        const pollsData = await pollService.getPolls();
        console.log('获取到的投票列表:', pollsData);
        setPolls(pollsData);
      } catch (err) {
        console.error('获取投票列表失败:', err);
        setError(err.message || '获取投票列表失败');
        notification.error({
          message: '加载失败',
          description: err.message || '无法加载投票列表',
        });
      } finally {
        setLoading(false);
      }
    };

    fetchPolls();
  }, []);

  if (loading) {
    return (
      <div className="poll-list-container loading">
        <Spin size="large" tip="加载中..." />
      </div>
    );
  }

  if (error) {
    return (
      <div className="poll-list-container error">
        <Title level={4}>加载投票列表时出错</Title>
        <Paragraph>{error}</Paragraph>
      </div>
    );
  }

  return (
    <div className="poll-list-container">
      <div className="list-header">
        <Title level={2}>投票列表</Title>
        <Link to="/create">
          <Button type="primary" icon={<PlusOutlined />}>
            新建投票
          </Button>
        </Link>
      </div>

      {polls.length === 0 ? (
        <Empty
          description="暂无投票"
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          className="empty-list"
        />
      ) : (
        <List
          grid={{
            gutter: 16,
            xs: 1,
            sm: 1,
            md: 2,
            lg: 3,
            xl: 3,
            xxl: 4,
          }}
          dataSource={polls}
          renderItem={poll => (
            <List.Item>
              <Link to={`/poll/${poll.ID}`} className="poll-card-link">
                <Card
                  className="poll-card"
                  hoverable
                >
                  <Card.Meta
                    title={poll.title || poll.question}
                    description={
                      <>
                        <div className="poll-card-info">
                          <Text type="secondary">
                            <FileOutlined /> {poll.options?.length || 0} 个选项
                          </Text>
                          <Text type="secondary">
                            {poll.poll_type === 0 ? '单选' : '多选'}
                          </Text>
                        </div>
                        {poll.description && (
                          <Paragraph ellipsis={{ rows: 2 }} className="poll-description">
                            {poll.description}
                          </Paragraph>
                        )}
                      </>
                    }
                  />
                </Card>
              </Link>
            </List.Item>
          )}
        />
      )}
    </div>
  );
};

export default PollList; 