import React, { useState, useEffect } from 'react';
import { apiRequest } from '../lib/api';

import { GlassCard } from './ui/GlassCard';
import { AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';

export default function SystemHealthCard() {
    const [metrics, setMetrics] = useState(null);
    const [history, setHistory] = useState([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState(null);

    const fetchMetrics = async () => {
        try {
            const data = await apiRequest('/api/v1/system/metrics/live');
            setMetrics(data);

            setHistory(prev => {
                const newState = [...prev, { time: new Date().toLocaleTimeString(), cpu: data.cpu_percent, ram: data.ram_percent }];
                if (newState.length > 20) newState.shift(); // Keep last 20 points
                return newState;
            });

            setLoading(false);
        } catch (err) {
            console.error("Failed to fetch live metrics:", err);
            setError("Verbindung fehlgeschlagen");
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchMetrics();
        const interval = setInterval(fetchMetrics, 5000); // Poll every 5s
        return () => clearInterval(interval);
    }, []);

    if (loading) return <GlassCard className="h-64 animate-pulse bg-gray-800/50" />;
    if (error) return <GlassCard className="h-64 flex items-center justify-center text-red-400">{error}</GlassCard>;

    const getStatusColor = (val) => {
        if (val > 90) return 'text-red-400';
        if (val > 70) return 'text-yellow-400';
        return 'text-green-400';
    };

    const formatBytes = (bytes) => {
        if (!bytes) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    };

    return (
        <GlassCard className="p-6">
            <h3 className="text-xl font-bold bg-gradient-to-r from-blue-400 to-purple-400 bg-clip-text text-transparent mb-6">
                Live System Status
            </h3>

            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                {/* CPU Gauge (Simplified as Percentage Display for now) */}
                <div className="bg-gray-800/50 rounded-lg p-4 flex flex-col items-center justify-center relative overflow-hidden group">
                    <div className={`text-4xl font-mono font-bold ${getStatusColor(metrics.cpu_percent)}`}>
                        {metrics.cpu_percent}%
                    </div>
                    <div className="text-gray-400 text-sm mt-2">CPU Auslastung</div>
                    {/* Decorative Ring */}
                    <div className="absolute inset-0 border-2 border-dashed border-gray-700/30 rounded-lg pointer-events-none group-hover:border-blue-500/20 transition-colors" />
                </div>

                {/* RAM Bar */}
                <div className="bg-gray-800/50 rounded-lg p-4 space-y-4">
                    <div className="flex justify-between text-sm">
                        <span className="text-gray-400">RAM</span>
                        <span className={getStatusColor(metrics.ram_percent)}>{metrics.ram_percent}%</span>
                    </div>
                    <div className="w-full bg-gray-700 h-2 rounded-full overflow-hidden">
                        <div
                            className={`h-full rounded-full transition-all duration-500 ${metrics.ram_percent > 90 ? 'bg-red-500' : 'bg-blue-500'}`}
                            style={{ width: `${metrics.ram_percent}%` }}
                        />
                    </div>
                    <div className="text-xs text-gray-500 text-right">
                        Total: {formatBytes(metrics.ram_total)}
                    </div>
                </div>

                {/* Disk Bar */}
                <div className="bg-gray-800/50 rounded-lg p-4 space-y-4">
                    <div className="flex justify-between text-sm">
                        <span className="text-gray-400">Disk (/)</span>
                        <span className={getStatusColor(metrics.disk_percent)}>{metrics.disk_percent}%</span>
                    </div>
                    <div className="w-full bg-gray-700 h-2 rounded-full overflow-hidden">
                        <div
                            className={`h-full rounded-full transition-all duration-500 ${metrics.disk_percent > 90 ? 'bg-red-500' : 'bg-purple-500'}`}
                            style={{ width: `${metrics.disk_percent}%` }}
                        />
                    </div>
                    <div className="text-xs text-gray-500 text-right">
                        Total: {formatBytes(metrics.disk_total)}
                    </div>
                </div>
            </div>

            {/* Mini Chart History */}
            <div className="h-32 mt-6 -ml-4">
                <ResponsiveContainer width="100%" height="100%">
                    <AreaChart data={history}>
                        <defs>
                            <linearGradient id="colorCpu" x1="0" y1="0" x2="0" y2="1">
                                <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3} />
                                <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
                            </linearGradient>
                        </defs>
                        <Tooltip
                            contentStyle={{ backgroundColor: '#1f2937', borderColor: '#374151', color: '#fff' }}
                            itemStyle={{ color: '#fff' }}
                        />
                        <Area type="monotone" dataKey="cpu" stroke="#3b82f6" fillOpacity={1} fill="url(#colorCpu)" />
                    </AreaChart>
                </ResponsiveContainer>
            </div>
        </GlassCard>
    );
}
