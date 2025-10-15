import React from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';

const MetricsChart = ({ data }) => {
  return (
    <ResponsiveContainer width="100%" height={400}>
      <LineChart data={data}>
        <CartesianGrid strokeDasharray="3 3" />
        <XAxis dataKey="window_start" />
        <YAxis />
        <Tooltip />
        <Legend />
        <Line type="monotone" dataKey="invocations" stroke="#8884d8" name="Invocations" />
        <Line type="monotone" dataKey="mem_mb_ms_sum" stroke="#82ca9d" name="Memory (MB×ms)" />
        <Line type="monotone" dataKey="cold_starts" stroke="#ff8042" name="Cold Starts" />
      </LineChart>
    </ResponsiveContainer>
  );
};

export default MetricsChart;
