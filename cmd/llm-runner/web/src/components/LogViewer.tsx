import { useState, useMemo } from 'react';
import type { ParsedRun } from '../types';
import { Search, AlertCircle, Info, AlertTriangle, XCircle } from 'lucide-react';
import YamlView from './YamlView';
import CopyButton from './CopyButton';

interface Props {
  run: ParsedRun;
  onNavigate?: (tab: any, turnIndex?: number) => void;
}

const LOG_LEVELS = ['trace', 'debug', 'info', 'warn', 'error', 'fatal'];

export default function LogViewer({ run }: Props) {
  const [searchTerm, setSearchTerm] = useState('');
  const [levelFilter, setLevelFilter] = useState<Set<string>>(new Set());

  const toggleLevel = (level: string) => {
    const newSet = new Set(levelFilter);
    if (newSet.has(level)) {
      newSet.delete(level);
    } else {
      newSet.add(level);
    }
    setLevelFilter(newSet);
  };

  const filteredLogs = useMemo(() => {
    return run.logs.filter((log) => {
      if (levelFilter.size > 0 && !levelFilter.has(log.level)) {
        return false;
      }
      if (searchTerm) {
        const searchLower = searchTerm.toLowerCase();
        return (
          log.message.toLowerCase().includes(searchLower) ||
          (log.error && log.error.toLowerCase().includes(searchLower)) ||
          JSON.stringify(log.extra).toLowerCase().includes(searchLower)
        );
      }
      return true;
    });
  }, [run.logs, levelFilter, searchTerm]);

  const getLevelIcon = (level: string) => {
    switch (level) {
      case 'error':
      case 'fatal':
        return <XCircle size={16} className="text-red-400" />;
      case 'warn':
        return <AlertTriangle size={16} className="text-yellow-400" />;
      case 'info':
        return <Info size={16} className="text-blue-400" />;
      default:
        return <AlertCircle size={16} className="text-gray-400" />;
    }
  };

  const getLevelColor = (level: string) => {
    switch (level) {
      case 'error':
      case 'fatal':
        return 'border-red-500 bg-red-900/20';
      case 'warn':
        return 'border-yellow-500 bg-yellow-900/20';
      case 'info':
        return 'border-blue-500 bg-blue-900/20';
      case 'debug':
        return 'border-purple-500 bg-purple-900/20';
      default:
        return 'border-gray-500 bg-gray-900/20';
    }
  };

  const getLevelBadgeColor = (level: string) => {
    switch (level) {
      case 'error':
      case 'fatal':
        return 'bg-red-600 text-white';
      case 'warn':
        return 'bg-yellow-600 text-white';
      case 'info':
        return 'bg-blue-600 text-white';
      case 'debug':
        return 'bg-purple-600 text-white';
      default:
        return 'bg-gray-600 text-white';
    }
  };

  return (
    <div className="h-full flex flex-col">
      {/* Controls */}
      <div className="bg-gray-800 border-b border-gray-700 p-4 space-y-3">
        {/* Search */}
        <div className="relative">
          <Search size={16} className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400" />
          <input
            type="text"
            placeholder="Search logs..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full pl-9 pr-3 py-2 bg-gray-700 border border-gray-600 rounded text-sm text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>

        {/* Level Filters */}
        <div className="flex items-center gap-2">
          <span className="text-sm text-gray-400">Levels:</span>
          <div className="flex flex-wrap gap-1">
            {LOG_LEVELS.map((level) => (
              <button
                key={level}
                onClick={() => toggleLevel(level)}
                className={`px-2 py-1 text-xs rounded uppercase transition-colors ${
                  levelFilter.size === 0 || levelFilter.has(level)
                    ? getLevelBadgeColor(level)
                    : 'bg-gray-700 text-gray-400 hover:bg-gray-600'
                }`}
              >
                {level}
              </button>
            ))}
            {levelFilter.size > 0 && (
              <button
                onClick={() => setLevelFilter(new Set())}
                className="px-2 py-1 text-xs bg-red-600 text-white rounded hover:bg-red-700"
              >
                Clear
              </button>
            )}
          </div>
        </div>

        <div className="text-xs text-gray-400">
          Showing {filteredLogs.length} of {run.logs.length} logs
        </div>
      </div>

      {/* Logs List */}
      <div className="flex-1 overflow-y-auto p-4">
        {filteredLogs.length === 0 ? (
          <div className="text-center text-gray-400 py-12">
            <Info size={48} className="mx-auto mb-4 opacity-50" />
            <p>No logs match the current filters</p>
          </div>
        ) : (
          <div className="space-y-2">
            {filteredLogs.map((log, index) => (
              <div
                key={index}
                className={`border-l-4 rounded-r-lg p-3 ${getLevelColor(log.level)}`}
              >
                <div className="flex items-start justify-between gap-2 mb-2">
                  <div className="flex items-center gap-2">
                    {getLevelIcon(log.level)}
                    <span
                      className={`px-2 py-0.5 text-xs rounded uppercase font-semibold ${getLevelBadgeColor(
                        log.level
                      )}`}
                    >
                      {log.level}
                    </span>
                  </div>
                  <span className="text-xs text-gray-400 font-mono">
                    {log.time}
                  </span>
                </div>

                <div className="text-sm text-white mb-2 flex items-start justify-between">
                  <span className="flex-1">{log.message}</span>
                  <CopyButton text={log.message} />
                </div>

                {log.error && (
                  <div className="mb-2 p-2 bg-red-950/50 rounded relative group">
                    <CopyButton 
                      text={log.error} 
                      className="absolute top-1 right-1 opacity-0 group-hover:opacity-100"
                    />
                    <div className="text-xs text-red-300 font-semibold mb-1">
                      Error:
                    </div>
                    <div className="text-xs text-red-200 font-mono whitespace-pre-wrap">
                      {log.error}
                    </div>
                  </div>
                )}

                {log.extra && Object.keys(log.extra).length > 0 && (
                  <details className="text-xs">
                    <summary className="cursor-pointer text-gray-400 hover:text-gray-300">
                      Additional Fields ({Object.keys(log.extra).length})
                    </summary>
                    <div className="mt-2">
                      <YamlView data={log.extra} maxHeight="200px" />
                    </div>
                  </details>
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

