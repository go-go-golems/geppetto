import { useState } from 'react';
import type { ParsedRun } from '../types';
import { ChevronDown, ChevronRight, Network, FileText, Activity, Copy, Maximize2, Minimize2 } from 'lucide-react';
import YamlView from './YamlView';
import CopyButton from './CopyButton';
import type { TabType } from './RunView';
import yaml from 'js-yaml';

interface Props {
  run: ParsedRun;
  onNavigate?: (tab: TabType, turnIndex?: number) => void;
}

export default function RawDataInspector({ run }: Props) {
  const [selectedTurn, setSelectedTurn] = useState<number>(0);
  const [expandedSections, setExpandedSections] = useState<Set<string>>(
    new Set(['http-request'])
  );

  if (run.raw.length === 0) {
    return (
      <div className="flex items-center justify-center h-full">
        <div className="text-center text-gray-400">
          <FileText size={48} className="mx-auto mb-4 opacity-50" />
          <p>No raw data available</p>
          <p className="text-sm mt-2">Run with --raw flag to capture HTTP/SSE data</p>
        </div>
      </div>
    );
  }

  const toggleSection = (section: string) => {
    const newSet = new Set(expandedSections);
    if (newSet.has(section)) {
      newSet.delete(section);
    } else {
      newSet.add(section);
    }
    setExpandedSections(newSet);
  };

  const artifact = run.raw[selectedTurn];

  return (
    <div className="h-full flex flex-col">
      {/* Turn Selector */}
      {run.raw.length > 1 && (
        <div className="bg-gray-800 border-b border-gray-700 p-4">
          <div className="flex items-center gap-2">
            <span className="text-sm text-gray-400">Turn:</span>
            <div className="flex gap-1">
              {run.raw.map((raw, index) => (
                <button
                  key={index}
                  onClick={() => setSelectedTurn(index)}
                  className={`px-3 py-1 text-sm rounded ${
                    selectedTurn === index
                      ? 'bg-blue-600 text-white'
                      : 'bg-gray-700 text-gray-300 hover:bg-gray-600'
                  }`}
                >
                  Turn {raw.turnIndex}
                </button>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* Content */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {/* HTTP Request */}
        {artifact?.httpRequest && (
          <Section
            title="HTTP Request"
            icon={<Network size={16} />}
            expanded={expandedSections.has('http-request')}
            onToggle={() => toggleSection('http-request')}
          >
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <span className="px-2 py-1 text-xs bg-blue-600 text-white rounded font-semibold">
                  {artifact.httpRequest.method}
                </span>
                <span className="text-sm text-gray-300 font-mono">
                  {artifact.httpRequest.url}
                </span>
              </div>

              {/* Input Turn Info */}
              {(artifact.inputTurnYaml || run.turns[artifact.inputTurnIndex]) && (
                <details className="border border-gray-700 rounded bg-gray-900/50 p-2">
                  <summary className="cursor-pointer text-xs text-gray-400 hover:text-gray-300 font-semibold">
                    📥 Input Turn: {run.turns[artifact.inputTurnIndex]?.label || `Turn ${artifact.inputTurnIndex}`}
                  </summary>
                  <div className="mt-2 text-xs text-gray-400">
                    {run.turns[artifact.inputTurnIndex] && (
                      <p className="mb-1">Turn ID: <span className="text-gray-300">{run.turns[artifact.inputTurnIndex].id}</span></p>
                    )}
                    <p className="mb-2">This HTTP request was made using the turn data shown below:</p>
                    {artifact.inputTurnYaml ? (
                      <div className="relative group">
                        <CopyButton 
                          text={artifact.inputTurnYaml} 
                          className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 z-10"
                        />
                        <pre className="p-3 bg-gray-900 rounded overflow-x-auto text-xs text-gray-200 max-h-80 overflow-y-auto border-l-4 border-green-500">
                          {artifact.inputTurnYaml}
                        </pre>
                      </div>
                    ) : run.turns[artifact.inputTurnIndex]?.rawYaml ? (
                      <div className="relative group">
                        <CopyButton 
                          text={run.turns[artifact.inputTurnIndex]?.rawYaml || ''} 
                          className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 z-10"
                        />
                        <pre className="p-3 bg-gray-900 rounded overflow-x-auto text-xs text-gray-200 max-h-80 overflow-y-auto">
                          {run.turns[artifact.inputTurnIndex]?.rawYaml}
                        </pre>
                      </div>
                    ) : (
                      <YamlView 
                        data={{ blocks: run.turns[artifact.inputTurnIndex]?.blocks }}
                        maxHeight="300px"
                      />
                    )}
                  </div>
                </details>
              )}

              <details>
                <summary className="cursor-pointer text-sm text-gray-400 hover:text-gray-300">
                  Headers
                </summary>
                <div className="mt-2">
                  <YamlView data={artifact.httpRequest.headers} maxHeight="200px" />
                </div>
              </details>

              <details open>
                <summary className="cursor-pointer text-sm text-gray-400 hover:text-gray-300">
                  Body
                </summary>
                <div className="mt-2">
                  <YamlView 
                    data={typeof artifact.httpRequest.body === 'string'
                      ? JSON.parse(artifact.httpRequest.body)
                      : artifact.httpRequest.body}
                    maxHeight="400px"
                  />
                </div>
              </details>
            </div>
          </Section>
        )}

        {/* HTTP Response */}
        {artifact?.httpResponse && (
          <Section
            title="HTTP Response"
            icon={<Network size={16} />}
            expanded={expandedSections.has('http-response')}
            onToggle={() => toggleSection('http-response')}
          >
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <span
                  className={`px-2 py-1 text-xs rounded font-semibold ${
                    artifact.httpResponse.status >= 200 && artifact.httpResponse.status < 300
                      ? 'bg-green-600 text-white'
                      : artifact.httpResponse.status >= 400
                      ? 'bg-red-600 text-white'
                      : 'bg-yellow-600 text-white'
                  }`}
                >
                  {artifact.httpResponse.status}
                </span>
              </div>

              <details>
                <summary className="cursor-pointer text-sm text-gray-400 hover:text-gray-300">
                  Headers
                </summary>
                <div className="mt-2">
                  <YamlView data={artifact.httpResponse.headers} maxHeight="200px" />
                </div>
              </details>

              <details open>
                <summary className="cursor-pointer text-sm text-gray-400 hover:text-gray-300">
                  Body
                </summary>
                <div className="mt-2">
                  <YamlView data={artifact.httpResponse.body} maxHeight="400px" />
                </div>
              </details>
            </div>
          </Section>
        )}

        {/* SSE Log */}
        {artifact?.sseLog && (
          <Section
            title="SSE Event Stream"
            icon={<Activity size={16} />}
            expanded={expandedSections.has('sse-log')}
            onToggle={() => toggleSection('sse-log')}
          >
            <SSEEventStream sseLog={artifact.sseLog} />
          </Section>
        )}

        {/* Provider Objects */}
        {artifact?.providerObjects && artifact.providerObjects.length > 0 && (
          <Section
            title={`Provider Objects (${artifact.providerObjects.length})`}
            icon={<FileText size={16} />}
            expanded={expandedSections.has('provider-objects')}
            onToggle={() => toggleSection('provider-objects')}
          >
            <div className="space-y-3">
              {artifact.providerObjects.map((obj, index) => (
                <div key={index} className="border border-gray-700 rounded-lg p-3">
                  <div className="flex items-center gap-2 mb-2">
                    <span className="px-2 py-1 text-xs bg-purple-600 text-white rounded">
                      #{obj.sequence}
                    </span>
                    <span className="text-sm text-gray-300">{obj.type}</span>
                  </div>
                  <details>
                    <summary className="cursor-pointer text-xs text-gray-400 hover:text-gray-300">
                      Show Data
                    </summary>
                    <div className="mt-2">
                      <YamlView data={obj.data} maxHeight="300px" />
                    </div>
                  </details>
                </div>
              ))}
            </div>
          </Section>
        )}
      </div>
    </div>
  );
}

interface SectionProps {
  title: string;
  icon: React.ReactNode;
  expanded: boolean;
  onToggle: () => void;
  children: React.ReactNode;
}

function Section({ title, icon, expanded, onToggle, children }: SectionProps) {
  return (
    <div className="bg-gray-800 border border-gray-700 rounded-lg overflow-hidden">
      <div
        className="flex items-center justify-between p-3 cursor-pointer hover:bg-gray-750"
        onClick={onToggle}
      >
        <div className="flex items-center gap-2">
          {expanded ? (
            <ChevronDown size={16} className="text-gray-400" />
          ) : (
            <ChevronRight size={16} className="text-gray-400" />
          )}
          {icon}
          <span className="font-semibold text-white">{title}</span>
        </div>
      </div>
      {expanded && <div className="p-3 border-t border-gray-700">{children}</div>}
    </div>
  );
}

interface SSEEventStreamProps {
  sseLog: string;
}

function SSEEventStream({ sseLog }: SSEEventStreamProps) {
  const events = parseSSELog(sseLog);
  const [expandedEvents, setExpandedEvents] = useState<Set<number>>(new Set());
  const [allExpanded, setAllExpanded] = useState(false);

  const copyAllSSEEvents = () => {
    const yamlString = yaml.dump(events.map(e => e.data), { indent: 2 });
    navigator.clipboard.writeText(yamlString);
  };

  const copySSEEventData = (eventData: any) => {
    const yamlString = yaml.dump(eventData, { indent: 2 });
    navigator.clipboard.writeText(yamlString);
  };

  const toggleAllSSEEvents = () => {
    if (allExpanded) {
      setExpandedEvents(new Set());
    } else {
      setExpandedEvents(new Set(events.map((_, idx) => idx)));
    }
    setAllExpanded(!allExpanded);
  };

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between mb-2">
        <span className="text-xs text-gray-400">{events.length} events</span>
        <div className="flex items-center gap-2">
          <button
            onClick={toggleAllSSEEvents}
            className="px-2 py-1 text-xs bg-gray-700 text-gray-300 hover:bg-gray-600 rounded flex items-center gap-1"
            title={allExpanded ? "Collapse all events" : "Expand all events"}
          >
            {allExpanded ? <Minimize2 size={12} /> : <Maximize2 size={12} />}
            {allExpanded ? 'Collapse All' : 'Expand All'}
          </button>
          <button
            onClick={copyAllSSEEvents}
            className="px-2 py-1 text-xs bg-gray-700 text-gray-300 hover:bg-gray-600 rounded flex items-center gap-1"
            title="Copy all SSE events as YAML"
          >
            <Copy size={12} />
            Copy All
          </button>
          <CopyButton text={sseLog} />
        </div>
      </div>
      {events.map((event, index) => (
        <details 
          key={index} 
          className="border border-gray-700 rounded-lg overflow-hidden"
          open={expandedEvents.has(index)}
          onToggle={(e) => {
            const newSet = new Set(expandedEvents);
            if ((e.target as HTMLDetailsElement).open) {
              newSet.add(index);
            } else {
              newSet.delete(index);
            }
            setExpandedEvents(newSet);
          }}
        >
          <summary className="cursor-pointer p-2 bg-gray-900 hover:bg-gray-850 flex items-center justify-between">
            <div className="flex items-center gap-2">
              <span className="px-2 py-0.5 text-xs bg-purple-600 text-white rounded">
                #{index}
              </span>
              <span className="text-xs font-mono text-gray-300">{event.eventType}</span>
              {event.data?.type && (
                <span className="text-xs text-gray-400">{event.data.type}</span>
              )}
            </div>
            <button
              onClick={(e) => {
                e.preventDefault();
                e.stopPropagation();
                copySSEEventData(event.data);
              }}
              className="p-1 text-gray-400 hover:text-white hover:bg-gray-700 rounded"
              title="Copy event data as YAML"
            >
              <Copy size={12} />
            </button>
          </summary>
          <div className="p-2">
            {event.data ? (
              <YamlView data={event.data} maxHeight="300px" />
            ) : (
              <pre className="text-xs text-gray-400 p-2">No data</pre>
            )}
          </div>
        </details>
      ))}
    </div>
  );
}

interface SSEEvent {
  eventType: string;
  data: any;
}

function parseSSELog(sseLog: string): SSEEvent[] {
  const lines = sseLog.split('\n');
  const events: SSEEvent[] = [];
  let currentEvent: string | null = null;

  for (const line of lines) {
    if (line.startsWith('#')) continue; // Skip comments
    if (line.startsWith('event: ')) {
      currentEvent = line.substring(7).trim();
    } else if (line.trim() && currentEvent) {
      try {
        const data = JSON.parse(line);
        events.push({ eventType: currentEvent, data });
        currentEvent = null;
      } catch (e) {
        // Skip invalid JSON
      }
    }
  }

  return events;
}

