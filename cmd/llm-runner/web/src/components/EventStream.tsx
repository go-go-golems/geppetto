import { useState, useMemo } from 'react';
import type { ParsedRun } from '../types';
import { format } from 'date-fns';
import { Search, Filter } from 'lucide-react';
import YamlView from './YamlView';
import type { TabType } from './RunView';

interface Props {
  run: ParsedRun;
  onNavigate?: (tab: TabType, turnIndex?: number) => void;
}

export default function EventStream({ run }: Props) {
  const [selectedRun, setSelectedRun] = useState<number>(0);
  const [searchTerm, setSearchTerm] = useState('');
  const [eventTypeFilter, setEventTypeFilter] = useState<Set<string>>(new Set());

  const allEventTypes = useMemo(() => {
    const types = new Set<string>();
    run.events.forEach((eventSet) => {
      eventSet.forEach((event) => types.add(event.type));
    });
    return Array.from(types).sort();
  }, [run.events]);

  const toggleEventType = (type: string) => {
    const newSet = new Set(eventTypeFilter);
    if (newSet.has(type)) {
      newSet.delete(type);
    } else {
      newSet.add(type);
    }
    setEventTypeFilter(newSet);
  };

  const filteredEvents = useMemo(() => {
    if (!run.events[selectedRun]) return [];
    
    return run.events[selectedRun].filter((event) => {
      if (eventTypeFilter.size > 0 && !eventTypeFilter.has(event.type)) {
        return false;
      }
      if (searchTerm) {
        const searchLower = searchTerm.toLowerCase();
        return (
          event.type.toLowerCase().includes(searchLower) ||
          JSON.stringify(event.data).toLowerCase().includes(searchLower)
        );
      }
      return true;
    });
  }, [run.events, selectedRun, eventTypeFilter, searchTerm]);

  const getEventColor = (type: string) => {
    switch (type) {
      case 'start':
        return 'bg-blue-900/30 border-blue-500 text-blue-200';
      case 'final':
        return 'bg-green-900/30 border-green-500 text-green-200';
      case 'error':
        return 'bg-red-900/30 border-red-500 text-red-200';
      case 'partial':
        return 'bg-yellow-900/30 border-yellow-500 text-yellow-200';
      case 'info':
        return 'bg-gray-900/30 border-gray-500 text-gray-200';
      default:
        return 'bg-purple-900/30 border-purple-500 text-purple-200';
    }
  };

  return (
    <div className="h-full flex flex-col">
      {/* Controls */}
      <div className="bg-gray-800 border-b border-gray-700 p-4 space-y-3">
        {/* Run Selector */}
        {run.events.length > 1 && (
          <div className="flex items-center gap-2">
            <span className="text-sm text-gray-400">Event Set:</span>
            <div className="flex gap-1">
              {run.events.map((_, index) => (
                <button
                  key={index}
                  onClick={() => setSelectedRun(index)}
                  className={`px-3 py-1 text-sm rounded ${
                    selectedRun === index
                      ? 'bg-blue-600 text-white'
                      : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
                  }`}
                >
                  Run {index + 1} ({run.events[index].length})
                </button>
              ))}
            </div>
          </div>
        )}

        {/* Search */}
        <div className="flex items-center gap-2">
          <div className="flex-1 relative">
            <Search size={16} className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400" />
            <input
              type="text"
              placeholder="Search events..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="w-full pl-9 pr-3 py-2 bg-gray-700 border border-gray-600 rounded text-sm text-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
        </div>

        {/* Filters */}
        <div className="flex items-start gap-2">
          <Filter size={16} className="text-gray-400 mt-1" />
          <div className="flex flex-wrap gap-1">
            {allEventTypes.map((type) => (
              <button
                key={type}
                onClick={() => toggleEventType(type)}
                className={`px-2 py-1 text-xs rounded transition-colors ${
                  eventTypeFilter.size === 0 || eventTypeFilter.has(type)
                    ? 'bg-blue-600 text-white'
                    : 'bg-gray-700 text-gray-400 hover:bg-gray-600'
                }`}
              >
                {type}
              </button>
            ))}
            {eventTypeFilter.size > 0 && (
              <button
                onClick={() => setEventTypeFilter(new Set())}
                className="px-2 py-1 text-xs bg-red-600 text-white rounded hover:bg-red-700"
              >
                Clear
              </button>
            )}
          </div>
        </div>

        <div className="text-xs text-gray-400">
          Showing {filteredEvents.length} of {run.events[selectedRun]?.length || 0} events
        </div>
      </div>

      {/* Events List */}
      <div className="flex-1 overflow-y-auto p-4 space-y-2">
        {filteredEvents.map((event, index) => (
          <div
            key={index}
            className={`border-l-4 rounded-r-lg p-3 ${getEventColor(event.type)}`}
          >
            <div className="flex items-start justify-between gap-2 mb-2">
              <div className="flex items-center gap-2">
                <span className="px-2 py-0.5 text-xs bg-black/30 rounded font-mono">
                  {event.type}
                </span>
                {event.meta?.message_id && (
                  <span className="text-xs text-gray-400">
                    {event.meta.message_id.slice(0, 8)}
                  </span>
                )}
              </div>
              <span className="text-xs text-gray-400">
                {format(event.timestamp, 'HH:mm:ss.SSS')}
              </span>
            </div>

            {event.meta && (
              <div className="mb-2 flex flex-wrap gap-2">
                {event.meta.model && (
                  <span className="text-xs text-gray-400">
                    Model: {event.meta.model}
                  </span>
                )}
                {event.meta.turn_id && (
                  <span className="text-xs text-gray-400">
                    Turn: {event.meta.turn_id}
                  </span>
                )}
              </div>
            )}

            <details className="text-xs">
              <summary className="cursor-pointer text-gray-300 hover:text-white">
                Event Data
              </summary>
              <div className="mt-2">
                <YamlView data={event.data} maxHeight="300px" />
              </div>
            </details>
          </div>
        ))}
      </div>
    </div>
  );
}

