import React from 'react';
import { Box, useTheme } from '@mui/material';
import { 
  PieChart, Pie, BarChart, Bar, Cell, XAxis, YAxis, 
  CartesianGrid, Tooltip, Legend, ResponsiveContainer 
} from 'recharts';

interface PollChartProps {
  data: Array<{
    name: string;
    value: number;
    id?: string | number;
  }>;
  type: 'pie' | 'bar';
}

const PollChart: React.FC<PollChartProps> = ({ data, type }) => {
  const theme = useTheme();
  
  // 如果没有数据则显示提示
  if (!data || data.length === 0 || data.every(item => item.value === 0)) {
    return (
      <Box sx={{ 
        height: 250, 
        display: 'flex', 
        alignItems: 'center', 
        justifyContent: 'center',
        color: 'text.secondary'
      }}>
        暂无足够数据显示图表
      </Box>
    );
  }
  
  // 生成颜色数组
  const COLORS = [
    theme.palette.primary.main,
    theme.palette.secondary.main,
    theme.palette.success.main,
    theme.palette.info.main,
    theme.palette.warning.main,
    theme.palette.error.main,
    '#8884d8',
    '#82ca9d',
    '#ffc658',
    '#ff8042'
  ];
  
  // 自定义工具提示内容
  const CustomTooltip = ({ active, payload }: any) => {
    if (active && payload && payload.length) {
      const item = payload[0];
      const totalValue = data.reduce((sum, item) => sum + item.value, 0);
      const percentage = ((item.value / totalValue) * 100).toFixed(1);
      
      return (
        <Box sx={{ 
          bgcolor: 'background.paper', 
          p: 1.5, 
          border: '1px solid',
          borderColor: 'divider',
          borderRadius: 1,
          boxShadow: 1
        }}>
          <div style={{ color: item.color, fontWeight: 'bold' }}>{item.name}</div>
          <div>{item.value} 票 ({percentage}%)</div>
        </Box>
      );
    }
    return null;
  };
  
  // 渲染图表
  return (
    <Box sx={{ width: '100%', height: 300 }}>
      <ResponsiveContainer width="100%" height="100%">
        {type === 'pie' ? (
          <PieChart>
            <Pie
              data={data}
              cx="50%"
              cy="50%"
              labelLine={false}
              outerRadius={80}
              fill="#8884d8"
              dataKey="value"
              nameKey="name"
              label={({ name, percent }) => `${name}: ${(percent ? (percent * 100).toFixed(0) : 0)}%`}
            >
              {data.map((entry, index) => (
                <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
              ))}
            </Pie>
            <Tooltip content={<CustomTooltip />} />
            <Legend />
          </PieChart>
        ) : (
          <BarChart
            data={data}
            layout="vertical"
            margin={{ top: 5, right: 30, left: 20, bottom: 5 }}
          >
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis type="number" />
            <YAxis dataKey="name" type="category" width={100} />
            <Tooltip content={<CustomTooltip />} />
            <Legend />
            <Bar dataKey="value" name="投票数">
              {data.map((entry, index) => (
                <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
              ))}
            </Bar>
          </BarChart>
        )}
      </ResponsiveContainer>
    </Box>
  );
};

export default PollChart; 