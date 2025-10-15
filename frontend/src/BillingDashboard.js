import React, { useState, useEffect } from 'react';
import axios from 'axios';
import MetricsChart from './MetricsChart';

const BillingDashboard = () => {
  const [aggregates, setAggregates] = useState([]);
  const [cost, setCost] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchUsageAggregates();
  }, []);

  const fetchUsageAggregates = async () => {
    try {
      const response = await axios.get('http://localhost:8081/api/v1/usage-aggregates', {
        params: {
          tenant_id: '123e4567-e89b-12d3-a456-426614174000',
          start_time: '2025-04-05T00:00:00Z',
          end_time: '2025-04-06T00:00:00Z'
        }
      });
      setAggregates(response.data.data);
      setLoading(false);
    } catch (error) {
      console.error('Error fetching usage aggregates:', error);
      setLoading(false);
    }
  };

  const calculateCost = async () => {
    try {
      const response = await axios.post('http://localhost:8081/api/v1/billing/calculate', {
        tenant_id: '123e4567-e89b-12d3-a456-426614174000',
        start_time: '2025-04-05T00:00:00Z',
        end_time: '2025-04-06T00:00:00Z'
      });
      setCost(response.data);
    } catch (error) {
      console.error('Error calculating cost:', error);
    }
  };

  if (loading) return <div>Loading...</div>;

  return (
    <div>
      <h1>Billing Dashboard</h1>

      <button onClick={calculateCost}>Calculate Cost</button>

      {cost && (
        <div>
          <h2>Cost Breakdown</h2>
          <p>Total Cost: {cost.total_cost} RUB</p>
          <ul>
            {cost.line_items.map((item, i) => (
              <li key={i}>
                {item.name}: {item.amount} {item.unit} = {item.cost} RUB
              </li>
            ))}
          </ul>
        </div>
      )}

      <h2>Metrics Over Time</h2>
      <MetricsChart data={aggregates} />
    </div>
  );
};

export default BillingDashboard;
